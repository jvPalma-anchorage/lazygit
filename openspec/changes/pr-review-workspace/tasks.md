# Tasks

Dependency-ordered, following the codex+copilot-agreed phasing. **Phases 1–6 are
done** (plus the post-6 fixes below). **Phases 7–12 are the live backlog**, driven
by real-PR dogfooding (`anchorlabsinc/anchorage`): performance/cache (P9) and the
default-tab fix (P10) are the highest-pain items. Each phase must compile and stay
green on its own (AGENTS.md).

Session-1 decisions (design.md D1.1–D1.7) are folded in: local-ref spine,
**project-agnostic** identity (no hardcoded paths), prune-on-boot, blob-OID
viewed-state, fail-fast boot. The original ≤10-file render cap (D1.7) was
**removed** in post-6 work — the tree shows every file (blob OIDs resolve in one
batched `ls-tree`); only the diff render is bounded, by the pager's lazy
streaming. Data shapes and contracts live in `docs/data-shapes.md`,
`docs/review-fetch-command.md`, `docs/viewed-state-schema.md`, and
`docs/REFERENCES.md` — read those before implementing the relevant task.

## Post-6 fixes (landed unplanned, from real-PR dogfooding — all done)

- [x] P6.a Remove the 10-file tree cap; batch blob-OID resolution via one
  `git ls-tree` per side (`ReviewFileBlobOids`); tree is never truncated.
- [x] P6.b Auto-hide generated files (lockfiles, `*.generated.*`, `*.pb.go`,
  `*.snap`, minified) with `G` toggle + hidden-count subtitle.
- [x] P6.c Left/Right arrows cycle review side panels (attach
  `SideWindowController` to the review contexts).
- [x] P6.d Reviewer-state fix: PENDING reviewer with any inline/PR-body comment
  upgrades to COMMENTED (`upgradePendingCommenters`).
- [x] P6.e Folder diff splits unviewed (main) / viewed (secondary), mirroring
  the unstaged/staged split.
- [x] P6.f Overview tab v1: title+number, state+author, `base ← head`,
  description (`RenderPrOverview` + `PrOverviewController`); query now captures
  author + branch names.

## 1. Layout foundation (review-mode owns its windows/tabs/panel numbers)

> Fixes bug #3 / ask 3.1. Proves `[`/`]`, `1/2/3` jumps, Escape/pop, config
> reload, and startup **never** return to a hidden `Files` context. Stub contexts
> only — no GitHub data yet.

- [x] 1.1 Add an `IsReviewMode` flag to `GuiRepoState` (set from
  `StartArgs.ReviewTarget`), and expose a `gui.sideWindowNames()` method that
  returns `["prList","prContent","prActivity"]` in review mode, else the normal
  set. Thread `IsReviewMode` into `WindowArrangementArgs`
  (`window_arrangement_helper.go`).
- [x] 1.2 Update `SideWindowNames`/`SideWindows` (`window_helper.go:145`) and
  `isSideWindowVisible` + `defaultSideContext` (`context_config.go:18,27-42`) to
  honor review mode so the default/fallback side context is `prList`, not `Files`.
- [x] 1.3 Fix focus paths that hard-code Files: `layout.go:17`, `context.go:254`,
  `context.go:281`, and guard `window_helper.go GetViewNameForWindow` (~`:31-35`)
  so an unmapped/suppressed window never panics or refocuses `Files`.
- [x] 1.4 Declare three new views `prList`, `prContent`, `prActivity` in
  `pkg/gui/views.go`, and add their `[1]/[2]/[3]` title-prefix/jump-label entries
  to the panel-number map (`views.go:231-255`).
- [x] 1.5 Create stub contexts (`pkg/gui/context/`) for the three windows, each
  with the correct `WindowName`, registered in `pkg/gui/context/setup.go`. Move
  the existing `PrReviewContext` off the `"files"` window
  (`pr_review_context.go:114`) onto `prContent`.
