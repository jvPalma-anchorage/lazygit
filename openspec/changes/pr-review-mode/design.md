## Context

lazygit's GUI is built around a git repo present in `cwd`. CLI args are parsed in
`pkg/app/entry_point.go` with the vendored `flaggy` library: `Start()`
(entry_point.go:53) builds a `cliArgs`, does repo-path/chdir/env setup, then calls
`Run(appConfig, common, NewStartArgs(...))`. Exactly **one** positional is
registered (`git-arg`, index 1, entry_point.go:190) and `parseGitArg`
(entry_point.go:249) `log.Fatal`s on anything but status/branch/log/stash.
flaggy's `ShowHelpOnUnexpected=true` rejects an unregistered 2nd positional. So
both `lazygit owner/repo` and `lazygit owner/repo 123` are hard errors today —
the two-positional grammar is free to claim and the change is non-breaking.

`StartArgs` (pkg/app/types/types.go) carries `FilterPath`/`GitArg`/`ScreenMode`/
`IntegrationTest` into the GUI; `gui.go` `initialContext()` maps `GitArg` → which
list context is focused first, but it returns `types.IListContext` only.

The fork already fetches PR data without `gh`: `github.go:141 GetAuthToken`,
`:198 fetchRecentPRsAux(endpoint, owner, repo, branches, token)`, `:339
graphQLEndpoint` — a hand-rolled GraphQL-over-`net/http` path fully parameterized
by owner+repo. The same transport can carry the larger review-data query and the
add-comment mutation/POST. `models.GithubPullRequest` exists but is metadata-only;
review threads/comments/reviewers are new models.

The unified-diff parser under `pkg/commands/patch` is reusable via its public API
(`patch.Parse`, `Patch.Lines()`, `Patch.LineNumberOfLine(idx)` → new-file line
number), but `patch_exploring.State` (the `v` range-select engine) is **not**
reusable as-is: it hard-assumes view-line==patch-line, which interleaved comment
rows break.

The diff-to-main render paths (`pty.go newPtyTask`, commit-files
`ShowFileDiffCmdObj` → `RunPtyTask`) stream raw pre-rendered text into a view with
no model and no injection point — a dead end for inline comments. A view is just
an ANSI buffer fed by `NewRenderStringWithoutScrollTask`, so a custom presenter
can build its own string.

`glow` 2.1.1 is installed. The argv command builder (`oscommands` `Cmd.New` +
`SetStdin` + `RunWithOutputs`) and the gocui `OutputTrue` ANSI render path make
piping markdown through glow clean and shell-injection-safe.

## Goals / Non-Goals

**Goals (v1):**
- Boot directly into review mode from `lazygit OWNER/REPO PR_NUMBER` launched by a
  gh-dash keybinding.
- Show the PR's changed files using lazygit's **file-tree** UX (nested,
  collapsible) — not a flat list (D7).
- **Mark files as reviewed** from the tree, session-scoped (D6).
- Render each file's diff; **toggle unified ↔ side-by-side** (`[t]`, D8).
- Show existing review threads interleaved at their anchored lines on **both
  sides** (RIGHT added/context and LEFT deletion), with author and
  resolved/unresolved badge, including reply chains.
- Show PR global (issue-level) comments with author.
- Show reviewers and their latest review state.
- View the PR **description** (`[d]`, D9).
- Add a new review comment over a selected start+end line range (RIGHT side).
- Render markdown through glow when present; plain-text fallback otherwise.

> **Reference blueprint:** `~/projects/gh-review` (Rust) implements this exact
> feature and is a proven reference. See `docs/gh-review-reference.md` for the
> module→lazygit mapping. Its `DisplayRow` interleaving model, `gh` API calls,
> and `DiffMode{Unified,SideBySide}` directly validate the decisions below.

**Non-Goals (deferred / explicitly out of v1):**
- **Clone-less boot.** v1 requires a local checkout (launched via
  `cd {{.RepoPath}}` or `-p`); operating purely from the API with a synthetic/
  temp RepoPaths is deferred (see D2).
