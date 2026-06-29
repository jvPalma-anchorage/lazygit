# gh-review as the reference blueprint

`~/projects/gh-review` (Rust, `cargo`, ~64 files) is a working, shipped TUI that
implements **this exact feature**: `gh-review <OWNER/REPO> <PR_NUMBER>`, launched
from gh-dash, with unified + side-by-side diffs, inline line-anchored comment
threads (resolved state + author + replies), global comments, reviewers, add /
reply / resolve, suggestions, and approve. It is the proof that `pr-review-mode`
is feasible and the template for the hard parts. This doc maps its modules to the
lazygit equivalents the tasks build.

It does **not** use git-delta. It has its own diff engine + tree-sitter syntax
highlighting. The "delta side-by-side" in its README is a *separate gh-dash
keybinding*, not gh-review's internal rendering. That settles the lazygit
question: to get consistent, high-quality, *interactive* (selectable + inline
comments) diffs, you own the renderer — you cannot pipe through delta-the-pager
and keep selection. (Lazygit's existing add/remove/context colorization from
`pkg/commands/patch/format.go` is the v1 styling baseline; per-token syntax
highlighting like gh-review's tree-sitter pass is a future enhancement, not v1.)

## Module map (gh-review → lazygit)

| gh-review (Rust) | What it does | lazygit equivalent (this change) |
|---|---|---|
| `src/cli.rs` | parse `OWNER/REPO` / URL / number | D1: 2nd positional in `entry_point.go` + `ReviewTarget` |
| `src/gh.rs` (821 LOC) | ALL GitHub I/O via `gh api` (REST+GraphQL) | D3: `github_review.go` via net/http + go-gh token (reuse existing transport) |
| `src/types.rs` | domain model | new models in `pkg/commands/models` (task 3.1) |
| `src/diff/parser.rs` | parse unified diff → hunks | reuse `pkg/commands/patch.Parse` |
| `src/diff/model.rs` (452 LOC) | flat `Vec<DisplayRow>` w/ interleaved comment rows | D4: `pr_review_diff.go` `content`+`rowKind`+`rowPatchIdx` slices |
| `src/diff/renderer.rs` (325 LOC) | unified + side-by-side draw | D4/D9: the presenter + `DiffMode` |
| `src/components/diff_view/comment_block.rs` | render a thread block | D4: thread block (author + badge + glow body) |
| `src/components/file_picker.rs` | **flat** fuzzy file list | **rejected** — D7 uses lazygit's file *tree* |
| `src/components/description_panel.rs` | `[d]` description | D10 |
| `src/highlight.rs` (arborium/tree-sitter) | syntax highlight | out of v1 (use lazygit's diff colorization) |
| (none — gh-review has no "viewed" state) | — | **D8 mark-reviewed is lazygit's addition** |

## The render model to copy (D4)

gh-review's `DisplayRow` enum is the key abstraction — one flat list, comment rows
interleaved at the anchor line:

```
enum DisplayRow { FileHeader, HunkHeader, DiffLine{..}, CommentHeader{..}, CommentBodyLine{..}, CommentFooter{..} }
```

Build algorithm (`model.rs` ~line 230): walk hunk lines; for each `DiffLine` look
up threads anchored to that `(line, side)`; group by thread id; push
`CommentHeader/BodyLine/Footer` rows immediately after the diff line. Orphan /
outdated threads (anchor line absent) are appended at the end. The viewport scrolls
and selects over row indices — exactly D4's `rowKind`/`rowPatchIdx` parallel slices.
Note gh-review carries **both** `old_lineno` and `new_lineno` per `DiffLine`, which
is what lets it anchor LEFT-side (deletion) threads — mirror that for D4 both-side
display.

## The exact GitHub calls (validated against gh-review's `src/gh.rs`)

Use these shapes (lazygit runs them over net/http+token, not the `gh` binary, per
D3 — but the endpoints/fields are identical):

- **Current user:** `GET /user` → `.login`
- **Review threads (GraphQL):** `repository.pullRequest.reviewThreads(first:N, after:$cursor)` → `isResolved, isOutdated, path, line, startLine, diffSide, comments(first:N){ author.login, body, createdAt, diffHunk, originalLine, databaseId, replyTo{databaseId} }`
- **Changed files:** `GET /repos/{o}/{r}/pulls/{n}/files --paginate` → `filename, status, patch`
- **Review comments (REST, alt to threads):** `GET /repos/{o}/{r}/pulls/{n}/comments --paginate`
- **Global/issue comments:** `GET /repos/{o}/{r}/issues/{n}/comments --paginate` → author, body
- **Reviewers + states:** `GET /repos/{o}/{r}/pulls/{n}/reviews --paginate` (+ `reviewRequests` from GraphQL for pending)
- **PR meta:** GraphQL `id`, `headRefOid`, `title`, `body` (D10 needs title+body)

### Deferred mutations (gh-review has them; v1 doesn't — proven for fast-follow)
- **Add comment:** `POST /repos/{o}/{r}/pulls/{n}/comments` with `{body, commit_id, path, side, line[, start_line, start_side]}` (D5; v1 RIGHT-only)
- **Reply:** `POST /repos/{o}/{r}/pulls/{n}/comments/{comment_id}/replies`
- **Resolve / unresolve:** GraphQL `resolveReviewThread`/`unresolveReviewThread(input:{threadId})`
- **Add review (batched):** GraphQL `addPullRequestReview`

## The comment-anchor params (from gh-review `types.rs ReviewComment`)

```
ReviewComment { line: usize, side: Side(Left|Right), start_line: Option<usize>, start_side: Option<Side> }
```

Single-line → omit `start_line`/`start_side`. Multi-line → set both. `side` is
`RIGHT` for added/context lines (new-file line numbers), `LEFT` for deletion lines
(old-file line numbers). `commit_id` MUST be the PR's live `headRefOid` (a stale
SHA silently makes the comment "outdated"). This is exactly D5.

## What lazygit adds that gh-review lacks

1. **File tree** (D7) — gh-review's `file_picker.rs` is a flat fuzzy list, the
   specific thing the user dislikes.
2. **Mark file reviewed** (D8) — gh-review has only collapse/expand; no per-file
   "reviewed" state. This is the lazygit-native differentiator.
