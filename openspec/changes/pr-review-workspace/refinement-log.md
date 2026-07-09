# Refinement Log: pr-review-workspace

## Progress Summary

| Topic | Status | Session |
|-------|--------|---------|
| Diff source (local-ref fetch vs API patch) | approved (A) | #1 |
| Relationship to gh-dash `L` flow | approved (A, project-agnostic) | #1 |
| Ref + viewed-state cleanup policy | approved (A) | #1 |
| Viewed-state identity (binary/deleted/renamed) | approved (A) | #1 |
| gh/auth/network failure UX | approved (A) | #1 |
| v1 scope + deferred-requirement representation | approved (A) | #1 |
| Status window + large-PR caps | approved (A, ≤10 files/2000 lines) | #1 |
| Missing v1 line-comment-surface requirement | applied | #1 |
| Repo-identity verification requirement | applied | #1 |
| Data-shape docs (port from pr-review-mode) | applied | #1 |
| pr-review-mode supersession formalization | applied | #1 |
| Per-context keybindings + initial focus scenarios | applied | #1 |

## Session #1 — 2026-06-29

### Focus
First refinement cycle on a freshly-authored change (proposal/design/spec/tasks already
drafted and critiqued by codex+copilot). Goal: surface remaining gaps via the 6-agent
sweep and separate genuine user-decisions from self-resolvable fixes.

### Agent Findings Summary
- **Agent 1 (Spec Completeness)**: 3 critical, 5 important, 4 minor. Headline: the v1
  line-addressable comment surface + standalone posting (phase 3) is entirely absent from
  spec; deferred phases (6–8) appear as full requirements with no DEFERRED marker; PR
  overview under-specified (1 scenario); pr-review-mode supersession not formalized.
- **Agent 2 (Problem & Context)**: All 4 bugs verified at file:line
  (`pr_review_context.go:114` WindowName="files" = tab-escape cause;
  `:283` folder→no-files; `:54` in-memory reviewed map; `pr_review_diff.go:156` bypasses
  pager). No real conflict with the pull_requests POC. gh/auth/pager assumptions not
  validated. Critique reconciliation correctly folded in.
- **Agent 3 (Requirements Coverage)**: Missing failure scenarios (gh unavailable/unauth,
  merge-base fail, repo-identity mismatch, comment 4xx, gh-dash config absent, fork
  inaccessible, merged/closed PR); edge cases (binary/renamed/deleted viewed-state, empty
  PR, huge PR, outdated threads); ref lifecycle + AppState growth undefined; spec overstates
  v1 scope.
- **Agent 4 (System & Infrastructure)**: 23 exact symbol/line matches; only 2 minor offsets
  (AddReviewComment→:105, WindowName→:114). New review-fetch command genuinely required;
  AppState extension backward-compatible; yaml.v3 vendored; no existing refs/lazygit* to
  collide with. **No blockers** — implementable as designed.
- **Agent 5 (Goal & Behaviour)**: Keybinding scheme not formalized per-context; async
  loading/error/empty states unspecified (the infinite-Loading precedent); initial focus +
  quit/cleanup unspecified; gh-dash filter injection safety; performance caps; v1/deferred
  mismatch.
- **Agent 6 (Documentation)**: Self-containment test FAILS — no data-shape docs (PR
  metadata, threads, check-runs, diff/patch, base/head OIDs, viewed-state struct, gh-dash
  config, window-context wiring). Rich versions exist in pr-review-mode/docs and should be
  ported/referenced. Critique docs are orphaned (referenced once).

### Topics Discussed

#### Architecture & behaviour decisions [IN_PROGRESS]
- **Severity**: Critical/Important
- **Discussion**: 7 genuine decisions distilled into `questions-session1.md` (diff source,
  `L`-flow relationship, cleanup policy, viewed-state identity, failure UX, v1 scope,
  status/caps).
- **Open items**: awaiting user answers in the questions file.

