# GitHub data shapes the workspace consumes

Every `gh` command and the exact JSON fields read from it. All access is via the
`gh` CLI (DW3) through the `oscommands` argv builder + `DontLog().RunWithOutputs()`,
mirroring the existing `github_review.go`. `{o}`/`{r}`/`{n}` are the
runtime-resolved owner / repo / PR number — NEVER hardcoded (see REFERENCES.md
project-agnostic mandate). Captured JSON lives in
`pr-review-mode/docs/github-api.md`; cite it rather than re-pasting blobs.

## (a) PR overview metadata + fork-aware fetch inputs

The existing `reviewDataQuery` (`github_review.go`) returns identity + threads but
**lacks** the base OID and the head/base repo URLs needed for the local-ref fetch
(`review-fetch-command.md`). Extend it (tasks 2.2). Use `gh pr view --json` for the
overview fields and `gh api graphql` for the fork-aware OIDs/URLs.

```sh
gh pr view {n} --repo {o}/{r} --json \
  number,title,baseRefName,headRefName,author,createdAt,updatedAt,\
labels,assignees,milestone,state,isDraft
```

Overview tab consumes (DW6): `title`, `number`, `baseRefName` ← `headRefName`
(rendered `base ← head`), `author.login`, `createdAt`/`updatedAt`, `state`,
`labels[].{name,color}` (color is a 6-digit hex → map to ANSI), `assignees[].login`,
`milestone.title`.

Fork-aware fetch inputs (add to the GraphQL query — REQUIRED, head often lives in
a fork so head-repo URL ≠ base-repo URL):

```graphql
pullRequest(number:$number){
  headRefOid
  baseRefOid                                  # base OID — currently missing
  baseRepository{ url nameWithOwner }         # base repo clone URL
  headRepository{ url nameWithOwner }          # head repo clone URL (may be a fork)
  headRepositoryOwner{ login }
}
```

`url` is the HTTPS clone URL; pass it verbatim to `git fetch <url> <oid>:…`. Do
not reconstruct URLs from owner/repo strings — read them from the API.

## (b) Changed files

```sh
gh api --paginate repos/{o}/{r}/pulls/{n}/files \
  --jq '.[]|{filename,status,previous_filename,additions,deletions,patch}'
```

Per-file fields consumed (see `github-api.md` §2 for the captured example):

| field | meaning |
|---|---|
| `filename` | new path; the tree node key |
| `status` | `added` / `modified` / `removed` / `renamed` / `copied` / `changed` |
| `previous_filename` | present only on `renamed`/`copied`; old path |
| `patch` | unified diff (`@@` header + `+`/`-`/space body); feeds `patch.Parse` |
| `additions`/`deletions` | per-file counts for the tree decoration |

**Binary detection**: a binary or too-large file has **no `patch` field** (GitHub
omits it; the API may instead carry a `Binary file …` note). Treat absent `patch`
as binary/unrenderable → tree shows the file but the diff pane shows a
"binary / no preview" placeholder, and viewed-state keys on `fileOid` not
`patchHash` (see `viewed-state-schema.md`). The PR line numbers in `patch` EQUAL
the `reviewThreads.line` values, so the same numbers anchor comments.

## (c) Review threads + reviewers + issue comments

All three come from the combined GraphQL read query — **do not re-derive**, see
`pr-review-mode/docs/github-api.md` §1 (query + field semantics + captured JSON
for cli/cli #9000 and #8995). Summary of what the Conversation tab (DW7) reads:

- **reviewThreads** nodes: `id, isResolved, isOutdated, path, line, startLine,
  originalLine, diffSide{LEFT,RIGHT}`, plus `comments.nodes{ databaseId, author.login,
  body, createdAt, diffHunk, replyTo.databaseId }`. A thread's `comments` IS the
  reply chain, root-first (`replyTo` null on root). Modeled by `models.ReviewThread`.
- **reviewers/states**: `latestReviews.nodes{ author.login, state, submittedAt }`
  (one-per-author for the badge) + `reviews.nodes` (bodies/history). `state` ∈
  `{APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED, PENDING}`.
  `reviewRequests.nodes.requestedReviewer` may be **null** (already reviewed) or a
  Team (split via `__typename`).
- **issue/PR-body comments**: `comments.nodes{ author.login, body, createdAt }`
  (the issue-level conversation, ask 3.4.1). Modeled by `models.IssueComment`.

Bodies render through glow (carried forward).

## (d) Check-runs + workflow jobs

Two `gh` shapes normalized into one tree node model (DW8). Documented from the
GitHub REST API — NOT captured against a live PR (gap).

```sh
# check runs for the PR's live head SHA (headRefOid from (a))
gh api repos/{o}/{r}/commits/{headSha}/check-runs --paginate \
  --jq '.check_runs[]|{name,status,conclusion,started_at,completed_at,details_url}'
# workflow run jobs (run id from the check-run/check-suite linkage)
gh api repos/{o}/{r}/actions/runs/{runId}/jobs --paginate \
  --jq '.jobs[]|{name,status,conclusion,started_at,completed_at,steps}'
```

Fields consumed per node:

- `name` — node label.
- `status` ∈ `queued | in_progress | completed`.
- `conclusion` (set when `status == completed`) ∈ `success | failure | neutral |
  cancelled | skipped | timed_out | action_required | stale | null`.
- `started_at` / `completed_at` — for the "last-updated desc" secondary sort.
- `steps[]` (jobs only): `{name, status, conclusion, number, started_at,
  completed_at}` → child nodes under the job.

**Importance enum** (primary sort, highest first; skipped sinks to the bottom):

```
failure > cancelled > interrupted/timed_out > pending/in_progress > success/neutral > skipped
```

Map raw values: `failure`+`action_required`+`timed_out`→failure/interrupted band;
`cancelled`+`stale`→cancelled; `queued`+`in_progress`→pending; `success`+`neutral`
→success; `skipped`→skipped. Secondary sort: `completed_at`/`started_at` desc.
Leaf `enter` renders logs via `gh run view <runId> --log` (`--log-failed` for
failures) with `FORCE_COLOR=1`.

## (e) Local `git diff --name-status` status letters

The Files-Changed tree (tasks 2.5) is populated from a **local** diff over the
fetched refs, not the API, so directory aggregation works (DW2 fixes #1):

```sh
git diff --name-status <base>...<head>
```

| letter | meaning | maps to API `status` |
|---|---|---|
| `M` | modified | `modified` |
| `A` | added | `added` |
| `D` | deleted | `removed` |
| `R<nnn>` | renamed (similarity %, two paths: old → new) | `renamed` |
| `C<nnn>` | copied (two paths) | `copied` |
| `T` | type change (e.g. file ↔ symlink) | `changed` |

Note `ShowFileDiffCmdObj` forces `--no-renames` (`working_tree.go:451`), so the
preview path shows renames as add+delete; reconcile against the API `status` /
`previous_filename` when keying viewed-state (renames need both paths).
