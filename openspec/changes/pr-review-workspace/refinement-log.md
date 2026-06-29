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