#### Self-resolvable spec/doc gaps [NOT_STARTED — apply next cycle]
- **Severity**: Critical/Important/Minor
- **Decision**: resolve without asking — add v1 comment-surface requirement; mark deferred
  requirements; add repo-identity + injection-safety scenarios; formalize supersession;
  port data-shape docs from pr-review-mode/docs; fix 2 line-offsets; break task 1.x into
  threading sub-items; add per-context keybinding + initial-focus scenarios from the
  original brief. Several depend on the Q1/Q4/Q5/Q6 answers, so applied on re-run.

### Questions Asked
7 questions in `questions-session1.md` (1 critical, 5 important, 1 minor).

### Files Updated
- `questions-session1.md` — created (7 questions).
- `refinement-log.md` — created (this entry).

### Gap Analysis
- Session start: ~45 raw findings (9 critical).
- Session end (questions phase): 7 open user-decisions + ~12 self-resolvable items.
  No implementation blockers found; the change is structurally sound and code-grounded.

### Answers Processed (re-run)

All 7 questions answered **A**, with two load-bearing notes:
- **Q2 → project-agnostic is now a hard requirement** (D1.2): no hardcoded
  project/path (the old `anc-review` reference is forbidden); repo identity, checkout
  path, and base/head OIDs+URLs are resolved at runtime; `R` replaces `L`.
- **Q1 note**: prefer reusing lazygit/gh's existing mechanisms under the hood.
- **Q7 cap**: render ≤ 10 files / 2000 lines, "truncated — N more" beyond.

Applied:
- `design.md` — added Decisions D1.1–D1.7; marked resolved Open Questions.
- `proposal.md` — added "Works in any repository (project-agnostic)"; `R` replaces `L`.
- `specs/…/spec.md` — added requirements: project-agnostic boot + repo-identity
  verification; v1 line-range comment surface + standalone posting; bounded
  rendering (10/2000); enriched gh-CLI requirement with fail-fast boot + per-panel
  toast + filter-arg safety; added [DEFERRED — Phase N] markers on
  Conversation/Checks/grouped-review/multi-section PR list; added scenarios for
  empty PR, renamed/deleted viewed-state, missing overview fields, initial focus,
  enter-on-directory, comment-post failure.
- `tasks.md` — project-agnostic + fail-fast boot (2.1); Status suppression (1.7);
  render cap + empty-PR (2.8); per-context keybindings (3.5); fixed
  `pr_review_context.go` line-offset (1.5); doc references in the header.
- `docs/` — created `REFERENCES.md`, `data-shapes.md`, `review-fetch-command.md`,
  `viewed-state-schema.md` (ported/cited from `pr-review-mode/docs/`).

Known doc gaps to capture before the relevant phase (flagged by the docs agent):
- check-runs / workflow-jobs JSON is from the public API contract, not a live
  capture (Phase 8 / DW8 — capture before implementing).
- `baseRefOid` + base/head repository URLs are new GraphQL query additions
  (Phase 2 / task 2.2 — verify when added).
- Binary-file `patch` omission documented from GitHub behavior, not captured.

### Status: ready for implementation
All Critical/Important findings resolved. v1 scope locked (phases 1–5). Begin with
`/opsx:apply pr-review-workspace 1.*` (layout foundation).

## Session 3 (2026-07-07) — dogfooding sync (real anchorage PRs)

Reported on `llg-dev anchorlabsinc/anchorage 217274`: horrible boot performance;
inline threads not selectable (no reply/resolve); panel `[3]` visually defaults
to Commits; Overview too thin (wants heading/labels/timeline layout); Checks tab
and panel `[1]` still stubs. User directive: per-PR filesystem cache at
`~/.config/lazygit/prReview/{owner}/{repo}/{number}` with per-data-type
last-updated snapshots.