- [x] 1.6 Add `viewTabMap()` (`gui.go:877`) entries (kept cheap/static,
  model-backed): `prContent → [Overview, Files Changed]`,
  `prActivity → [Conversation, Checks, Commits]`, `prList → [<sections>]` (a
  single stub section for now). Ensure tab-click bindings register in the
  keybinding reset (`keybindings.go:368-376`).
- [x] 1.7 Gate the normal default refresh set (`refresh_helper.go:86-104`, status
  `:219`) off in review mode so Files/Branches/Commits/Stash work doesn't run/race,
  and **suppress the Status window** in review mode (D1.7) — it is not in
  `sideWindowNames()` for review mode.
- [x] 1.8 Integration test: boot in review mode → assert the three review windows
  exist, `]` cycles only review tabs, `2`/`3` jump to the review windows, and
  `Esc` never focuses a normal window. `make generate` to register.

## 2. Local-ref read model (fetch base/head, pager preview — no commenting)

> The DW2 read surface. Validate identity, fetch safely, preview through delta.

- [x] 2.1 Harden the boot guard (`app.go:184-199`) — **project-agnostic, fail-fast**
  (D1.2, D1.5): derive `OWNER/REPO` + checkout path at runtime (no hardcoded paths);
  verify the checkout's remotes resolve to the launched `OWNER/REPO` (via
  `gh repo view`/remote URL match); fail fast with a clear, actionable message on
  mismatch, on `gh` missing/unauthenticated, or on initial-load failure — never hang
  on "Loading…". See `docs/review-fetch-command.md`.
- [x] 2.2 Extend the review read query (`github_review.go:34-52`) to return the
  **base OID**, **base repo**, and **head repo URL** (for fork heads), alongside
  `headRefOid`.
- [x] 2.3 Add a dedicated review-fetch git command (new func in
  `git_commands`, not the generic `sync.go:57`):
  `git fetch <repo-url> <oid>:refs/lazygit-review/<pr>/{head,base}` with
  `--no-write-fetch-head`, no checkout, no branch writes. Fetch both base and head
  by explicit URL+OID; verify `git merge-base` succeeds.
- [x] 2.4 Exclude `refs/lazygit-review/*` from normal `--all` loaders
  (`commit_loader.go:589-603`) and prune stale review refs on boot.
- [x] 2.5 Files-Changed tree: populate the `prContent`/Files-Changed `filetree`
  from `git diff --name-status base...head` (statuses → tree node decorations).
- [x] 2.6 Hover preview: render the selected file's diff via
  `WorkingTree.ShowFileDiffCmdObj(base, head, …, path)` (`working_tree.go:439`)
  through the pty/pager path (`pty.go:46`) → delta. A directory node passes the
  dir path → aggregate diff (fixes #1). Async (`OnWorker`) + loading state.
- [x] 2.7 Integration test (fixture/seam): selecting a file shows a delta-rendered
  diff; selecting a folder shows the aggregate (not "no changed files").
- [x] 2.8 ~~Render cap~~ **Superseded by P6.a** (cap removed; tree never
  truncated). Original: cap the files tree at ≤10 files and surface the hidden
  count as a subtitle; add the empty-PR state. The diff preview keeps full delta
  rendering via lazygit's incremental pager streaming (which bounds each render and
  never stalls) rather than a hard 2000-line truncation — Session-2 decision: keep
  delta, lazy streaming is the line guard (a hard cap would sacrifice delta).

## 3. Parsed-diff selection surface (line-addressable comments)

> Keep the existing row-map context; treat pager output as display-only.

- [x] 3.1 Keep `pr_review_diff_context.go`/`pr_review_diff.go` as the focusable
  selection surface, but source its diff from a **plain parsed** diff
  (`git diff --no-color base...head -- path` or the API patch), parallel to the
  pager preview. Do **not** reuse `StagingController` (`staging_controller.go:204`
  mutates the index).
- [x] 3.2 Map the selected `[start,end]` rows → GitHub `path/side/line/start_line`
  using the parsed diff's new-file line numbers; reject non-RIGHT selections in
  v1 (carried from `pr-review-mode` D5).
- [x] 3.3 Unit tests asserting local-diff line mapping matches the GitHub
  `pulls/{n}/files` patch coordinates (renames/`--no-renames` divergence guard).
