## Context

The branches side window is a tabbed window. Tabs are declared in
`gui.viewTabMap()` (`pkg/gui/gui.go:842`) as an ordered list of `TabView{Tab,
ViewName}` per window name. The tab system is purely view-name driven:

- `views.go:274` walks `viewTabMap()` to set each gocui view's `Tabs` slice and
  `TabIndex`.
- `keybindings.go:368` walks the same map to register next/prev-tab keybindings.
- `view_helpers.go:61` reads `viewTabMap()[windowName]` to resolve a window's
  tab set.

Each tab is a full view + context pair: `localBranches`, `remotes`, `tags` are
three `SIDE_CONTEXT`s that all share `WindowName: "branches"` (see
`remotes_context.go:38`). The contexts are constructed once at startup in
`setup.go` and registered in the context tree.

lazygit already has a GitHub PR data path: `Model().PullRequests`
(`[]*models.GithubPullRequest`) is populated by `RefreshHelper.
refreshGithubPullRequests` via a branch-scoped GraphQL query
(`git_commands/github.go:FetchRecentPRs`) and used only to badge local branches.
That data is **branch-scoped** (PRs whose head-ref matches a local branch), not
**user-scoped** (author / review-requested / involved), so it does not satisfy
this feature's content requirement. The model type, however, is reusable.

## Goals / Non-Goals

**Goals:**
- A second tab in the branches window (`Local Branches · Pull Requests ·
  Remotes · Tags`) that renders a list of the current user's relevant PRs.
- Tab visibility gated on `gh` being present on `PATH`; absence is a clean
  no-op (three tabs, no renumber, no broken cycling or focus).
- Reuse `models.GithubPullRequest`; render via a presentation function mirroring
  `GetRemoteListDisplayStrings`.
- POC ships hardcoded data so the UI/gating can be validated independent of
  network/auth.

**Non-Goals (deferred):**
- Live fetching, refresh wiring, on-disk caching.
- Checkout / open / per-PR keybindings.
- GitHub Enterprise host resolution and multi-remote base selection.
- Reacting to `gh` appearing/disappearing mid-session (availability is resolved
  once per session).

## Decisions

### D1 — Gate tab membership inside `viewTabMap()`, computed once
The PR tab is conditionally appended to the `branches` entry of `viewTabMap()`
only when `gh` is available. Because every consumer (tab rendering, keybindings,
window tab-set lookup) reads `viewTabMap()`, gating in one place makes the tab
uniformly present-or-absent with no scattered conditionals.

`exec.LookPath("gh")` must **not** run on every call — `viewTabMap()` is invoked
on layout passes and keybinding setup. Resolve it once at gui startup and store a
`bool` (e.g. on the `Gui` struct, set during `Run`/`onNewRepo`). Rationale:
availability is static for a session; repeated `PATH` scans are wasteful.

*Alternative considered — conditionally construct the context/view at startup
(Approach B):* cleaner "doesn't exist when off" semantics, but the context tree
in `setup.go` uses fixed named struct fields, so a nil/absent context ripples
into every place that dereferences it. Rejected for the POC: more touchpoints,
more nil-guards, no user-visible benefit over D1.

### D2 — Context always exists; only the tab is conditional
Following the `remotes`/`tags` pattern, the `PullRequestsContext` and its
`pullRequests` view are always constructed and registered (`WindowName:
"branches"`, `Kind: SIDE_CONTEXT`, `Focusable: true`). When `gh` is absent the
context simply has no tab pointing at it.

**Focus-safety invariant:** tab cycling only walks `viewTabMap()`, so an absent
tab is unreachable by cycling. The remaining risk is the *default* context for
the `branches` window resolving to the PR context while its tab is hidden. The
implementation MUST verify window→default-context resolution is driven by
`viewTabMap()` (so the default falls back to `localBranches`), and MUST NOT make
the PR context the branches-window default. This is the same focus-leak class
handled in the side-panel-visibility change; treat it with the same care.

### D3 — Hardcoded data lives behind a single seam
The POC data source is one function returning `[]*models.GithubPullRequest`
(e.g. `samplePullRequests()` colocated with the context). The view-model reads
from it exactly where it will later read from `Model().PullRequests`. This keeps
the later live-path swap to a one-line source change.

### D4 — Live path (deferred) will use the `gh` CLI, not hand-rolled GraphQL
The content scope (author / review-requested / involved) maps directly to
`gh search prs --involves @me` / `--author @me` / `--review-requested @me`
(or `gh pr list --search "involves:@me"`), with `--json number,title,state,url,
headRefName,author`. The existing in-repo GraphQL path is branch-scoped and
would have to be substantially rewritten to do a user-scoped search; shelling
out to `gh` is less code and is the reason `gh` is the visibility gate. Recorded
here so the gate choice and the eventual fetch are consistent. Not built in this
POC.

## Risks / Trade-offs

- **Hidden tab still focusable via default-context resolution** → Verify D2's
  invariant during implementation; add a guard/test that the branches-window
  default is `localBranches` when the PR tab is absent.
- **`exec.LookPath` on a hot path** → D1 caches the result once per session; do
  not call it from `viewTabMap()` directly.
- **`gh` present but unauthenticated** → Out of scope for the POC (data is
  hardcoded, so auth is irrelevant). The live path will need a `gh auth status`
  / empty-result fallback; noted for later, not handled now.
- **Tab indices/badges drift** → The existing title-prefix and jump-key logic is
  view-name driven; inserting a tab at index 1 shifts Remotes/Tags. Confirm the
  branches-window badge/prefix loop tolerates a 4th tab and that absence leaves
  it at 3. Covered by the visibility spec scenarios.
- **POC masquerading as done** → The hardcoded data must be obviously a stub
  (clearly fake titles) so it is never mistaken for live PRs.

## Open Questions

- Does any code path set the branches-window *default/active* context directly
  to a tab other than `localBranches`? (Verify in `context_config.go` /
  `view_helpers.go` before wiring — drives whether D2's guard is one line or a
  small refactor.)
- Should the tab title carry a `[n]` jump badge at all, given the branches
  window already owns one jump slot? POC: inherit whatever the existing tab
  mechanism does for siblings; do not add new jump-key behavior.