Diagnoses recorded in design.md:
- **D3.2 (root cause found)**: default-tab defect is gocui z-order —
  `orderedViewNameMappings` must declare the default tab view LAST per window
  stack; ours has PrCommits last → drawn on top + swallows clicks. Focus-based
  tests can't see it; must assert topmost view.
- **D3.1**: boot is fully serial and re-fetches review refs by URL every boot
  even when they already point at the right OIDs — dominant cost on a monorepo.
  Cache + ref-fetch-skip is the fix.

Spec updated: Overview v2 layout requirement (heading/labels/separator/
description/timeline oldest-first incl. bots); Conversation un-deferred + state
of the commenting-reviewer fix; new requirements for default tab + self-first
reviewer row, inline-thread interaction, and the per-PR cache; current-state
notes on the stub PR list; generated-file hiding scenario. Tasks: post-6 fixes
logged as done (P6.a–P6.f); new phases 9 (cache/perf), 10 (default tab +
self-first), 11 (thread reply/resolve), 12 (overview v2); task 8.0 (minimal
launched-PR row) added; 2.8 marked superseded.

## Session 4 (2026-07-07) — full backlog implemented in one pass (offline-worker)

Implemented phases 7–12 end to end, all tested, no git writes, all GitHub write
paths behind seams (nothing left this machine):

- **P10**: z-order reorder in `orderedViewNameMappings` (default tab last per
  review window stack) + `TestReviewTabViewZOrder`; `viewer{login}` captured;
  Conversation lists the authenticated user first.
- **P9**: `ReviewSnapshotStore` (`<configdir>/prReview/{owner}/{repo}/{number}/`,
  atomic, corrupt=miss), warm boot from snapshots (validated against head+base
  OIDs AND the local refs), background revalidation, `R` bypass, ref-fetch skip
  when refs already at wanted OIDs, boot-stage timing logs. Warm-boot integration
  test proves zero-network render.
- **P12**: Overview v2 — `#N - title` heading, state+author, `base ← head`, label
  chips in GitHub colors (luminance-picked fg), separators, description, oldest-
  first timeline (issue comments + review summaries, bots included, timestamps).
  `labels(first:100)` + pageInfo folded into Truncated.
- **P11**: thread blocks are selectable anchors (RowThreadID parallel map; plain
  moves land on threads, range-extends skip them); `c` on a thread replies (REST
  replies endpoint), `t` toggles resolve (GraphQL mutations); failures toast,
  queue/input never dropped.
- **P7**: `C` queues pending comments; `S` submits ONE review (menu:
  comment/approve/request-changes + optional body) via `pulls/{n}/reviews`
  `--input -`; single-line comments omit start_line; queue cleared only on
  success.
- **P8.0/8.1**: PR list window — launched PR under "Current", gh-dash
  `prSections` parsed (env/XDG/home, defaults on error), `gh search prs` per
  section, Enter retargets the whole workspace (`Retarget` resets all per-PR
  state). Sections are list data, not tabs (spec updated to match).
- **P8.2**: checks tree — `gh pr checks --repo` normalized, importance sort
  (fail>cancel>pending>pass>skipping, recency within bucket), workflow parents
  collapse/expand, Enter renders logs via `gh run view --repo … --log` pty with
  FORCE_COLOR.

Multi-agent verification: codex found 4 real issues, 3 fixed (in-flight load vs
Retarget → loadGeneration guard; checks not repo-scoped → --repo everywhere;
checks async apply unguarded → target comparison) and 1 documented as designed
(pending comments survive head-move; 422 surfaces at submit, queue retained).
An earlier codex pass caught the cache base-OID pairing hole and label
truncation — both fixed. antigravity (`agy`) remains unusable headless (needs a
TTY). Full integration suite (entire repo) green in 12s; lint 0 issues.

Debug war story: the PR-list panel initially crashed every review boot with
`index out of range` in the column aligner — display strings must have UNIFORM
column counts across rows (section headers padded with empty cells). The panic
was masked by a `close of closed channel` in the test harness's failure path,
which cost a round of misdirected probing.