- [x] 3.4 `c` posts a standalone comment via `gh api` with the live `headRefOid`;
  refresh threads on success; toast on 4xx/5xx (never silently drop).
- [x] 3.5 Per-context keybindings (consistency): in the tree, `space` = toggle
  viewed, `enter` on a file = open the diff surface, `enter` on a directory =
  collapse/expand. In the diff surface, `space` / `Shift`+↑↓ = range-select,
  `c` = comment, `Esc` = back. Add i18n strings; `make generate` for cheatsheets.

## 4. Persisted, content-invalidated viewed state

> Fixes bug #2 + ask 3.3.2 ("discard if a new commit changes the file").

- [x] 4.1 Add a status-aware viewed-state field to `AppState`
  (`app_config.go:697`): keyed `{repo, pr, path, status, previousPath?,
  fileOid|patchHash}`. Read via `GetAppState`, write via `SaveAppState` on toggle.
- [x] 4.2 On load, show a file as viewed only if its current `fileOid`/patchHash
  matches the stored one (auto-clears when a new commit changes the file). Handle
  deleted/renamed/binary files per the status-aware key.
- [x] 4.3 `space` toggles viewed on the selected file with a distinct indicator;
  prune merged/closed-PR entries opportunistically.
- [x] 4.4 Integration test: mark viewed → leave → re-enter shows viewed; a changed
  blob shows not-viewed.

## 5. PR commits as a separate scoped context (window `[3]` Commits)

> Ask 3.4.3 — only `base..head` commits; do not clobber the normal commits panel.

- [x] 5.1 New scoped PR-commits model/context (not `LocalCommitsContext`, which is
  hardwired to `Model().Commits`/branch/rebase state —
  `local_commits_context.go:28`). Load via `CommitLoader` with `RefName: base..head`.
- [x] 5.2 Hover shows the commit message; `enter` drills into that commit's
  changed files → diff in the main panel (reuse commit-files rendering, scoped).
- [x] 5.3 Integration test: the Commits tab lists exactly `base..head`, drill-in
  shows that commit's files.

## 6. Conversation display (DONE)

> Ask 3.4.1 display half (reviewers, states, threads, PR-body comments).

- [x] 6.1 `prActivity`/Conversation lists reviewers + state icons (pending /
  commented / approved / changes-requested); null/team reviewer safe.
- [x] 6.2 Selecting a reviewer shows their review body + thread comments with
  per-thread resolved/unresolved/outdated state, plus PR-body issue comments;
  bodies via glow.

## 7. Comment submission — grouped review (BACKLOG)

> Ask 3.4.1 write half. Standalone posting already lands in P3.4.

- [x] 7.1 Accumulate per-file pending comments in a session and submit one PR
  review (`gh api … pulls/{n}/reviews` with `comments[]` + body + event
  comment/approve/request-changes).

## 8. gh-dash PR list + checks tree (BACKLOG — panel-1 + Checks tab)

> Asks 3.2 and 3.4.2 — the two largest sub-features; ship value before these.
> Re-flagged in dogfooding (2026-07): panel-1 has NO logic and Checks is NOT
> connected; 8.0 gives panel-1 minimal value before the full gh-dash list.

- [x] 8.0 Minimal PR list: window `[1]` shows the launched PR as a selected row
  (number, title, state) instead of an empty stub; selecting it is a no-op (it is
  already loaded). Unblocks the spec's "PR list window shows the launched pull
  request" requirement without waiting for gh-dash sections.
- [x] 8.1 Parse `~/.config/gh-dash/config.yml` `prSections[]` (honor
  `GH_DASH_CONFIG`/XDG; default set when absent). Feed sections as data to the
  `prList` window (not inside `viewTabMap()`); run each `filters` via
  `gh search prs --json`. Selecting a PR loads it (triggers the P2 fetch).
- [x] 8.2 Checks tree: normalize `gh api …/check-runs` + workflow `…/jobs` into one
  `filetree` node model (parent jobs, child steps). Sort by importance
  (failure › cancelled › interrupted › pending › success › skipped) then
  last-updated desc; skipped last. `enter` renders logs via `gh run view <id>
  --log` with `FORCE_COLOR=1`.
