## Why

The in-flight `pr-review-mode` change proved a single-panel PR review can live
inside lazygit: boot from `llg-dev OWNER/REPO PR_NUMBER`, show a changed-file
tree, render diffs, interleave review threads, and post a range comment. But in
use it does **not** feel like lazygit. The user's north star is explicit:

> the PR review experience should mirror the usual unstage/stage entire logic
> with the normal codebase — that's the goal.

Today it falls short of that bar in concrete, repeatable ways:

1. **Selecting a folder shows "This pull request has no changed files"** instead
   of the combined diff of every file under it (lazygit's commit-files panel
   shows the aggregate diff for a directory node).
2. **"Viewed" marks evaporate.** Toggling a file `[ ] → [x]`, leaving the PR, and
   returning resets everything to `[ ]` — there is no persistence, and no
   invalidation when a new commit changes a file (GitHub's "viewed" checkbox
   un-checks itself when the file changes; ours can't, because it doesn't persist
   at all).
3. **Tab cycling escapes the mode.** Pressing `[` / `]` drops the user back into
   the normal `Files · Worktrees · Submodules` / `Local branches · Remotes ·
   Tags` tabs — the review UI borrows the default windows instead of owning its
   own. The user has asked four times for review mode to have its **own** panels,
   tabs, and panel numbers, ignoring the normal layout entirely.
4. **Diffs ignore the user's configured diff tool.** The user reads diffs with
   git-delta everywhere else; the review diff is rendered by a bespoke presenter
   that bypasses the pager, so it looks nothing like the rest of their workflow.

