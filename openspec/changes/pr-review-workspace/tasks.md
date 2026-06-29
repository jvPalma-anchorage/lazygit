# Tasks

Dependency-ordered, following the codex+copilot-agreed phasing. **Phases 1–5 are
v1** (ship the real "mirror stage/unstage" review of a single launched PR).
**Phases 6–8 are deferred** fast-follows. Each phase must compile and stay green
on its own (AGENTS.md). The order is deliberate: nothing renders until the layout
foundation (P1) and the read model (P2) exist.

Session-1 decisions (design.md D1.1–D1.7) are folded in: local-ref spine,
**project-agnostic** identity (no hardcoded paths), prune-on-boot, blob-OID
viewed-state, fail-fast boot, v1=phases 1–5, Status suppressed + render cap (≤10
files / 2000 lines). Data shapes and contracts live in `docs/data-shapes.md`,
`docs/review-fetch-command.md`, `docs/viewed-state-schema.md`, and
`docs/REFERENCES.md` — read those before implementing the relevant task.

## 1. Layout foundation (review-mode owns its windows/tabs/panel numbers)

> Fixes bug #3 / ask 3.1. Proves `[`/`]`, `1/2/3` jumps, Escape/pop, config
> reload, and startup **never** return to a hidden `Files` context. Stub contexts
> only — no GitHub data yet.

- [ ] 1.1 Add an `IsReviewMode` flag to `GuiRepoState` (set from
  `StartArgs.ReviewTarget`), and expose a `gui.sideWindowNames()` method that
  returns `["prList","prContent","prActivity"]` in review mode, else the normal
  set. Thread `IsReviewMode` into `WindowArrangementArgs`
  (`window_arrangement_helper.go`).
- [ ] 1.2 Update `SideWindowNames`/`SideWindows` (`window_helper.go:145`) and
  `isSideWindowVisible` + `defaultSideContext` (`context_config.go:18,27-42`) to
  honor review mode so the default/fallback side context is `prList`, not `Files`.
- [ ] 1.3 Fix focus paths that hard-code Files: `layout.go:17`, `context.go:254`,
  `context.go:281`, and guard `window_helper.go GetViewNameForWindow` (~`:31-35`)
  so an unmapped/suppressed window never panics or refocuses `Files`.
- [ ] 1.4 Declare three new views `prList`, `prContent`, `prActivity` in
  `pkg/gui/views.go`, and add their `[1]/[2]/[3]` title-prefix/jump-label entries
  to the panel-number map (`views.go:231-255`).
- [ ] 1.5 Create stub contexts (`pkg/gui/context/`) for the three windows, each
  with the correct `WindowName`, registered in `pkg/gui/context/setup.go`. Move
  the existing `PrReviewContext` off the `"files"` window
  (`pr_review_context.go:114`) onto `prContent`.
- [ ] 1.6 Add `viewTabMap()` (`gui.go:877`) entries (kept cheap/static,
  model-backed): `prContent → [Overview, Files Changed]`,
  `prActivity → [Conversation, Checks, Commits]`, `prList → [<sections>]` (a
  single stub section for now). Ensure tab-click bindings register in the
  keybinding reset (`keybindings.go:368-376`).
- [ ] 1.7 Gate the normal default refresh set (`refresh_helper.go:86-104`, status
  `:219`) off in review mode so Files/Branches/Commits/Stash work doesn't run/race,
  and **suppress the Status window** in review mode (D1.7) — it is not in
  `sideWindowNames()` for review mode.
- [ ] 1.8 Integration test: boot in review mode → assert the three review windows
  exist, `]` cycles only review tabs, `2`/`3` jump to the review windows, and
  `Esc` never focuses a normal window. `make generate` to register.

## 2. Local-ref read model (fetch base/head, pager preview — no commenting)

> The DW2 read surface. Validate identity, fetch safely, preview through delta.

- [ ] 2.1 Harden the boot guard (`app.go:184-199`) — **project-agnostic, fail-fast**
  (D1.2, D1.5): derive `OWNER/REPO` + checkout path at runtime (no hardcoded paths);
  verify the checkout's remotes resolve to the launched `OWNER/REPO` (via
  `gh repo view`/remote URL match); fail fast with a clear, actionable message on
  mismatch, on `gh` missing/unauthenticated, or on initial-load failure — never hang
  on "Loading…". See `docs/review-fetch-command.md`.
- [ ] 2.2 Extend the review read query (`github_review.go:34-52`) to return the
  **base OID**, **base repo**, and **head repo URL** (for fork heads), alongside
  `headRefOid`.
- [ ] 2.3 Add a dedicated review-fetch git command (new func in
  `git_commands`, not the generic `sync.go:57`):
  `git fetch <repo-url> <oid>:refs/lazygit-review/<pr>/{head,base}` with
  `--no-write-fetch-head`, no checkout, no branch writes. Fetch both base and head
  by explicit URL+OID; verify `git merge-base` succeeds.
- [ ] 2.4 Exclude `refs/lazygit-review/*` from normal `--all` loaders
  (`commit_loader.go:589-603`) and prune stale review refs on boot.
- [ ] 2.5 Files-Changed tree: populate the `prContent`/Files-Changed `filetree`
  from `git diff --name-status base...head` (statuses → tree node decorations).
