# Tasks

Dependency-ordered. Phase 1 lands the CLI entry + boot-into-mode skeleton (a
review context that opens but renders a placeholder), so every later phase has a
real boot path to attach to.

## 1. CLI entry + boot-into-mode skeleton

- [x] 1.1 In `pkg/app/types/types.go`, add `type ReviewTarget struct { Owner, Repo string; PRNumber int }`, add a `ReviewTarget *ReviewTarget` field to `StartArgs`, and thread it through `NewStartArgs(...)`.
- [x] 1.2 In `pkg/app/entry_point.go` `parseCliArgsAndEnvVars`, register a 2nd optional positional `flaggy.AddPositionalValue(&prNumberArg, "pr-number", 2, false, ...)` and add `ReviewOwner`/`ReviewRepo`/`ReviewPRNumber` to the `cliArgs` struct + its return.
- [x] 1.3 In `pkg/app/entry_point.go` `Start()`, after `parseCliArgsAndEnvVars()` and before the `cliArgs.RepoPath` block, detect review mode: if `GitArg` matches `^[^/\s]+/[^/\s]+$` and `prNumberArg` is all digits, parse it (strip trailing `.git`), populate `cliArgs.Review*`, and clear `cliArgs.GitArg` so `parseGitArg` never sees `owner/repo`. Atoi failure → `log.Fatalf` with a clear message.
- [x] 1.4 In `Start()`, build a `*appTypes.ReviewTarget` when the review fields are set and pass it into `NewStartArgs(...)` at the `Run(...)` call.
- [x] 1.5 In `pkg/app/app.go`, when `StartArgs.ReviewTarget != nil` and cwd is not a repo, fail fast in `Start()`/`NewApp` with a clear message (do NOT reach `setupRepo`'s git-init prompt / `os.Exit`). Add a test for the not-a-repo + ReviewTarget path.
- [x] 1.6 Create `pkg/gui/context/pr_review_context.go`: a new context (NOT an `IListContext`) registered in `pkg/gui/context/setup.go` and declared in `pkg/gui/views.go`, rendering a placeholder string for now.
- [x] 1.7 In `pkg/gui/gui.go` `resetState` (one level above `initialContext`), when `startArgs.ReviewTarget` is set, make the PR review context the initial/focused context instead of the Files context. Ensure `RepoStateMap` keying for a review session does not clobber the normal repo state (scope by PR number or to the review context).
- [x] 1.8 Add user-facing strings to `pkg/i18n/english.go` (review-mode title, loading, error messages). Run `make generate` after string/schema changes.
- [ ] 1.9 Manual smoke: `cd <checkout> && llg-dev OWNER/REPO 123` boots into the placeholder review context; `lazygit`, `lazygit status` still work.

## 2. Glow rendering gate (independent; needed by 4 and 5)

- [x] 2.1 Create `pkg/gui/glow.go` mirroring `pkg/gui/gh.go`: a `glowAvailable bool` Gui field resolved once at startup, a `detectGlowAvailable()` honoring a `GLOW_AVAILABLE_OVERRIDE` env var, and a testable `glowIsAvailable(lookPath func(string)(string,error)) bool` core.
- [x] 2.2 Implement `renderMarkdown(body string, width int) string`: return `body` when `!glowAvailable` or `strings.TrimSpace(body)==""`; else `Cmd.New([]string{"glow","-s","dark","-w",strconv.Itoa(width)}).AddEnvVars("CLICOLOR_FORCE=1").SetStdin(body).DontLog().RunWithOutputs()`; on err or empty stdout return `body`. Use `RunWithOutputs` (NOT `RunWithOutput`).
- [x] 2.3 Create `pkg/gui/glow_test.go`: assert `glowIsAvailable` true/false via injected lookPath; assert the glow argv contains no interpolated body text (security: body only via stdin).
- [x] 2.4 Memoize rendered output keyed by `(body,width)` and run glow off the main loop (`OnWorker`/`OnUIThread`) or only on selection/width change.

## 3. GitHub review data path (read)

- [x] 3.1 Add models in `pkg/commands/models/github.go` (or sibling): `ReviewThread{ID, Path string; Line, StartLine int; Side string; IsResolved, IsOutdated bool; DiffHunk string; Comments []ReviewComment}`, `ReviewComment{DatabaseID int; Author, Body, CreatedAt string; ReplyToID int}`, `IssueComment{Author, Body, CreatedAt string}`, `Reviewer{Login, State string}`. Capture PR `ID` and `HeadRefOid`.
- [x] 3.2 Create `pkg/commands/git_commands/github_review.go`: a `FetchPRReviewData(owner, repo string, number int, token string)` that POSTs the combined GraphQL query (threads + reply chains + issue comments + reviewRequests/latestReviews/reviews + `id`/`headRefOid`) to `graphQLEndpoint(host)` with the go-gh token, mirroring `fetchRecentPRsAux`. Cap each connection at `first:50`; set a `Truncated` flag when `hasNextPage`.
- [x] 3.3 In the same file, add `FetchPRChangedFiles(owner, repo, number, token)` calling `GET /repos/{o}/{r}/pulls/{n}/files` for per-file `filename/status/patch`. Decide diff source: REST `patch` per file (chosen, integrates with per-file rendering).
- [x] 3.4 Null-safety: handle `reviewRequests.requestedReviewer == null` and Team via `__typename`; map `thread.line == null` (outdated) to the outdated bucket using `originalLine`/`diffHunk`.
- [x] 3.5 Unit-test the response parsing against the captured sample JSON in `docs/github-api.md` (threads, reply chain, reviewers, outdated case).
- [x] 3.6 Wire the fetch into the review context load: call on `OnWorker` at boot, store results on the context, `OnUIThread` to render.

## 4. Inline diff + comment presenter (read render)

- [x] 4.1 Create `pkg/gui/presentation/pr_review_diff.go`: parse the per-file diff with `patch.Parse`; iterate `Patch.Lines()`; for ADDITION/CONTEXT lines compute new-file line via `Patch.LineNumberOfLine(idx)`; emit diff line, then (if a RIGHT-side thread anchors there) the comment block.
- [x] 4.2 Build parallel `content string`, `rowKind []enum{DIFF,COMMENT,HEADER}`, `rowPatchIdx []int` while emitting (the selection map). Colorize mirroring `pkg/commands/patch/format.go` (additions green, deletions red, hunk header cyan).
- [x] 4.3 Render each thread block: header line = first comment author + RESOLVED/UNRESOLVED badge (from `IsResolved`); each comment = author label + body via `renderMarkdown(body, view.InnerWidth())`; wrap with `utils.WrapViewLinesToWidth`.
- [x] 4.4 Render an outdated/other section at the end of the file for threads with no live anchor (show `diffHunk` verbatim + comments).
- [x] 4.5 Feed the buffer via `NewRenderStringWithoutScrollTask` into the review main view (`RenderToMainViews`); do NOT use `RunPtyTask`.
- [x] 4.6 Rebuild buffer + maps on width change (`NeedsRerenderOnWidthChange`); cache rendered string on file-select, re-render only on width change or comment add.
- [x] 4.7 Render global comments (issue-level) and reviewers+states in side panels/sections, authors via the models, bodies via `renderMarkdown`.

## 5. Add review comment by range (write)

- [x] 5.1 Implement range selection over the review buffer driving gocui directly (`view.SetRangeSelectStart` / `SetCursorY` like `patch_explorer_context.go`), snapping the cursor to the next/prev `rowKind==DIFF` row (skip COMMENT/HEADER rows). Do NOT reuse `patch_exploring.State`.
- [x] 5.2 Translate selected `[start,end]` DIFF rows → new-file line numbers via `rowPatchIdx` + `Patch.LineNumberOfLine`. If start>end after mapping, swap. If either endpoint resolves to a non-RIGHT (deletion) line, reject with a toast (v1 RIGHT-only).
- [x] 5.3 Add a controller action: prompt for the comment body, then call the write path.
- [x] 5.4 In `github_review.go`, add `AddReviewComment(owner, repo string, number int, headOid, path, body string, startLine, line int, token string)`: `POST /repos/{o}/{r}/pulls/{n}/comments` (net/http + token) with `{body, commit_id=headOid, path, side:RIGHT, line}`; include `{start_line, start_side:RIGHT}` only when `startLine != line`.
- [x] 5.5 On success, refresh the review data (re-fetch threads) so the new comment appears inline; on 422 surface the error via toast.

## 6. gh-dash integration + docs + tests

- [x] 6.1 Document the gh-dash keybinding (see `docs/gh-dash-integration.md`): `cd {{.RepoPath}} && /home/user/.local/bin/llg-dev {{.RepoName}} {{.PrNumber}}` under `keybindings.prs`, key `R`, with the required `repoPaths:` map entry. No gh-dash code change.
- [x] 6.2 Add integration tests under `pkg/integration/tests/`: launch-into-review-mode (mode boots, files listed), and glow present/absent rendering (using `GLOW_AVAILABLE_OVERRIDE`). Run `make generate` to register them in `test_list.go`.
- [x] 6.3 Run `make format`, `make lint`, `make unit-test`; ensure each commit compiles and is green (per AGENTS.md).
- [x] 6.4 Update `docs-master/keybindings/*` and cheatsheets via `make generate` if any new keybindings are user-visible.

## 7. File tree, reviewed state, view toggles, description (gh-review parity + lazygit refinements)

> See `docs/gh-review-reference.md` for the proven blueprint these map onto.

- [x] 7.1 (extends 3.2) Add `body` and `title` to the combined read query, and per review-thread capture both `line`/`side` and `originalLine`/`startLine` so LEFT-side (deletion) threads can be anchored (D4 both-side display).
- [x] 7.2 (extends 4.1) In `pr_review_diff.go`, also anchor DELETION-line threads: compute the old-file line per row (add an `OldLineNumberOfLine`-style helper to `pkg/commands/patch` or derive from `patch.Parse` line metadata) and emit LEFT-side thread blocks under their deletion line. No comment may be silently dropped; outdated-only threads still go to the trailing section.
- [x] 7.3 File tree (D7): render the PR's changed files via lazygit's existing file-tree presentation (nested, collapsible) in the review file panel — reuse the `filetree` machinery used by commit-files/working-tree, not a flat list. Cursor-on-file drives the diff render.
- [x] 7.4 Mark reviewed (D8): add a `reviewed map[string]bool` on the PR review context, a context-scoped keybinding to toggle it for the selected file, and a distinct reviewed indicator column in the tree. Session-scoped in-memory; orthogonal to the git index. Add i18n strings + the keybinding; `make generate` for cheatsheets.
- [x] 7.5 Diff-mode toggle (D9): add a `DiffMode {Unified, SideBySide}` flag on the review context, bind `[t]` to cycle it, and re-render the presenter accordingly. Unified is required; side-by-side (parallel columns, threads still anchored) may land as a fast-follow if schedule slips — gate it so unified always works.
- [x] 7.6 PR description (D10): bind `[d]` to open the PR title+body (from 7.1) rendered via `renderMarkdown` (glow when present) in a transient/secondary panel, dismissible back to the review.
- [x] 7.7 Integration tests: file marked-reviewed indicator toggles; `[t]` switches unified↔side-by-side; `[d]` opens/dismisses the description. Register via `make generate`.
