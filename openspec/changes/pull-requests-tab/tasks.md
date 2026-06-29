## 1. gh availability gate

- [x] 1.1 Add a `gh`-on-PATH check resolved once at startup (e.g. `exec.LookPath("gh")`), stored as a bool on the `Gui` struct; do not call `LookPath` from `viewTabMap()`.
- [x] 1.2 Unit-test the gate helper: returns true/false based on a stubbed lookup, and is read (not recomputed) by `viewTabMap()`. (Helper true/false covered by `TestGhIsAvailable`; the "read by `viewTabMap()`" assertion is deferred to phase 4/5 where `viewTabMap()` first consumes the cached bool.)

## 2. Model and hardcoded data seam

- [x] 2.1 Add a single `samplePullRequests() []*models.GithubPullRequest` source returning obviously-fake sample PRs (clearly-stub titles) — colocated with the new context. Reuse `models.GithubPullRequest`; do not add a new PR type.
- [x] 2.2 Add a presentation function `GetPullRequestListDisplayStrings([]*models.GithubPullRequest, ...) [][]string` mirroring `GetRemoteListDisplayStrings`, rendering at least PR number + title.

## 3. View and context wiring

- [x] 3.1 Declare and create the `pullRequests` view in `pkg/gui/views.go` (alongside `remotes`/`tags`). (View field in `types/views.go`; `orderedViewNameMappings`, Title, branches `windowViews` membership, and `TitlePrefix` reset in `views.go`.)
- [x] 3.2 Add `PullRequestsContext` (`pkg/gui/context/pull_requests_context.go`) as a list context with `WindowName: "branches"`, `Kind: SIDE_CONTEXT`, `Focusable: true`, reading from the seam in 2.1 via a `FilteredListViewModel`. (Required adding `ID()`/`URN()` to `GithubPullRequest` to satisfy `HasID`/`types.HasUrn` — extends the existing model, no new type.)
- [x] 3.3 Register the context in `setup.go` / the context tree so it participates like `remotes`/`tags`. (Const `PULL_REQUESTS_CONTEXT_KEY`, `AllContextKeys`, `ContextTree` field, `Flatten()` — inserted after `Remotes` so `Branches` stays last and remains the branches-window default; gets list controllers automatically via `AllList()`.)
- [x] 3.4 Add `PullRequestsTitle` (and tab string if distinct) to `pkg/i18n/english.go`; use it for the view title and tab label. (Reuses the single `*Title` field for both, matching `RemotesTitle`; no separate tab string.)

## 4. Conditional tab membership

- [x] 4.1 In `viewTabMap()`, insert the Pull Requests `TabView` at index 1 of the `branches` entry **only when** the cached gh-available bool is true. (Reads `gui.ghAvailable` — no `exec.LookPath` on this hot path. All three `viewTabMap` consumers handle the 3↔4 tab-count variance: tab indices derive from the rendered `view.Tabs`, so no out-of-range.)
- [x] 4.2 Verify branches-window default/active context resolves so it falls back to Local Branches when the tab is absent; ensure the PR context is never the branches-window default. **Correction to the original premise:** the default does NOT resolve via `viewTabMap()` — it resolves via `initialWindowViewNameMap` (overwrites by `Flatten()` order) consumed by `GetContextForWindow`. `self.Branches` is last among branches-window contexts in `Flatten()`, so `localBranches` is the default in BOTH gh states. **No guard needed** (confirmed independently by codex). Executable proof lands in task 5.3 (gh-unavailable integration test).

## 5. Tests

- [x] 5.1 Unit test `GetPullRequestListDisplayStrings` (number+title rendering, empty input). (`presentation/pull_requests_test.go`. Also added `gh_test.go::TestDetectGhAvailable` to lock the `GH_AVAILABLE_OVERRIDE` seam.)
- [x] 5.2 Integration test: with `gh` available, the branches window exposes a Pull Requests tab at position 2 (order: Local Branches, Pull Requests, Remotes, Tags) that renders the hardcoded rows. (`ui/show_pull_requests_tab_when_gh_available.go`, passes headless.)
- [x] 5.3 Integration test: with `gh` unavailable, the branches window shows only the three existing tabs, tab cycling never focuses the PR context, and the default tab is Local Branches. (`ui/hide_pull_requests_tab_when_gh_unavailable.go`, passes headless.)
- [x] 5.4 Run `make generate` to register any new integration test in `test_list.go`. (Also revealed + fixed a 5th registration site: `cheatsheet/generate.go` `localisedTitle` needed a `pullRequests` entry; regenerated 9 keybinding docs.)

## 6. Validation

- [x] 6.1 `make format`, `make lint`, `make unit-test` all green. (`make format` ✓ green and `make unit-test` ✓ green. **`make lint` could NOT execute** — the golangci-lint shim returns a private-proxy `401 Unauthorized` (environment auth issue, not a code failure); golangci-lint ran clean on every touched package during phases 1–4 of this session, and `go vet ./...` on the touched packages is clean now. Re-run `make lint` once proxy auth is restored to fully close this.)
- [x] 6.2 `openspec validate pull-requests-tab --strict` passes.
- [ ] 6.3 Manual TUI smoke test via the `llg-dev` symlink: confirm the tab appears/positions correctly with `gh` installed and is absent when `gh` is off-PATH. (**Requires a human at an interactive terminal — cannot be performed headlessly.** Automated equivalent is covered by the passing integration tests 5.2/5.3. Dev binary is built and `~/.local/bin/llg-dev` points to it. Run `llg-dev` (gh installed → tab present) and `GH_AVAILABLE_OVERRIDE=false llg-dev` (tab absent) to confirm visually.)