- [ ] 8.3 (optional) Focus-inverts-split (80/20 ↔ 20/80) via the review layout
  weights in `window_arrangement_helper.go`; side-by-side via `delta
  --side-by-side` toggle (no bespoke renderer).

## 9. Per-PR filesystem cache + boot performance

> Dogfooding: boot on `anchorlabsinc/anchorage` (huge monorepo) is painfully slow
> even for a small PR. Diagnosis: boot is fully serial — `gh api graphql` →
> `git fetch <url> <oid>` for base+head (re-fetched EVERY boot, even when
> `refs/lazygit-review/<pr>/*` already point at the right OIDs; URL-fetch forces
> full negotiation against the monorepo) → merge-base → name-status diff →
> 2× ls-tree → per-file fallback diffs. The cache is the architecture fix; the
> ref-fetch skip is the single biggest win.

- [x] 9.1 Cache store: snapshot files under
  `<lazygit-config-dir>/prReview/{owner}/{repo}/{number}/` (resolve via lazygit's
  config-dir helper, not a hardcoded `~/.config`), one JSON file per data type —
  `meta.json` (PR data + conversation), `files.json` (changed files + blob OIDs),
  `checks.json` (reserved for P8.2) — each wrapped in
  `{fetchedAt, headRefOid, payload}`. Atomic write (temp+rename); corrupt/missing
  snapshots are treated as cache-miss, never an error.
- [x] 9.2 Ref-fetch skip: before fetching, compare
  `git rev-parse refs/lazygit-review/<pr>/{base,head}` to the target OIDs from the
  read query; skip the network fetch when they already match. Log timings around
  each boot stage so before/after is measurable.
- [x] 9.3 Warm boot: when snapshots exist, render the whole workspace from cache
  immediately (no network on the render path), then revalidate in the background —
  one cheap `gh api graphql` for `updatedAt`/`headRefOid`; re-fetch only the data
  types whose upstream changed and rewrite their snapshots. `R` (refresh) bypasses
  the cache entirely.
- [x] 9.4 Tests: unit tests for the snapshot store (roundtrip, corrupt file =
  miss, atomicity); integration test proving a warm boot renders with zero
  `gh`/network invocations (fixture-seeded cache) and that refresh rewrites
  snapshots.

## 10. Activity default tab + reviewer ordering

> Dogfooding: panel `[3]` visually defaults to Commits. ROOT CAUSE FOUND: gocui
> draws overlapping tab views in creation order and routes mouse clicks to the
> topmost; lazygit's convention is that a window's DEFAULT tab view is declared
> LAST in `orderedViewNameMappings` (`views.go` — cf. `Tags, Remotes,
> PullRequests, Branches`). Our stack declares `PrConversation, PrChecks,
> PrCommits`, so PrCommits is on top. Focus-based test assertions don't catch
> z-order — the new test must assert the topmost view.

- [x] 10.1 Reorder `orderedViewNameMappings` so each review window's default tab
  view is last in its stack (`PrCommits, PrChecks, PrConversation`; keep
  `PrOverview` before `PrReview` for prContent). Integration test asserts the
  TOPMOST view in `prActivity` on fresh boot is Conversation (z-order, not just
  focus), including after the async PR load completes.
- [x] 10.2 Current-user-first reviewer list: resolve the authenticated login once
  (`gh api user --jq .login`, cached in the P9 cache dir), and sort the
  Conversation reviewer list with the own-login row first (stable order for the
  rest). Unit test the sort; integration test with a fixture reviewer matching
  the seeded login.

## 11. Inline review-thread interaction (reply + resolve)

> Dogfooding: threads render inline in the diff surface but cannot be selected,
> replied to, or resolved.

- [x] 11.1 Make inline threads selectable in the parsed-diff surface: thread
  blocks become navigable anchors (cursor lands on a thread, distinct highlight);
  the row→line map skips thread rows for comment-range selection.
