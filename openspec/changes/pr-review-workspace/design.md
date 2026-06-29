## Context

This change builds on the in-flight `pr-review-mode` work (CLI entry, boot guard,
models, glow gate, add-comment mapping) and reshapes the UI into a full alternate
workspace. The relevant lazygit internals (verified by reading the code):

- **Side-window set** is computed in
  `pkg/gui/controllers/helpers/window_arrangement_helper.go`:
  `sidePanelChildren()` (≈line 434) lays out the boxes returned by
  `SideWindowNames(UserConfig)`. Windows are conditionally shown from that list —
  the same seam the `configurable-side-panel-visibility` work uses.
- **Tabs within a window** come from `gui.go viewTabMap()` (≈line 877), a
  `map[string][]context.TabView` (`{Tab, ViewName}`). `[`/`]`
  (`view_helpers.go handleNextTab/handlePrevTab`, ≈line 77) read the **focused
  window's** name, look up its tabs there, wrap the index, and push the tab's
  context. This is the only place tab membership is defined.
- **A context's window** is its `WindowName` (`base_context.go`), set at
  construction. Many contexts can share one window → they are its tabs.
- **Diff-through-pager**: `pty.go newPtyTask()` sets `GIT_PAGER` =
  `GetPagerConfig().GetPagerCommand(width)` and runs the git command in a PTY so
  the pager (delta) engages; `diff_helper.go RenderDiff()` wraps a git-diff cmd in
  `NewRunPtyTaskWithPrefix` → `RenderToMainViews`. **This path only renders local
  git output**, not arbitrary API patch text.
- **Staging/line-selection** (`patch_explorer_context.go`,
  `pkg/commands/patch`) already does hunk/line range selection over a **local**
  `git diff`, with `space` toggling staged state — i.e. the exact "stage/unstage"
  interaction the user wants to mirror.
- **Persistence**: `app_config.go AppState` (YAML at
  `$XDG_STATE_HOME/lazygit/state.yml`) already stores per-repo-path GitHub data
  (`GithubPullRequests map[string][]CachedPullRequest`). `SaveAppState`/
  `GetAppState` are the read/write API.
- **Screen mode**: `GuiRepoState.ScreenMode` is a global runtime flag the layout
  branches on (NORMAL/HALF/FULL) — precedent for a per-session layout flag.

## Goals / Non-Goals