### Session 4 addendum — verification-fleet findings, resolutions

An 8-verifier workflow (one adversarial read-only agent per requirement cluster) +
codex reviewed the full diff. Fixed:
- worker-thread `self.data` write eliminated (`loadLocalRefModel` now takes the
  data + PR number as parameters; `applyCachedSnapshots` takes the captured
  target) — no context field is written off the UI thread anymore;
- a failed background revalidation no longer clobbers a successful warm render
  (toast + keep cached view; same for the checks tab);
- checks are now a cached data type (`checks.json`, warm-read + always
  revalidate, freshest-wins) — the warm-boot test seeds and asserts it;
- checks tree grouping made contiguous per workflow (the importance sort could
  interleave workflows and scatter children under wrong parents) via pure
  `groupChecksRows` + unit tests;
- failed reply/comment writes re-open the prompt with the typed body intact
  (spec: "input is not lost");
- label chips validate hex before trusting gookit (6-char garbage colors fall
  back to a plain chip);
- PR list's launched row title refreshes once the PR data arrives;
- test gaps closed: labels-truncation → Truncated, trailing-thread RowThreadID
  tagging, interleaved checks grouping.

Accepted residuals (documented, deliberate):
- `LAZYGIT_PR_REVIEW_OFFLINE` skips revalidation when a warm cache exists — a
  test seam that doubles as an airplane mode; inert unless explicitly set.
- Revalidation always re-runs the GraphQL fetch (the cheap part); only the git
  ref fetch is conditionally skipped. The spec's `updatedAt`-gated per-type
  refetch is a future optimization.
- Body-less submitted reviews (bare approvals) don't appear as timeline entries;
  their state still shows in the reviewer list badges.
- Pending grouped-review comments survive a Reload; if the head moved and a line
  no longer anchors, GitHub's 422 surfaces at submit and the queue is retained.
- copilot review hung with no output and was killed; agy needs a TTY (unusable
  headless). codex + the verifier workflow provided the independent review.

### Session 5 (2026-07-07) — Phase 13: PR list becomes a section-tabbed browser

Dogfooding correction: 8.1 rendered gh-dash sections as inline header rows in one
list; the wanted model is a gh-dash-style browser. Implemented (all tested):
- **13.1** sections are window [1]'s TABS — PrListContext holds sections +
  activeSection, drives `PrList.Tabs`/`TabIndex` dynamically (prList removed from
  the static `viewTabMap` so the layout loop doesn't overwrite them), and the
  controller binds `[` / `]` (which shadow the no-op global tab handler) to cycle
  sections. The launched PR is pinned as a "Current" first section.
- **13.2** hovering a PR previews its Overview in panel-0; the current PR reuses
  the live review data, other PRs fetch on-select (guarded, cached in
  `overviewData`) with a light header until the data lands.
- **13.3** `enter` pushes the main context (scroll the overview), Esc returns;
  `SPACE` retargets the workspace at the PR (loads panels 2/3).
- **13.4** the Overview left panel-2 entirely — prContent now has only Files
  Changed; `prOverview` dropped from `viewTabMap` + the window's view stack (the
  context/controller remain wired but unreachable, harmless).
- **13.5** prList collapses to `Size: 1` in `window_arrangement_helper` when it is
  not the focused side window, expanding to `Weight: 1` when focused.

Rendered frame confirms both the tab bar (`╭─[1]─Current - Needs My Review─╮`) and
the collapsed single-line state when unfocused (`╶─[1]─Pull Requests─╴`). Three
pre-existing tests were updated for the new layout (overview now via panel-1 hover,
prContent single-tab, PR list tabbed). Full unit + entire integration suite green
(13s), lint 0, `openspec --strict` valid, binary rebuilt.

Only remaining task: 8.3 (optional — focus-inverts-split + side-by-side toggle).
