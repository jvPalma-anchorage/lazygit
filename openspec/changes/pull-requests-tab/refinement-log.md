# Refinement Log: pull-requests-tab

## Progress Summary

| Topic | Status | Session |
|-------|--------|---------|
| Change structure (split vs mono) | not-started | #1 (Q1) |
| Live data source / auth / gate | not-started | #1 (Q2) |
| PR detail model shape (two-model split) | not-started | #1 (Q2) |
| Preview pane layout | not-started | #1 (Q3) |
| Preview fetch strategy (debounce/cache) | not-started | #1 (Q4) |
| Review-mode diff source | not-started | #1 (Q5) |
| Reviewed-state lifetime + `space` key | not-started | #1 (Q6) |
| glow rendering + injection safety | not-started | #1 (Q7) |
| Inline comments (item 4) | deferred-candidate | #1 (Q1) |

## Session #1 — 2026-05-29

### Focus
First refinement of a major expansion: the user wants to grow the implemented POC (gh-gated tab, hardcoded sample PRs) into a full in-TUI PR-review experience — (1) live PRs via gh, (2) a rich preview pane, (3) [Enter] → code review with file-marking, (4) future inline diff comments.

### Agent Findings Summary
- **Agent 1 (Spec Completeness)**: all 4 asks have zero spec coverage; "real PRs" directly contradicts the POC's hardcoded requirement (needs MODIFIED); recommends splitting into a sequence of changes; item 4 should be a separate future capability.
- **Agent 2 (Problem & Context)**: data-source contradiction — existing branch-annotation fetch uses hand-rolled GraphQL + go-gh token, NOT the `gh` CLI the POC gate checks; existing create/open-PR flows are browser-only and complement (don't duplicate) an in-TUI view; no "code review" concept exists today.
- **Agent 3 (Requirements Coverage)**: missing error/empty/loading/auth/rate-limit/timeout states; model too thin for preview; lazy-vs-batch fetch, debounce, and cross-repo dedup unspecified; diff-source and reviewed-state storage are blocking.
- **Agent 4 (System & Infrastructure)**: main area is a fixed 2-slot Main/Secondary model — no 3-way split (the biggest constraint); preview "tabs" can't reuse `viewTabMap` (side-window only); commitFiles is the reusable precedent for review mode (transient context borrowing the branches window, `base...head` range); no "mark reviewed" precedent; glow needs PTY/stdin.
- **Agent 5 (Goal & Behaviour)**: network-per-keystroke risk (no cancellation/debounce primitives); token-exposure differs by transport (GraphQL header never logged vs `gh` argv logged unless `.DontLog()`); **shell-injection risk** rendering attacker-controlled PR markdown through glow — must use stdin/argv, never `NewShell`; full state-machine has undefined loading/Esc transitions; force-push staleness of reviewed marks.
- **Agent 6 (Documentation)**: five `docs/*.md` needed (gh-commands, data-shapes, preview-layout, review-mode, glow-rendering); key locked fact — `gh search prs` (cross-repo, user-scoped) lacks `reviewDecision`/`statusCheckRollup`/`headRefName`, forcing a thin-list + lazy-per-PR-detail two-model split; `go-gh` api client and `glamour` are NOT vendored (so shell-out for both gh and glow); gh 2.92.0, glow 2.1.1 confirmed installed.

### Topics Discussed
(none resolved yet — questions written, awaiting answers)

### Questions Asked
7 questions in `questions-session1.md` (5 critical: Q1 split, Q2 transport/gate, Q3 layout, Q4 fetch-strategy, Q5 diff-source; 2 important: Q6 reviewed-state/key, Q7 glow/security). The thinner model-split decision and refresh/concurrency separation are folded into Q2/Q4 as forced consequences rather than standalone questions.

### Files Updated
- `questions-session1.md` — created (7 questions)
- `refinement-log.md` — created (this entry)

### Gap Analysis
- Session start: 4 expansion areas, 0% specified (~45 raw findings, ~14 critical).
- Session end: 0 decisions applied yet; 7 decisions framed and awaiting answers. Re-run after answering to apply.