- **LEFT-side (deletion-line) comment *adding*.** Both sides' existing threads are
  *displayed* in v1 (gh-review proves it); only *creating* a new comment is
  RIGHT-side-only in v1 because the new-line→API mapping is simpler there.
- **Persisted reviewed state.** Reviewed marks are in-memory for the session only;
  on-disk persistence keyed by PR head SHA is deferred.
- **Resolve / unresolve mutations.** Reading `isResolved` is in scope; the
  GraphQL `resolveReviewThread`/`unresolveReviewThread` mutations are deferred.
- **Batched / pending reviews** (`POST /pulls/{n}/reviews` with a `comments[]`
  array, submit as APPROVE/REQUEST_CHANGES). v1 posts standalone comments.
- **Reply-to-thread and approve/request-changes from the TUI.** Reply needs the
  comment `databaseId`; we fetch it (so it is plumbed) but the reply action UI is
  deferred.
- **Full LEFT+RIGHT selectable, collapsible inline comment blocks.** v1 blocks are
  always-expanded and non-selectable.
- **Pagination completeness.** v1 caps each connection at `first:50` and shows a
  truncation note; endCursor looping is deferred.
- **PR-URL positional** (`https://github.com/O/R/pull/N`). v1 accepts only the
  `OWNER/REPO` + numeric forms.

## Decisions

### D1 — CLI entry: two positionals, detected after Parse, before parseGitArg
Register a 2nd optional positional `pr-number` at index 2 in
`parseCliArgsAndEnvVars` and add `ReviewOwner`/`ReviewRepo`/`ReviewPRNumber` to
`cliArgs`. After `flaggy.Parse()`, if positional-1 matches `^[^/\s]+/[^/\s]+$`
(one slash, no spaces) **and** positional-2 is all digits, set the review fields,
normalize (strip a trailing `.git`, lowercase host-insensitively is unnecessary),
and **clear `GitArg`** so `parseGitArg` never sees `owner/repo`. Otherwise fall
through to today's single-positional behavior unchanged.

*Rationale:* The GOAL fixes the bare two-positional signature. The owner/repo
regex cannot match the four git-arg keywords (none contain `/`), so fallthrough is
safe. Detecting before `parseGitArg` avoids the `log.Fatal`.

*Alternatives:* (a) a flaggy subcommand `lazygit review owner/repo 123` parses
unambiguously but violates the GOAL's signature and makes gh-dash emit an extra
token. (b) a flag form `--pr owner/repo#123` is unambiguous but also off-spec.
Positional detection chosen to honor the GOAL; subcommand noted as the fallback if
the heuristic proves fragile.

