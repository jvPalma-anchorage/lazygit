# Add-comment-by-range flow

Self-contained reference for mapping a line-range selection in the review diff to
a GitHub review comment. **Verdict: feasible; the patch infrastructure already
hands you ~90%.** The fear-inducing part (legacy unified-diff `position` offsets)
is obsolete — GitHub takes file line numbers + side, which the patch package
already computes.

## Key modern fact

`POST /repos/{o}/{r}/pulls/{n}/comments` takes `line` / `side` / `start_line` /
`start_side` (file line numbers), NOT the legacy diff `position`. A review does
NOT need to be created first — this endpoint posts immediately (GitHub wraps it in
a single-comment review internally).

## Selection (no patch_exploring.State reuse)

The review buffer interleaves non-selectable COMMENT rows, so `patch_exploring.
State` cannot drive selection (see inline-comments-rendering.md). Drive gocui
directly like `patch_explorer_context.go:107-110`:

```go
view.SetRangeSelectStart(start)
view.SetCursorY(end - origin)
```

On every cursor move, SNAP to the next/prev row whose `rowKind == rowDiff` (skip
COMMENT/HEADER rows).

## Mapping selected rows → (path, side, line, start_line)

Inputs: the selected `[startRow, endRow]` DIFF rows, the file path (from the file
list), and the live head SHA (`headRefOid` from the read query, github-api.md).

```go
// rowPatchIdx maps a DIFF row to its index in patch.Lines()
l1 := p.LineNumberOfLine(rowPatchIdx[startRow]) // new-file line (RIGHT)
l2 := p.LineNumberOfLine(rowPatchIdx[endRow])
if l1 > l2 { l1, l2 = l2, l1 }                  // start must precede line
```

- `side = "RIGHT"`, `start_side = "RIGHT"` (v1 RIGHT-only).
- `line = l2` (range END maps to GitHub's `line`).
- `start_line = l1` — but if `l1 == l2` (single line), OMIT `start_line`/
  `start_side` entirely (do NOT set `start_line == line` → 422).
- v1 guard: if either endpoint resolves to a DELETION (LEFT) line, reject with a
  toast ("v1 supports commenting on added/context lines only"). This avoids
  mixed-`start_side`/`side` 422s and the missing old-file-line helper.

> `LineNumberOfLine` returns a NEW-file number even for DELETION lines (it counts
> ADDITION+CONTEXT), so it is ONLY valid for RIGHT-side. The RIGHT-only guard
> keeps this correct without a new helper.

## The POST (mirror github.go net/http)

```sh
# multi-line
gh api --method POST /repos/OWNER/REPO/pulls/N/comments \
  -f body='nit: extract this' \
  -f commit_id=HEAD_SHA \
  -f path='pkg/gui/gh.go' \
  -F line=42 -f side=RIGHT \
  -F start_line=38 -f start_side=RIGHT
# single-line: drop start_line/start_side
```

In-code (`pkg/commands/git_commands/github_review.go`): net/http POST with
`Authorization: token <GetAuthToken(host)>`, `Accept: application/vnd.github+json`,
JSON body. Same token/transport as the read path — no `gh` dependency.

```go
func (self *GitHubCommands) AddReviewComment(
    owner, repo string, number int, headOid, path, body string,
    startLine, line int, token string) error
```

Build the body with `commit_id=headOid`, `path`, `side="RIGHT"`, `line`; include
`start_line`/`start_side` only when `startLine != line`.

## After posting

On 2xx, re-fetch the review data (read query) so the new thread appears inline.
On 422, surface the GitHub error message via toast (most common cause: stale
`commit_id`, or `line` not in the diff).

## Critical correctness rules

- `commit_id` MUST be the live `headRefOid` from the API, never local HEAD. A
  stale SHA silently produces an "outdated" comment, not an error.
- `line`/`start_line` MUST be real diff-body lines — exclude HUNK_HEADER /
  PATCH_HEADER rows (selection snapping already enforces this).
- `start_line` strictly precedes `line` on the same side; single-line omits
  `start_line`.
- The diff under review must be the PR's base...head (the REST `patch` is, by
  construction), so the line numbers match what GitHub expects.

## Deferred (non-goals)

- Reply to an existing thread:
  `POST /pulls/{n}/comments/{comment_id}/replies` with `body` (uses a comment
  `databaseId`, which we fetch). UI deferred.
- Resolve / unresolve: GraphQL `resolveReviewThread`/`unresolveReviewThread`
  (needs thread node `id`, which we fetch). Deferred.
- Batched pending review (`POST /pulls/{n}/reviews` with `comments[]`,
  submit as APPROVE/REQUEST_CHANGES). Deferred — v1 posts standalone comments,
  no review lifecycle to manage.
- LEFT-side / mixed-side ranges (need `Patch.OldLineNumberOfLine(idx)` counting
  DELETION+CONTEXT + 422 guards). Deferred.
