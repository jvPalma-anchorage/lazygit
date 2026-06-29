# GitHub API for PR review mode

Self-contained reference for every data call PR review mode makes: the combined
read query, the diff fetch, and the add-comment write. All six review-mode data
needs are satisfiable. JSON below was captured from real public PRs (cli/cli
#9000 and #8995).

## Transport decision (design.md D3)

Reuse the in-repo `net/http` + go-gh token path, NOT `gh`:
- `github.go:141 GetAuthToken(host)` resolves the host-scoped token.
- POST to `graphQLEndpoint(host)` (github.go:339) mirroring `fetchRecentPRsAux`
  (github.go:198).
- REST calls use the same `Authorization: token <token>` +
  `Accept: application/vnd.github+json` headers.

The `gh api`/`gh api graphql` forms shown below are the **manual-test / spec
contract**; the shipped code uses the net/http path so there is no new dependency
and no auth-account mismatch.

## 1. Combined read query (needs: threads, global comments, reviewers)

One query against `repository.pullRequest` returns everything except the diff.

```graphql
query($owner:String!,$repo:String!,$number:Int!){
  repository(owner:$owner,name:$repo){
    pullRequest(number:$number){
      id number title state isDraft baseRefName headRefName headRefOid
      author{ login }
      comments(first:50){ pageInfo{ hasNextPage endCursor }
        nodes{ author{ login } body createdAt url } }
      reviewRequests(first:50){ nodes{
        requestedReviewer{ __typename ... on User{ login } ... on Team{ name } } } }
      latestReviews(first:50){ nodes{ author{ login } state submittedAt } }
      reviews(first:50){ nodes{ author{ login } state submittedAt body } }
      reviewThreads(first:50){ pageInfo{ hasNextPage endCursor }
        nodes{
          id isResolved isOutdated isCollapsed path line startLine originalLine diffSide
          comments(first:50){ pageInfo{ hasNextPage endCursor }
            nodes{ databaseId author{ login } body createdAt diffHunk
                   originalLine line position outdated state replyTo{ databaseId } } } } } } } }
```

Manual run:

```sh
gh api graphql -F owner=cli -F repo=cli -F number=9000 -f query=@query.graphql
```

### Field semantics

- `headRefOid` — live head SHA; REQUIRED as `commit_id` for add-comment.
- `diffSide` enum `{LEFT(base/old), RIGHT(head/new)}`.
- `line`/`startLine` — file line numbers on `diffSide`. Single-line:
  `startLine == line` or `startLine == null`.
- `originalLine` — line at the commit when the comment was made (use when
  `isOutdated`).
- `replyTo.databaseId` — links a reply to its root (null on the root). A thread's
  `comments.nodes` IS the reply chain, ordered root-first.
- `reviewRequests.requestedReviewer` — can be **null** (already reviewed) or a
  Team (split via `__typename`).
- `reviews.state` ∈ `{APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED, PENDING}`;
  `latestReviews` dedupes to one-per-author (use it for the badge,
  `reviews` for bodies/history).

### Captured: review thread (cli/cli #9000) — line anchoring + resolution

```json
{"id":"PRRT_kwDODKw3uc48Rk4m","isResolved":true,"isOutdated":false,
 "path":"pkg/cmd/attestation/verify/verify.go","line":130,"startLine":130,
 "originalLine":130,"originalStartLine":null,"diffSide":"RIGHT",
 "comments":{"totalCount":1,"nodes":[
   {"author":{"login":"williammartin"},
    "body":"Could I interest you in the following tests...",
    "createdAt":"2024-04-29T14:06:54Z",
    "diffHunk":"@@ -127,6 +127,7 @@ func NewVerifyCmd(...)\n \n \t// general flags\n \tverifyCmd.Flags().StringVarP(&opts.BundlePath,...)\n+\tcmdutil.DisableAuthCheckFlag(verifyCmd.Flags().Lookup(\"bundle\"))",
    "line":130,"position":4,"state":"SUBMITTED"}]}}
```

### Captured: reply chain (cli/cli #8995)

A thread's `comments` node IS the chain — root first (`replyTo=null`), each reply
carries `replyTo.databaseId == root.databaseId`. No separate query.

```
THREAD path=pkg/cmd/attestation/verify/verify.go line=97 side=RIGHT resolved=false count=2
  dbId=1577888327 replyTo=None       author=steiza        body='I thought about using ... authHelp() here'
  dbId=1577925765 replyTo=1577888327 author=williammartin body="Realistically if we go down this route..."
```

### Captured: reviewers + states (cli/cli #9000)

```json
"reviewRequests":{"totalCount":1,"nodes":[{"requestedReviewer":null}]},
"latestReviews":{"nodes":[
  {"author":{"login":"steiza"},"state":"APPROVED","submittedAt":"2024-04-25T19:55:42Z"},
  {"author":{"login":"williammartin"},"state":"APPROVED","submittedAt":"2024-04-29T16:07:45Z"}]},
"reviews":{"nodes":[
  {"author":{"login":"steiza"},"state":"APPROVED","body":"This looks like a workable alternative..."},
  {"author":{"login":"williammartin"},"state":"COMMENTED","body":"One test addition I think would be useful"}]}
```

## 2. Changed files + diff (needs: file list, diff render)

REST per-file `patch` (chosen — integrates with per-file rendering and the patch
parser):

```sh
gh api --paginate repos/cli/cli/pulls/9000/files \
  --jq '.[]|{filename,status,additions,deletions,patch}'
# => {"filename":"pkg/cmdutil/auth_check.go","status":"modified",
#     "additions":27,"deletions":2,"patch":"@@ -1,16 +1,29 @@ ..."}
```

The `patch` field is a unified diff (`@@` header + `+`/`-`/space body), so
`pkg/commands/patch.Parse` consumes it directly. The PR file line numbers in the
diff EQUAL the `line` values from `reviewThreads`, so the same numbers anchor
comments to rendered lines.

Alternative: `gh pr diff 9000 --repo cli/cli` returns the whole unified diff in
one stream — simpler to pipe through a pager, but worse for per-file inline
anchoring. Not chosen.

## 3. Comment → rendered-line mapping rules

1. Match `thread.path` to the file being rendered.
2. Pick the column from `diffSide`: RIGHT → new/head line numbers, LEFT → old/base.
   **v1 anchors RIGHT only** (design.md D4); LEFT threads go to the outdated/other
   section.
3. Anchor line = `thread.line` (range end); range start = `thread.startLine`
   (== line when single-line; may be null → treat as line).
4. If `thread.line == null` but `isOutdated`/`originalLine` set → no live anchor;
   render under "Outdated" using `comments[0].diffHunk` verbatim (its LAST line is
   the commented line).
5. Multi-line range highlight: shade `startLine..line` on `diffSide`.
6. Badge = `isResolved ? RESOLVED : UNRESOLVED`; author = `comments[0].author.login`.

> Prefer `line`+`side`; `position`/`originalPosition` (diff-relative offsets) are
> only for the legacy REST position API and are not used.

## 4. Add review comment by range (need: add comment) — write

GitHub's modern API takes **file line + side**, not the legacy diff position.
NO prior review is needed: `POST /pulls/{n}/comments` posts immediately (GitHub
auto-wraps it in a single-comment review). v1 is RIGHT-side only.

```sh
# multi-line (new-file lines 38-42)
gh api --method POST /repos/OWNER/REPO/pulls/N/comments \
  -f body='nit: extract this' \
  -f commit_id=HEAD_SHA \
  -f path='pkg/gui/gh.go' \
  -F line=42 -f side=RIGHT \
  -F start_line=38 -f start_side=RIGHT
# single-line: drop start_line/start_side, keep line+side
```

Rules: `commit_id` MUST be the live `headRefOid` (a stale SHA silently makes the
comment "outdated"). `start_line` MUST strictly precede `line` on the same side;
single-line MUST omit `start_line`. `line`/`start_line` MUST be real diff-body
lines (exclude hunk/file headers) or GitHub 422s.

In-code: net/http POST mirroring github.go with `Authorization: token <token>`
and a JSON body.

## 5. Deferred mutations (non-goals, documented for completeness)

- **Reply to a thread:** `POST /pulls/{n}/comments/{comment_id}/replies` with
  `body` only (`comment_id` = a comment `databaseId`; we fetch it so it is
  plumbed). UI deferred.
- **Resolve / unresolve:** GraphQL only — `resolveReviewThread(input:{threadId})` /
  `unresolveReviewThread`. Needs the thread node `id` (we fetch it). Deferred.
- **Batched pending review:** `POST /pulls/{n}/reviews` with a `comments[]` array
  + `event`. Deferred.

## Pagination caveat

`reviewThreads`, `thread.comments`, `comments`, and `pulls/{n}/files` are all
paginated (`hasNextPage` was true even at `first:1`). v1 caps at `first:50` and
surfaces a truncation note; endCursor looping is a non-goal.