### D2 — Local-repo-or-not: v1 requires a local checkout (guard tweak only)
gh-dash resolves `{{.RepoPath}}` from its `repoPaths:` map and the keybinding does
`cd {{.RepoPath}} && llg-dev ...`, so lazygit boots inside a real work tree. v1
therefore does **not** attempt repo-less boot. The only app.go change is to make
sure `setupRepo`'s not-a-repo git-init prompt / `os.Exit(1)` is never reached in a
misconfigured launch: when `ReviewTarget` is set and cwd is not a repo, fail fast
in `Start()` with a clear message ("PR review mode must be launched inside a
checkout of OWNER/REPO; configure gh-dash repoPaths") rather than prompting on
stdin (which would hang under gh-dash's non-interactive exec).

*Rationale:* Repo-less boot means relaxing the deepest GUI invariant
(`resetState` keys all state off `RepoPaths.WorktreePath()`, `gui.git` is built
per-repo). That is a large, risky rewrite. Since the user already maps the target
repo in gh-dash, requiring a checkout is realistic for v1 and removes the single
biggest boot landmine.

*Alternative (deferred):* temp `git init` scratch dir to satisfy `RepoPaths`
while all data comes from the API — smaller than a fully repo-less context but
still touches `resetState`/`RepoStateMap` keying; flagged as the v2 path to true
clone-less review.

### D3 — Data source: reuse the in-repo net/http + go-gh GraphQL transport
Do **not** shell out to `gh` for the read path. Add `github_review.go` next to
`github.go` and reuse `GetAuthToken(host)` + a `graphQLEndpoint(host)` POST,
mirroring `fetchRecentPRsAux`. One combined query returns review threads (+reply
chains), issue comments, reviewers/reviews, and PR `id`/`headRefOid`. The unified
diff (which GraphQL cannot return) comes from a second REST call —
`GET /repos/{o}/{r}/pulls/{n}/files` per-file `patch` field — over the same token.

*Rationale:* Zero new runtime dependency, consistent with existing PR-fetch code,
and the token/account is already the one the rest of the fork uses (shelling to
`gh` risks a different `GITHUB_TOKEN`/account than go-gh resolves). The queries are
identical regardless of transport; only the transport differs.

*Alternative:* shell `gh api graphql` (matches the gh-dash parent context, simpler
query authoring) — rejected to avoid a gh binary dependency, a process spawn per
fetch, and a possible auth-account mismatch.

*Caveat carried as a risk:* connections are paginated; v1 caps at `first:50` with
a visible truncation note (see Non-Goals).

### D4 — Inline render: custom both-side presenter with non-selectable blocks
The PR-review diff view does **not** use `RunPtyTask`. A new presenter
(`pr_review_diff.go`) parses the per-file diff with `patch.Parse`, iterates
`Patch.Lines()`, and for each ADDITION/CONTEXT line computes its new-file line via
`Patch.LineNumberOfLine(idx)`; when a thread anchors to that (Side=RIGHT, line),
it emits the diff line then the thread's comment block. It builds two parallel
slices while emitting: the ANSI `content` and a `rowKind []enum`
(DIFF/COMMENT/HEADER) plus `rowPatchIdx []int` — this is the selection map that
replaces `patch_exploring.State`. The buffer is fed via
`NewRenderStringWithoutScrollTask` into the review main view.

Comment blocks are **non-selectable** (tagged COMMENT). Each block shows the
thread author + resolved badge (green RESOLVED / yellow UNRESOLVED from
`IsResolved`) and each comment's author + body (body run through `renderMarkdown`,
see D6). Bodies wrap to `view.InnerWidth()`; the whole buffer + maps rebuild on
width change (`NeedsRerenderOnWidthChange`).

*Rationale:* This is the only path that supports interleaving; every primitive has
an in-repo precedent (staging_helper builds a diff string then RenderToMainViews).
RIGHT-side-only needs **no new methods on the patch package** for v1.

*Alternative (cheaper fallback if schedule slips):* a gutter marker on commented
lines + a secondary split-main panel showing the thread for the line under the
cursor (reuses `splitMainPanel`, no custom main renderer). Recorded as the
de-scope option; not the chosen v1.

*Both-side display (refinement):* v1 *displays* threads on both sides so no
comments are hidden — additions/context anchor by new-file line (Side=RIGHT),
deletions anchor by old-file line (Side=LEFT). Deletion anchoring needs the
old-file line per row (an `OldLineNumberOfLine`-style lookup; gh-review computes
both `old_lineno`/`new_lineno` per diff line in `diff/model.rs` — mirror that).
Only *adding* a comment stays RIGHT-only (D5).

*Honest verdict:* the inline renderer is feasible but a non-trivial build
(selection-skips-over-comment-rows + width re-wrap + glow-in-block alignment), not
a quick job — gh-review's `diff/{model,renderer}.rs` (~780 lines) is the proof and
the template. v1 commits to both-side *display*; selectable/collapsible blocks
remain a non-goal.

### D5 — Add-comment mapping: line+side, standalone POST, RIGHT-only in v1
GitHub's review-comment API takes **file line numbers + side**, not the legacy
diff `position` offset. From the selected DIFF-row range, translate the start/end
rows back to new-file line numbers via `rowPatchIdx` + `LineNumberOfLine`, then
`POST /repos/{o}/{r}/pulls/{n}/comments` with `{body, commit_id=headRefOid, path,
side:RIGHT, line=end}` and, when start≠end, `{start_line=start, start_side:RIGHT}`.
Single-line comments omit `start_line`. `commit_id` is the live `headRefOid` from
the read query (never local HEAD — a stale SHA silently makes the comment
"outdated"). No prior review is needed (GitHub auto-wraps it in a single-comment
review). Transport mirrors github.go's net/http+token.

*Rationale:* line+side maps directly to numbers the patch package already
computes; standalone POST needs no review lifecycle/id to manage. RIGHT-only keeps
v1 free of an old-file line-number helper and avoids mixed-`start_side`/`side`
422s.

*Alternatives:* (a) pending-review batch (`POST /pulls/{n}/reviews`) — v2 nicety,
adds a pending-review-id state machine, deferred. (b) LEFT/mixed-side ranges —
require a new `OldLineNumberOfLine(idx)` on the patch package and careful 422
guards; deferred. v1 rejects/clamps any selection that resolves to a non-RIGHT
line with a toast.

### D6 — Glow: cached LookPath gate, argv builder, explicit style + forced color
Add `pkg/gui/glow.go` mirroring `pkg/gui/gh.go`: a `glowAvailable bool` resolved
once at startup (with a `GLOW_AVAILABLE_OVERRIDE` env knob for integration tests)
and a unit-testable `glowIsAvailable(lookPath)` core. `renderMarkdown(body, width)`
returns `body` unchanged when glow is absent/errored/empty; otherwise it runs
`Cmd.New([]string{"glow","-s","dark","-w",width})` (argv — **never** `NewShell`)
`.AddEnvVars("CLICOLOR_FORCE=1").SetStdin(body).RunWithOutputs()` and returns
stdout.

*Grounded corrections baked in:* (1) `-s auto` emits **zero** ANSI when stdout is
a pipe (verified on glow 2.1.1) — use explicit `-s dark` + `CLICOLOR_FORCE=1`.
(2) Use `RunWithOutputs` (separate streams), not `RunWithOutput`
(`CombinedOutput`), so a glow stderr warning never splices into the rendered body.
(3) glow spawn is ~tens of ms — render off the gocui main loop (`OnWorker`/
`OnUIThread`) or memoize keyed by `(body,width)` and only re-render on
selection/width change.

*Security:* the markdown bytes go via `SetStdin`, never into a shell string; a
test asserts the glow argv contains no interpolated body text.

*Rationale/alternative:* `-s dark` hardcoded for v1 (lazygit exposes no
"terminal is light" signal to glow); a `git.pullRequests.glowStyle` config knob is
a later nicety.

### D7 — File list: reuse lazygit's file-tree presentation, not a flat list

The review file panel reuses lazygit's existing file-tree machinery (the
`filetree`/`patch_explorer`-adjacent presentation used by the commit-files and
working-tree views: nested directory nodes, collapse/expand) rather than a flat
filename list. The model is the PR's changed-file paths (from `pulls/{n}/files`).