**Goals**
- Review mode renders its **own** side windows, tabs, and panel numbers; `[`/`]`
  never escapes to the normal `Files/Branches/...` tabs (fixes bug #3, ask 3.1).
- The diff reading experience is **identical to the rest of lazygit**: hover a
  file → preview through the user's configured pager (delta); enter → a focusable,
  line-selectable surface where the stage/unstage interaction is reused for review
  (the user's explicit north star).
- Selecting a directory node shows the **aggregate** diff of its files (fixes #1).
- "Viewed" marks **persist** across leaving/returning to a PR and **invalidate**
  when a new commit changes the file (fixes #2).
- Window `[1]` lists PRs from the user's **gh-dash** saved queries; selecting one
  loads it. The CLI target pre-selects a PR (3.2).
- Window `[2]` = `Overview · Files Changed`; window `[3]` =
  `Conversation · Checks · Commits`, each behaving as specified (3.3–3.4).
- All GitHub access goes through the **`gh` CLI**.

**Non-Goals (deferred, phased out — see tasks.md)**
- Repo-less / clone-less boot (still requires a checkout, as `pr-review-mode` D2).
- Editing gh-dash config from inside lazygit (read-only consumption).
- Re-running / cancelling checks from the TUI (read + log view only in v1).
- Resolve/unresolve and reply mutations on existing threads (display only;
  carried from `pr-review-mode` non-goals), though batched **new** review submit
  is in scope (DW9).
- Side-by-side diff mode is whatever the pager provides (delta `--side-by-side`),
  not a bespoke renderer.

## Decisions

### DW1 — Review layout via a session flag branching `SideWindowNames` + `viewTabMap`
Add a review-mode predicate on the GUI/repo state (set when `StartArgs.ReviewTarget
!= nil`). Two seams branch on it:
1. `SideWindowNames()` returns the review windows
   `["prList", "prContent", "prActivity"]` instead of the normal set, so the
   normal windows are suppressed and the review windows own the left column.
2. `viewTabMap()` returns review tabs for those windows:
   `prContent → [Overview, Files Changed]`, `prActivity → [Conversation, Checks,
   Commits]`, `prList → [<gh-dash section titles…>]`.

Each review context sets `WindowName` to one of the three review windows; the
window→context membership and `[`/`]` cycling then work through the **existing**
mechanism — but only after the review flag is threaded deeper than the two seams
(see "Required threading" below).

*Rationale*: reuses the real seams the architecture exposes; the tab handler and
accordion layout need no review-specific code. Directly fixes the tab-escape bug
(#3) and ask 3.1 ("own panel numbers") because the windows are a different set.

*Required threading (critique correction — this is NOT just two functions).*
`SideWindowNames()` takes only `UserConfig`, but review mode is **session
state**; and several paths hard-code `Files` as the default/fallback focus, so
branching only the window list will misfocus or **panic** on a hidden window. The
flag must live on `GuiRepoState` (derived from `StartArgs.ReviewTarget`) and be
threaded into:
- `WindowArrangementArgs` + `window_helper.go SideWindows`/`SideWindowNames` so the
  arrangement uses the review windows.
- `context_config.go isSideWindowVisible` and `defaultSideContext` (~`:27-42`) so
  the default side context is `prList`, not `Files`.
- `layout.go:17` and `context.go:254/281` (Escape/pop + initial focus) so popping
  back never lands on a suppressed `Files` context.
- `window_helper.go GetViewNameForWindow` (~`:31-35`), which **panics** on an
  unmapped window — review windows must be registered before any focus call.
- `views.go:231-255` panel-number/title-prefix map (`[1]/[2]/[3]` labels) — add a
  review branch or these jump labels are wrong/absent.
- `refresh_helper.go` default refresh set (`:86-104`, always-status `:219`) —
  gate the normal Files/Branches/Commits/Stash refresh off in review mode (wasted
  work + races otherwise).

*Alternative*: a dedicated `SCREEN_PR_REVIEW` screen mode. Rejected as the
primary lever — screen mode controls *sizing*, not *which windows/tabs exist*. (We
*do* reuse the focus-inverts-split behavior for ask "80/20 ↔ 20/80" — see DW10.)

*Caveat — dynamic PR-list tabs (DW5)*: tabs are assigned during **view
configuration** (`views.go:278-291`) and tab-click bindings during **keybinding
reset** (`keybindings.go:368-376`); `viewTabMap()` (`gui.go:877`) is a cheap
static method. Reading gh-dash config or live PR sections *inside* `viewTabMap()`
risks stale tabs, missing click bindings, and hot-path shelling. Keep
`viewTabMap()` model-backed and cheap; load gh-dash sections elsewhere and feed
them in as data.

### DW2 — Local-refs for the READ preview only; keep the existing row-map surface for selection
This is the load-bearing decision, and the critiques (codex + copilot, agreeing)
corrected my first draft: the local-refs spine is right **for read-only diff
preview**, but it does **not** make the line-selection/comment surface or the
commits panel "free," and it must **not** reuse the staging controller (which
mutates the index/worktree). The corrected design has **two parallel diff
surfaces**, exactly like the normal app (pager preview vs. line-addressable
staging):

**(a) Read preview (hover) — local ref + pager, delta, display-only.** The
`R`/`llg-dev` launch runs **inside a checkout**. On PR-select, fetch the PR's head
and base into an isolated `refs/lazygit-review/<pr>/{head,base}` namespace, then
render preview via `WorkingTree.ShowFileDiffCmdObj(base, head, …, path)`
(`working_tree.go:439`) through the existing pty/pager path (`pty.go:46`) → delta
for free (fixes #4). A **directory** node passes the dir as the path → aggregate
diff (fixes #1), as commit-files already does. This output is **display-only**:
pager text is not line-addressable, and `ShowFileDiffCmdObj` forces `--no-renames`
(`working_tree.go:451`).

**(b) Selection surface (enter) — the EXISTING custom row-map context over a
parsed diff.** Keep `pr_review_diff_context.go` / `pr_review_diff.go` — they
already exist *because* `patch_exploring.State` breaks on interleaved comment rows
(`pr_review_diff_context.go:10`). The selectable surface runs over a **plain,
parsed** diff (the GitHub `pulls/{n}/files` patch, or `git diff --no-color
base...head -- path`), which preserves `path/side/line/start_line` for the
add-comment API (`github_review.go:97`) with the live `headRefOid`. **Do NOT
reuse `StagingController`** (`staging_controller.go:204` applies patches to the
index; `staging_helper.go:42` reads `Contexts().Files`). At most, extract the
`patch_exploring.State` *selection* logic behind a non-mutating interface — but
the existing row-map already does this, so v1 keeps it.

**PR commits (window `[3]`)** are **not** free either: `LocalCommitsContext` is
hardwired to `Model().Commits`, checked-out-branch rendering, rebase state, and
global refresh (`local_commits_context.go:28`, `refresh_helper.go:351`). Use a
**separate scoped PR-commits model/context** that loads `base..head` via the
existing `CommitLoader` `RefName`, not the normal commits context (3.4.3).

GitHub (via `gh`) remains the **metadata** layer: overview, reviewers, threads,
issue comments, checks, viewed-state target, and comment posting.

*Required new git command*: there is **no** existing wrapper to fetch arbitrary PR
refspecs (`sync.go:57` is a generic remote fetch). Add an explicit review-fetch:
`git fetch <head-repo-url> <headOid>:refs/lazygit-review/<pr>/head` and the base,
with `--no-write-fetch-head`, **no checkout, no branch writes**. PR heads often
live in **forks**, so fetch by explicit **repo URL + OID** for *both* base and
head (the read query currently has `headRefOid` but no base OID / head-repo URL —
`github_review.go:34-52` — so the query must add them). Verify `git merge-base`
succeeds before rendering. A separate worktree is **not** needed for read-only
diffs (extra complexity); plain ref fetch suffices.

*Repo-safety (critique P0)*:
- **Verify the checkout is actually `OWNER/REPO`** before any fetch/post — today
  the guard only checks "is a work tree" (`app.go:184-199`). A gh-dash misconfig
  could otherwise write refs into the wrong repo and post comments against
  mismatched files. Check remotes resolve to the target host/owner/repo (via
  `gh repo view` or remote URL match).
- **Isolate from normal views**: `git log --all` would surface
  `refs/lazygit-review/*` (`commit_loader.go:589-603`). Exclude that ref glob from
  the normal loaders (or accept it only in review mode) and **prune stale review
  refs on boot**.

*Rationale*: preview-through-pager genuinely "mirrors the usual diff experience"
and gets delta + folder-aggregate for free, while the already-built row-map
surface keeps line-addressing correct for comments. This is the minimal split the
normal app itself uses.

*Alternative (kept as fallback)*: skip the fetch and render the GitHub API patch
text through the pager via a PTY (pristine repo, no refs). Loses real `git diff`
fidelity and the PR-commits/drill-in story, but avoids local mutation entirely —
the fallback if the ref-fetch is judged too invasive.

### DW3 — Data via `gh` CLI; PR list from the gh-dash config
Supersedes `pr-review-mode` D3 (net/http + go-gh). All metadata reads/writes shell
`gh` (`gh api`, `gh api graphql`, `gh pr …`, `gh run view --log`) via the
`oscommands` argv builder + `DontLog().RunWithOutputs()`. The PR list window reads
`~/.config/gh-dash/config.yml` `prSections[]` (title + `filters`) and runs each
query (`gh search prs --json …` / `gh api`), so the user's existing sections
(`⚪ ANC`, `Needs My Review`, `Changes Requested (mine)`, `Involved`, …) appear as
tabs with no new config surface.

*Rationale*: matches the user's stated preference ("use the gh-dash builtin… not
adhoc implementations"), reuses their `gh` auth and saved queries, and is the same
tool the parent gh-dash process uses (no token/account mismatch).

*Risks*: gh-dash config may be absent/empty (fall back to a default
author/review-requested/involved set scoped to the launched repo); `filters`
strings are GitHub search syntax — pass through to `gh search prs` verbatim,
don't re-parse them. Config path honors `GH_DASH_CONFIG`/XDG.

### DW4 — Persisted, content-invalidated viewed-state
Supersedes `pr-review-mode` D8 (session-only). Persist viewed marks in `AppState`
(new field, e.g. `ReviewedFiles map[string]ReviewedFileState`), keyed by
`repoPath + "#" + prNumber + "#" + path`, storing the **blob SHA** (or patch hash)
the file had when marked. On load, a file shows as viewed only if its current blob
SHA still matches the stored one; a new commit that changes the file produces a
new SHA → the mark auto-clears (mirrors GitHub's "viewed" checkbox). Saved via
`SaveAppState` on toggle.

*Rationale*: fixes #2 with the persistence subsystem that already exists; the
content key gives the requested "discard if a new commit adds changes to that
file" invalidation for free.

*Correction (critique)*: a bare `path + blobSHA` key is insufficient —
**deleted** files have no head blob, **renames** need old/new path handling, and
binary/large files may lack a useful patch. Use a status-aware key
`{repo, pr, path, status, previousPath?, fileOid|patchHash}`. Prune entries for
merged/closed PRs opportunistically (state-file growth otherwise).

### DW5 — Window `[1]` PR list (tabs = gh-dash sections)
A list context per the `prList` window. Tabs are the gh-dash `prSections` titles;
the active tab's `filters` runs via `gh` and renders a PR row list (number, title,
author, review status — mirroring gh-dash's columns). `enter`/selection loads that
PR into windows `[2]`/`[3]` and triggers the DW2 fetch. The CLI `OWNER/REPO N`
target pre-selects the matching row (synthesizing a transient row if it isn't in
any section).

*Phasing*: v1 may ship with a single synthetic section containing just the
launched PR; the multi-section gh-dash browser is a distinct later phase.

### DW6 — Window `[2]` Overview · Files Changed
- **Overview tab**: window `[2]` shows one row (the PR number); the main panel
  renders a gh-dash-style overview — title, `base ← head`, author, created/updated
  dates, number, labels (with their hex colors → ANSI), assignees, milestone,
  linked development — from `gh pr view --json` / `gh api`.
- **Files Changed tab**: window `[2]` is the `filetree` changed-file tree (reused
  from `pr-review-mode` D7) with a persisted viewed indicator (DW4). Hover renders
  the file's (or folder's) diff into the main panel through the pager (DW2). Enter
  focuses the diff as a patch-explorer-style selectable surface (line/hunk select;
  `space` = reviewed, `c` = comment).

### DW7 — Window `[3]` Conversation · Checks · Commits
- **Conversation tab**: window `[3]` lists reviewers + a state icon (PENDING /
  COMMENTED / APPROVED / CHANGES_REQUESTED). Main panel shows the selected
  reviewer's review body + their thread comments with per-thread state
  (resolved / unresolved / outdated) and **PR-body issue-level comments** too
  (ask 3.4.1), grouped per reviewer/file. Authors and bodies render via glow.
- **Checks tab**: see DW8.
- **Commits tab**: window `[3]` reuses the commits-panel behavior over
  `git log base..head` (DW2). Hover shows the commit message; enter drills into
  the commit's changed files → diff in the main panel — standard lazygit
  sub-commits/commit-files behavior, scoped to PR commits only (3.4.3).

### DW8 — Checks tree, importance sort, logs with forced color
Window `[3]` Checks renders check runs / workflow jobs as a `filetree`-style tree
(jobs with child steps; expand/collapse/enter). Source: `gh api
repos/{o}/{r}/commits/{headSha}/check-runs` + workflow jobs (`gh api …/jobs`).
Sort key: **importance** then **last-updated desc**, importance order
`failure › cancelled › interrupted/timed_out › pending/in_progress ›
success/neutral › skipped`, with skipped sinking to the bottom. Enter on a leaf
renders its logs in the main panel via `gh run view <run-id> --log` (and
`--log-failed` for failures), executed with `FORCE_COLOR=1` so colored output
survives (ask 3.4.2).

*Risk*: check-runs vs. workflow-jobs are two different `gh` shapes; normalize into
one tree node model. Logs can be large — stream/cap and render off the main loop.

### DW9 — Batched review submission
New comments created across multiple files during a session accumulate as a
**pending review** and submit as one GitHub review with a body
(`gh api … pulls/{n}/reviews` with a `comments[]` array, event
COMMENT/APPROVE/REQUEST_CHANGES) — ask 3.4.1 "group comments … post their review
with an extra comment in the PR body". The immediate standalone-POST path from
`pr-review-mode` D5 remains the fallback for a single quick comment.

*Phasing*: v1 can keep standalone single-comment POST; batched review is a later
phase.

### DW10 — Focus inverts the split (80/20 ↔ 20/80)
When the user focuses a side window that needs more room (e.g. `[3]` Conversation
to read a long review), invert the side/main weighting so the side column grows
(the user's "focusing should switch 80-20 to 20-80"). Implement via the existing
accordion/screen-mode weighting in `window_arrangement_helper.go` (the focused
window already gets extra weight in HALF/FULL); tune the review layout's weights
so a focused activity/conversation window expands, and restore on blur.

*Phasing*: cosmetic; lands after the windows/tabs exist.

## Risks / Trade-offs

- **Local-repo mutation (DW2)** is the biggest one — see DW2. If judged
  unacceptable, fall back to the API-patch-through-pager alternative (loses the
  staging-machinery reuse).
- **Window-set assumptions (DW1)**: any code assuming `Files`/`Commits` windows
  always exist could panic in review mode. Audit before wiring.
- **Two GitHub data shapes** (REST check-runs vs. GraphQL threads vs. `gh pr
  view`): normalize at the command layer into local models; don't leak `gh` JSON
  upward.
- **`gh` latency & failure**: every panel does a `gh` call; need async loading
  states and graceful "gh failed" messaging, not a frozen "Loading…" (the exact
  bug that already bit this feature once).
- **Scope**: this is an epic. The phasing in tasks.md must let each phase ship and
  be green independently; the layout foundation (DW1) + local-refs spine (DW2)
  must land first because everything hangs off them.

## Critique reconciliation (codex gpt-5.5 + GitHub Copilot)

Both tools reviewed the draft independently and **converged** on the same
verdict, which raises confidence in these corrections:

- **DW1 is a foundation change, not two function edits.** The review flag is
  session state and must thread through `WindowArrangementArgs`,
  `SideWindowNames`/`SideWindows`, `isSideWindowVisible`/`defaultSideContext`,
  the Escape/pop + initial-focus paths, the panic-on-unmapped-window helper, the
  `[1]/[2]/[3]` panel-label map, and the default refresh set. (Folded into DW1.)
- **DW2 must not reuse the staging controller** (it mutates the index); keep the
  existing custom row-map selection surface, treat delta/pager as display-only,
  and maintain a parallel parsed diff for line→GitHub mapping. The local-ref
  fetch needs a new dedicated command, fork/base-OID awareness, repo-identity
  verification, and ref isolation/pruning. (Rewrote DW2.)
- **PR commits and viewed-state are under-claimed**; both need scoped
  models/keys, not the normal context. (Folded into DW2/DW4.)

**Agreed v1 scope (ship these):** the layout foundation, single launched PR,
files tree, pager read-preview (delta), the existing selectable comment surface,
standalone comment posting, and persisted viewed-state.

**Agreed deferrals (later phases / fast-follows):** the full multi-section
gh-dash PR browser, the checks tree + logs, grouped/batched review submission,
the 80/20↔20/80 focus-split inversion, and bespoke side-by-side (use
`delta --side-by-side` if wanted).

**Agreed phase order** (tasks.md follows it): 1) layout foundation with stub
contexts → 2) local-ref read model (fetch + pager preview, no commenting) →
3) parsed-diff selection surface with line-mapping tests → 4) persisted
viewed-state → 5) PR-commits scoped context → 6) conversation display →
7) comment submission (standalone, then grouped) → 8) gh-dash PR list + checks
tree last.