- [x] 11.2 Reply: `enter`/`r` on a selected thread opens the comment editor and
  posts via REST `pulls/{n}/comments` with `in_reply_to`; refresh the thread
  display on success; toast + preserve input on failure.
- [x] 11.3 Resolve/unresolve: a key on a selected thread calls the GraphQL
  `resolveReviewThread` / `unresolveReviewThread` mutation (thread node ID is
  already captured); update the thread's badge in place.
- [x] 11.4 Tests: unit tests for thread-anchor navigation mapping; integration
  test (fixture) for reply posting and resolve toggling through the command seam.

## 12. Overview v2 — labels, heading, timeline

> Dogfooding ask: full overview layout (see spec "PR overview"): heading, state
> line, branches, label chips in GitHub colors, separators, description, then an
> oldest-first timeline (issue comments + submitted review summaries, bots
> included, with timestamps). Bot review comments are currently invisible
> anywhere in the UI — the timeline is where they surface.

- [x] 12.1 Extend the read query + `PullRequestReviewData` with
  `labels(first:20){ name color }` and ensure issue-comment `createdAt` /
  review `submittedAt` are captured (they are) and carried into the timeline.
- [x] 12.2 Renderer: prominent number+title heading (bold/underline — a real H1
  needs glow, use it when available), state+author, `base ← head`, label chips
  (hex → truecolor background via `style.New().SetBg`, readable fg), horizontal
  separators, markdown description, then the merged timeline sorted ascending by
  timestamp with `@author · <time>` headers. Unit tests: layout order, label
  colors, chronological merge incl. bot entries, empty-block omissions.
- [x] 12.3 Integration test: overview shows chips + timeline for a fixture with
  labels, a bot comment, and a human review; assert order (oldest first).

## 13. PR list as a section-tabbed browser (panel-1 redesign)

> Dogfooding: 8.1 rendered gh-dash sections as inline header rows in one list. The
> desired model is a gh-dash-style BROWSER: the sections are the panel's TABS, the
> list shows only the active tab's PRs, hovering a PR previews its Overview in
> panel-0, and panel-1 shrinks out of the way when unfocused. The Overview leaves
> panel-2 entirely (panel-2 becomes just Files Changed).

- [x] 13.1 Sections become panel-1 TABS: PrListContext holds the gh-dash sections +
  an active-section index; the list renders only the active section's PRs. Drive
  `PrList.Tabs`/`TabIndex` dynamically (remove prList from the static `viewTabMap`)
  and bind `[`/`]` in the controller to change the active section. The launched PR
  stays reachable (a "Current" section, or pinned atop every section).
- [x] 13.2 Hover-preview: selecting a PR renders its Overview into panel-0 (main) —
  the same renderer moved from the Overview tab. Fetch the hovered PR's data
  on-select (debounced; served from the per-PR snapshot cache when warm; a light
  header from the list row until the full data arrives) so browsing is responsive
  on a large monorepo.
- [x] 13.3 Keys: `enter` on a PR focuses panel-0 (main) so the user can scroll the
  overview; `Esc` returns to the list. `SPACE` starts the review of that PR
  (Retarget + load panels 2/3). Arrow keys move the selection.
- [x] 13.4 Remove the Overview tab from panel-2: prContent hosts only Files Changed;
  drop `prOverview` from `viewTabMap` and retire its context/controller (the
  renderer now lives behind panel-1's hover-preview).
- [x] 13.5 Accordion: panel-1 collapses to a single line when it is not the focused
  side window (giving panels 2 and 3 the height), and expands when focused —
  special-cased in `window_arrangement_helper.go`.
- [x] 13.6 Integration tests: sections render as tabs and `]` switches them showing
  a different PR set; hovering a PR shows its overview in main; `SPACE` retargets;
  `enter` focuses main; panel-1 is one line when unfocused.

## Validation (every phase)

- [x] V.1 `make format`, `make lint`, `make unit-test` green; each commit compiles.
- [x] V.2 `openspec validate pr-review-workspace --strict` stays green.
- [x] V.3 `make generate` after any new keybinding/i18n/test so cheatsheets +
  `test_list.go` stay in sync.