*Rationale:* This is the user's explicit preference over gh-review's flat
`file_picker.rs` fuzzy list — and lazygit already owns a battle-tested tree
renderer, so reuse beats reinventing. The diff view is driven by the file under
the cursor (standard side-list → main render).

*Alternative:* a flat fuzzy-filtered list (gh-review's approach) — rejected; it's
the specific thing the user dislikes about gh-review.

### D8 — Mark file reviewed: session-scoped in-memory set, distinct indicator

A new key in the review context toggles a "reviewed" flag on the selected file.
State lives in an in-memory `map[string]bool` (or `set`) on the PR review context,
keyed by file path, surfaced as a distinct indicator column in the tree. It is
**not** persisted across restarts in v1 and is **orthogonal to the git index**
(this is a review-session concept, not staging).

*Rationale:* gh-review lacks this entirely (only collapse/expand) — it's the key
thing lazygit adds. Ephemeral state matches lazygit's "git is the source of truth"
philosophy and needs no new persistence subsystem. A review session is
launch→review→quit, so in-memory is sufficient. Force-push staleness is a non-issue
within a single session (the head SHA is fixed for the run).

*Alternative (deferred):* persist to a per-repo client-state file keyed by
PR-head-SHA so marks survive restarts and invalidate on force-push — a real
subsystem, deferred to v2.

*Keybinding:* a dedicated key (not the staging `space`) to avoid implying a git
mutation; the exact key is an Open Question, but it is context-scoped to the review
file tree.

### D9 — Diff mode toggle: `[t]` cycles unified ↔ side-by-side

A `DiffMode { Unified, SideBySide }` flag on the PR review context (mirroring
gh-review's `types.rs DiffMode`), toggled by `[t]`, drives the presenter (D4).
Unified is the v1-primary, fully-specified path; side-by-side renders old/new in
parallel columns with threads still anchored to their lines.

*Rationale:* explicit user ask, and a proven gh-review feature
(`diff/renderer.rs` has both modes off one `DisplayRow` model). Building the
`DisplayRow`/row-map abstraction in D4 once lets both modes share it.

*Honest note:* side-by-side is the more complex render path (column widths, wrap,
placing a full-width comment block across two columns). If schedule slips,
side-by-side is the first thing to defer to a fast-follow — unified alone already
satisfies "read the diff + inline threads".

### D10 — PR description: `[d]` opens title+body via glow in a transient panel

`[d]` opens the PR description (title + body, already fetched in the combined read
query — add `body`/`title` to it) rendered through `renderMarkdown` (D6) into a
transient/secondary panel, dismissible back to the review.

*Rationale:* explicit user ask; trivial given the data is already fetched and glow
rendering exists (D6). Reuses lazygit's existing transient-panel/secondary-view
mechanism rather than a new window.

## Risks / Trade-offs

- **Boot landmine:** if review detection isn't wired before `setupRepo`, launching
  from gh-dash's arbitrary cwd either hangs on a stdin git-init prompt or
  `os.Exit(1)`s — a silent no-op to gh-dash. D2 fails fast instead. *Mitigation:*
  detect in `Start()` before the repo-path block; add a test for the
  not-a-repo + ReviewTarget path.
- **flaggy ShowHelpOnUnexpected:** if `pr-number` index 2 isn't registered,
  `lazygit owner/repo 123` prints help and exits 0 — feature appears to do
  nothing. *Mitigation:* register the positional (D1) and integration-smoke the
  argv.
- **initialContext type:** it returns `IListContext`; the review context is not
  one. Branch one level up in `resetState`, not inside `initialContext`. *Don't*
  force the review context to satisfy `IListContext`.
- **RepoStateMap keying:** review state must not clobber the normal repo's cached
  state for the same worktree path. *Mitigation:* key review session state
  distinctly (e.g. include the PR number) or scope it to the review context.
- **Outdated comments:** when code changed after a comment, `thread.line` is null
  and only `originalLine`/`diffHunk` are valid — these can't anchor to the current
  diff and must render in a separate "outdated" section, else they vanish or
  mis-anchor.
- **Null reviewer:** `reviewRequests.requestedReviewer` can be null (team requests
  / already-reviewed); naive `.login` panics. Null-check + handle Team via
  `__typename`.
- **Stale commit_id → 422 / silent outdated:** add-comment must use the live
  `headRefOid`, and `line`/`start_line` must be real diff-body lines (exclude
  hunk/file headers, snap selection to change lines), or GitHub 422s.
- **`patch_exploring.State` reuse temptation:** it will silently corrupt selection
  once comment rows interleave. The parallel selection map is mandatory.
- **glow `-s auto` footgun + spawn cost** — covered by D6.
- **Pagination truncation:** `first:50` can drop data on huge PRs; v1 shows a
  truncation note (Non-Goal to loop endCursor).

## Open Questions

- Which gh-dash key to bind — recommend `R` (free; `L/E/D/B` are taken). Should PR
  review mode coexist with the existing `L` worktree-diff flow or replace it?
- Should `Start()` fail fast when the go-gh token is empty/unauthorized, or boot
  then surface an in-UI error? Recommend fail-fast in `Start()` with a clear
  message.
- Should the review buffer be one scrollable concatenation of all files, or
  one-file-at-a-time selected from a file-tree side panel (reusing
  CommitFilesContext UX)? One-file-at-a-time is far simpler for selection
  bookkeeping — recommended for v1.
- Whether to plumb `ReviewTarget` through `NewApp` (matches IntegrationTest
  precedent) vs a Gui field set in `Start()`. Recommend StartArgs threading.
- Does any code path set the review window's default/active context directly,
  bypassing the boot branch? Verify before wiring.