## Decisions from refinement (Session 1)

These resolve the Session-1 questions (`questions-session1.md`); they supersede the
matching Open Questions below.

### D1.1 — Diff source: local-ref fetch (Q1=A)
Confirmed the DW2 local-ref spine: fetch PR head+base into
`refs/lazygit-review/<pr>/*` and render through the user's pager. Where lazygit or
`gh` already provide a mechanism (pager pipeline, ref fetch, diff rendering),
reuse it rather than re-implementing. The API-patch fallback (DW2 alternative)
stays documented but is not the path.

### D1.2 — `R` replaces `L`; **project-agnostic is a hard requirement** (Q2=A)
PR review mode (`R`) supersedes the user's old gh-dash `L` worktree flow. The
load-bearing constraint: **nothing may be hardcoded to a specific project, repo,
or path.** The earlier `pr-review-mode` references to a fixed `/home/user/anc-review`
worktree are explicitly **forbidden** here. Concretely:
- Repo identity (`owner/repo`), the checkout path, and base/head OIDs + repo URLs
  are **derived at runtime** from the launched checkout and the fetched PR data —
  never read from a per-project config or constant.
- Review refs (`refs/lazygit-review/*`) and persisted viewed-state are keyed by the
  **runtime-resolved** repo identity, so the same lazygit binary works in any repo.