- [ ] 2.6 Hover preview: render the selected file's diff via
  `WorkingTree.ShowFileDiffCmdObj(base, head, …, path)` (`working_tree.go:439`)
  through the pty/pager path (`pty.go:46`) → delta. A directory node passes the
  dir path → aggregate diff (fixes #1). Async (`OnWorker`) + loading state.
- [ ] 2.7 Integration test (fixture/seam): selecting a file shows a delta-rendered
  diff; selecting a folder shows the aggregate (not "no changed files").
- [ ] 2.8 Render cap (D1.7): cap the files tree and the diff preview at ≤10 files /
  2000 lines; beyond the cap render a "truncated — N more" note instead of streaming
  the whole diff/tree (guards the large-PR hang). Add the empty-PR state too.

## 3. Parsed-diff selection surface (line-addressable comments)

> Keep the existing row-map context; treat pager output as display-only.

- [ ] 3.1 Keep `pr_review_diff_context.go`/`pr_review_diff.go` as the focusable
  selection surface, but source its diff from a **plain parsed** diff
  (`git diff --no-color base...head -- path` or the API patch), parallel to the
  pager preview. Do **not** reuse `StagingController` (`staging_controller.go:204`
  mutates the index).
- [ ] 3.2 Map the selected `[start,end]` rows → GitHub `path/side/line/start_line`
  using the parsed diff's new-file line numbers; reject non-RIGHT selections in
  v1 (carried from `pr-review-mode` D5).
- [ ] 3.3 Unit tests asserting local-diff line mapping matches the GitHub
  `pulls/{n}/files` patch coordinates (renames/`--no-renames` divergence guard).
- [ ] 3.4 `c` posts a standalone comment via `gh api` with the live `headRefOid`;
  refresh threads on success; toast on 4xx/5xx (never silently drop).
- [ ] 3.5 Per-context keybindings (consistency): in the tree, `space` = toggle
  viewed, `enter` on a file = open the diff surface, `enter` on a directory =
  collapse/expand. In the diff surface, `space` / `Shift`+↑↓ = range-select,
  `c` = comment, `Esc` = back. Add i18n strings; `make generate` for cheatsheets.

## 4. Persisted, content-invalidated viewed state

> Fixes bug #2 + ask 3.3.2 ("discard if a new commit changes the file").

- [ ] 4.1 Add a status-aware viewed-state field to `AppState`
  (`app_config.go:697`): keyed `{repo, pr, path, status, previousPath?,
  fileOid|patchHash}`. Read via `GetAppState`, write via `SaveAppState` on toggle.
- [ ] 4.2 On load, show a file as viewed only if its current `fileOid`/patchHash
  matches the stored one (auto-clears when a new commit changes the file). Handle
  deleted/renamed/binary files per the status-aware key.
- [ ] 4.3 `space` toggles viewed on the selected file with a distinct indicator;
  prune merged/closed-PR entries opportunistically.
- [ ] 4.4 Integration test: mark viewed → leave → re-enter shows viewed; a changed
  blob shows not-viewed.

## 5. PR commits as a separate scoped context (window `[3]` Commits)

> Ask 3.4.3 — only `base..head` commits; do not clobber the normal commits panel.

- [ ] 5.1 New scoped PR-commits model/context (not `LocalCommitsContext`, which is
  hardwired to `Model().Commits`/branch/rebase state —
  `local_commits_context.go:28`). Load via `CommitLoader` with `RefName: base..head`.
- [ ] 5.2 Hover shows the commit message; `enter` drills into that commit's
  changed files → diff in the main panel (reuse commit-files rendering, scoped).
- [ ] 5.3 Integration test: the Commits tab lists exactly `base..head`, drill-in
  shows that commit's files.

## 6. Conversation display (DEFERRED — fast-follow)

> Ask 3.4.1 display half (reviewers, states, threads, PR-body comments).

- [ ] 6.1 `prActivity`/Conversation lists reviewers + state icons (pending /
  commented / approved / changes-requested); null/team reviewer safe.
- [ ] 6.2 Selecting a reviewer shows their review body + thread comments with
  per-thread resolved/unresolved/outdated state, plus PR-body issue comments;
  bodies via glow.

## 7. Comment submission — grouped review (DEFERRED — fast-follow)

> Ask 3.4.1 write half. Standalone posting already lands in P3.4.

- [ ] 7.1 Accumulate per-file pending comments in a session and submit one PR
  review (`gh api … pulls/{n}/reviews` with `comments[]` + body + event
  comment/approve/request-changes).

## 8. gh-dash PR list + checks tree (DEFERRED — last)

> Asks 3.2 and 3.4.2 — the two largest sub-features; ship value before these.

- [ ] 8.1 Parse `~/.config/gh-dash/config.yml` `prSections[]` (honor
  `GH_DASH_CONFIG`/XDG; default set when absent). Feed sections as data to the
  `prList` window (not inside `viewTabMap()`); run each `filters` via
  `gh search prs --json`. Selecting a PR loads it (triggers the P2 fetch).
- [ ] 8.2 Checks tree: normalize `gh api …/check-runs` + workflow `…/jobs` into one
  `filetree` node model (parent jobs, child steps). Sort by importance
  (failure › cancelled › interrupted › pending › success › skipped) then
  last-updated desc; skipped last. `enter` renders logs via `gh run view <id>
  --log` with `FORCE_COLOR=1`.
- [ ] 8.3 (optional) Focus-inverts-split (80/20 ↔ 20/80) via the review layout
  weights in `window_arrangement_helper.go`; side-by-side via `delta
  --side-by-side` toggle (no bespoke renderer).

## Validation (every phase)

- [ ] V.1 `make format`, `make lint`, `make unit-test` green; each commit compiles.
- [ ] V.2 `openspec validate pr-review-workspace --strict` stays green.
- [ ] V.3 `make generate` after any new keybinding/i18n/test so cheatsheets +
  `test_list.go` stay in sync.
