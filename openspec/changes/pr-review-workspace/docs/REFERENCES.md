# References index — pr-review-workspace

This change reshapes the in-flight `pr-review-mode` work into a 3-window review
workspace. The earlier `pr-review-mode/docs/*` files already captured the real
GitHub data shapes against live public PRs; this folder ports the parts that
survive and records what changed. Read the cited source doc for the full blob;
the workspace docs only restate fields and commands that the implementer needs.

## PROJECT-AGNOSTIC MANDATE (read this first)

Nothing in this change may be hardcoded to a specific project, path, owner, or
repo. The old `pr-review-mode` docs reference `/home/user/anc-review` and
`anchorlabsinc/anchorage` as *examples* — those are FORBIDDEN as literals in
code. Everything must be derived at RUNTIME:

- **Repo identity** (`owner`, `repo`) — from the launched checkout's remotes
  (verified against the CLI target) and the PR data, never a constant.
- **Checkout path** — from cwd or the `-p`/`--path` flag, never a constant.
- **base/head OIDs and repo URLs** — from the PR read query at runtime.
- **Review refs and persisted viewed-state** are keyed by the runtime-resolved
  repo identity, so two checkouts of different repos never collide.

When a `pr-review-mode` doc shows `cli/cli`, `OWNER/REPO`, or an absolute path,
treat it as a placeholder for a runtime value.

## Where each data shape lives

| Data shape | Source doc | Used by workspace doc |
|---|---|---|
| Combined GraphQL read query (PR meta, threads, reviewers, issue comments) | `pr-review-mode/docs/github-api.md` §1 | `data-shapes.md` (c) |
| `pulls/{n}/files` patch shape (filename/status/patch/additions/deletions) | `pr-review-mode/docs/github-api.md` §2 | `data-shapes.md` (b) |
| Comment line+side mapping rules (anchor/diffSide/start_line) | `pr-review-mode/docs/github-api.md` §3, §4 | carried forward (see below) |
| Captured JSON for threads / reply chains / reviewers | `pr-review-mode/docs/github-api.md` §1 captured blocks | cited, not duplicated |
| gh-dash template vars + `prs:` keybinding handoff | `pr-review-mode/docs/gh-dash-integration.md` | `review-fetch-command.md` (repo identity), tasks P8.1 |
| Render model (`DisplayRow`, interleaved comment rows) | `pr-review-mode/docs/gh-review-reference.md` | carried forward (selection surface) |
| AppState persistence precedent (`GithubPullRequests`) | `pkg/config/app_config.go:697` | `viewed-state-schema.md` |

## SUPERSEDED by this change

- **Transport: `net/http` + go-gh token → `gh` CLI.** `pr-review-mode/docs/github-api.md`
  §"Transport decision" mandates net/http; design DW3 supersedes it — all GitHub
  I/O shells `gh` (`gh api`, `gh api graphql`, `gh pr …`, `gh run view`). The
  already-shipped `github_review.go` already uses `gh` (see `FetchPRReviewData`).
  The endpoints/fields in the old doc are still correct; only the caller changed.
- **Bespoke inline diff presenter → delta pager preview + row-map selection
  surface.** `gh-review-reference.md` argues you must own the renderer. DW2 splits
  this: read-preview goes through the user's pager (delta) as display-only; the
  existing custom row-map context (`pr_review_diff.go`) remains ONLY as the
  line-addressable selection surface for commenting.
- **Session-only reviewed state → persisted blob-OID viewed-state.** `pr-review-mode`
  D8 kept "viewed" in memory; DW4 persists it in `AppState` keyed by blob OID with
  content invalidation. See `viewed-state-schema.md`.
- **Single "files" window → 3-window workspace.** The old single Files view becomes
  `prList [1]`, `prContent [2]` (Overview · Files Changed), `prActivity [3]`
  (Conversation · Checks · Commits). See design DW1/DW5–DW8.

## CARRIED FORWARD (unchanged)

- **Glow rendering** of comment/review bodies (markdown → terminal). Same gate as
  `pr-review-mode`.
- **Add-comment line+side mapping** — the `path/side/line/start_line` rules and
  the live-`headRefOid` `commit_id` requirement from `github-api.md` §3–§4 are
  unchanged. v1 stays RIGHT-side only.
- **Models** in `pkg/commands/models/github.go` (`ReviewThread`, `ReviewComment`,
  `Reviewer`, `IssueComment`, `GithubPullRequestFile`) and the
  `PullRequestReviewData` aggregate in `github_review.go`.

## New data shapes this change introduces (no prior source)

- The **local-ref fetch** git command — `review-fetch-command.md` (new).
- **check-runs + workflow jobs** REST shapes — `data-shapes.md` (d); documented
  from the GitHub REST API, not captured against a live PR (gap, see that file).
- **base OID + base/head repo URLs** added to the read query for fork-aware
  fetch — `data-shapes.md` (a); not in the existing `reviewDataQuery`.