- The gh-dash launch keybinding uses only generic template vars
  (`{{.RepoPath}} {{.RepoName}} {{.PrNumber}}`), bound to `R` ("[R]eview"); the
  exact key is not load-bearing. lazygit must boot review mode in any checkout.

### D1.3 — Prune on boot (Q3=A)
On startup, prune `refs/lazygit-review/*` older than ~30 days and drop viewed-state
entries for merged/closed PRs. No eager on-quit cleanup.

### D1.4 — Viewed-state identity = blob OID, status-aware key (Q4=A)
Key `{repo, pr, path, status, previousPath?, fileOid|patchHash}`; identity is the
file's **blob OID** (one `git rev-parse`, matches GitHub). A file shows viewed only
if its current blob OID matches the stored one; deleted files keyed by
`previousPath+status` (not re-viewable once gone); a rename = new path → mark
clears; binary/large files fall back to patch hash. See `docs/viewed-state-schema.md`.

### D1.5 — Fail-fast at boot, toast + retry later (Q5=A)
If `gh` is missing/unauthenticated or the **initial** load (identity check, fetch,
first query) fails, exit fast with an actionable message (no infinite "Loading…").
Once the workspace is up, per-panel `gh` errors show a toast + retry, never a
frozen panel.

### D1.6 — v1 = phases 1–5; deferred requirements annotated (Q6=A)
v1 ships phases 1–5 (layout · local-ref preview · line-comment surface + standalone
posting · viewed-state · PR commits). Conversation, Checks, grouped-review, and the
multi-section gh-dash PR browser are marked `[DEFERRED — Phase N]` in `spec.md`;
v1's PR-list window shows only the launched PR.

### D1.7 — Suppress Status; cap render at 10 files / 2000 lines (Q7=A)
The Status window is suppressed in review mode. The files tree and the diff preview
are capped at **max 10 files OR 2000 lines**; beyond the cap, render a
"truncated — N more" note rather than streaming a huge diff/tree (protects against
the infinite-load/hang failure mode on large PRs).

## Open Questions

- ~~replace/coexist with `L`~~ → **resolved D1.2** (`R` replaces `L`; project-agnostic).
- ~~ref cleanup timing~~ → **resolved D1.3** (prune on boot).
- ~~viewed-state key blob vs patch~~ → **resolved D1.4** (blob OID).
- ~~PR list v1 scope~~ → **resolved D1.6** (launched PR only in v1).
- **Fetch on every PR selection vs. lazy on first diff view?** Recommend fetch on
  PR-select (so commits/files are ready) with a visible spinner. (Still open;
  low-risk default.)
- **Side-by-side**: rely on `delta --side-by-side` via a pager toggle (no bespoke
  renderer). (Still open; deferred with the pager-toggle nicety.)