The fix is not more patches on the single panel — it is to make PR review a
**first-class alternate workspace**: when lazygit boots in review mode it swaps
the entire left-hand side for a PR-shaped layout (a PR list, the PR's content,
the PR's activity), drives the main panel exactly like the normal app does
(hover-to-preview through the user's pager, enter-to-focus for line-level
actions), and persists "viewed" state the way GitHub does. This change absorbs
and supersedes the relevant parts of `pr-review-mode`.

## What Changes

### A dedicated review workspace (own windows, own tabs, own panel numbers)

When `ReviewTarget` is set, lazygit renders a **review layout** in place of the
normal side windows. Tab cycling (`[`/`]`) stays inside this layout; the normal
`Files/Branches/Commits/Stash/Status` windows are suppressed for the session.
The three side windows are:

- **`[1]` PR list** — tabs are the user's gh-dash PR sections
  (`My PRs · Needs My Review · Changes Requested · Involved · …`), **read from
  the existing `~/.config/gh-dash/config.yml`** rather than a new config. Each
  tab lists the PRs matching that saved query. Selecting a PR loads it into the
  other windows and the main panel. The CLI target (`llg-dev OWNER/REPO N`)
  pre-selects PR `N`.
- **`[2]` PR content** — tabs `Overview · Files Changed`.
  - **Overview**: window `[2]` shows a single PR-number row; the main panel shows
    the gh-dash-style PR overview (title, base ← head branch, author, dates, PR
    number, labels with their colors, assignees, milestone, linked development).
  - **Files Changed**: window `[2]` is the lazygit changed-file **tree** with
    per-file "viewed" tracking (persisted, invalidated when a new commit touches
    the file); the main panel renders the selected file's diff **through the
    user's configured pager (git-delta)**. Selecting a directory node shows the
    aggregate diff of all files under it.
- **`[3]` PR activity** — tabs `Conversation · Checks · Commits`.
  - **Conversation**: window `[3]` lists reviewers with a state icon (pending /
    commented / approved / changes-requested); the main panel shows that
    reviewer's comments/reviews with per-thread resolved/pending/answered state.
    PR-body (issue-level) comments are shown too, not just code comments.
    Submitting groups pending per-file comments into one review with a body.
  - **Checks**: window `[3]` lists every check / action as a **tree** (actions
    can have child actions; expand / collapse / enter), sorted by importance
    (failed › cancelled › interrupted › pending › succeeded › skipped) then by
    last-update; skipped sinks to the bottom. The main panel shows the selected
    action's logs/errors with forced color (`FORCE_COLOR=1`).
  - **Commits**: window `[3]` is lazygit's normal commits-panel behavior, but
    scoped to **only the commits between the PR's head and base**. Hover shows
    the commit message; enter drills into that commit's changed files → diff in
    the main panel.

### Behaviors that mirror the normal app

- **Hover previews through the pager; enter focuses for actions.** Just as the
  normal Files panel previews the working-tree diff through delta and the staging
  view is a focusable, line-selectable surface, Files Changed previews through
  delta and enter opens a focusable diff where the user marks reviewed
  hunks/lines and adds comments.
- **Folder → aggregate diff**, matching commit-files.
- **Persisted "viewed" state**, keyed by PR + path + content, invalidated on
  change (fixes #2).
- **Data via `gh`.** All GitHub reads/writes go through the `gh` CLI (the same
  tool gh-dash uses), not the hand-rolled `net/http`+go-gh transport — reusing
  the user's existing `gh` auth and gh-dash queries. This **supersedes**
  `pr-review-mode` decision D3.

### Works in any repository (project-agnostic)

PR review mode is **not** tied to any project, path, or repo. Repo identity
(`owner/repo`), the checkout path, and the PR's base/head OIDs + repo URLs are all
resolved **at runtime** from the launched checkout and the PR data — never from a
per-project config or a hardcoded path. Review refs and persisted viewed-state are
keyed by the runtime-resolved repo, so the same lazygit binary boots review mode in
any checkout. The dedicated review session (`R` = "[R]eview" in gh-dash, generic
`{{.RepoPath}} {{.RepoName}} {{.PrNumber}}` template vars) **replaces** the user's
old project-specific `L` worktree flow.

### Explicitly superseded from `pr-review-mode`

- **D3 (net/http transport) → `gh` CLI.**
- **D4 (bespoke inline presenter as the only diff path) → delta-rendered preview
  for reading**, with the bespoke line-addressable renderer retained only for the
  focusable comment-selection surface (the two-surface split the normal app
  already uses: pager preview vs. staging view).
- **D8 (session-only in-memory reviewed state) → persisted + content-invalidated.**

## Capabilities

### New Capabilities
- `pr-review-workspace`: a dedicated alternate lazygit layout, active when booted
  in PR review mode, presenting a PR list (from gh-dash queries), the selected
  PR's content (overview + a changed-file tree with delta-rendered diffs and
  persisted viewed-state), and the PR's activity (conversation/reviewers, a
  sortable checks tree with logs, and the PR-only commit list), with its own
  windows, tabs, and panel numbers, mirroring lazygit's hover-preview /
  enter-to-focus interaction model.

### Modified Capabilities
- `pr-review-mode`: superseded where it overlaps (transport, diff rendering,
  reviewed-state persistence, single-window layout). The CLI entry, boot guard,
  glow rendering, models, and add-comment mapping are carried forward.

## Impact

- **New side-window arrangement** gated on review mode (window-arrangement helper
  + `viewTabMap`), plus new contexts/views for: PR list, PR overview, checks
  tree, commits-in-PR, conversation/reviewers. Reuses the `filetree` machinery
  for the files tree and the checks tree.
- **New `gh`-backed command layer** (`pkg/commands/git_commands/github_review.go`
  reworked to shell `gh`): PR sections list, PR metadata, changed files + patches,
  review threads + issue comments + reviewers, check runs + logs, PR commit range.
- **Persisted viewed-state store** (per-repo JSON in lazygit's state dir), keyed
  by PR + path + content hash.
- **Pager integration**: render the per-file (and aggregate) diff through the
  user's configured external diff pager, as the normal diff preview does.
- **Dependencies**: `gh` (already required and present), `delta` (user-configured
  pager, already present), `glow` (already gated). No new Go modules.
- **No CI/pipeline changes. No PRs created.**
