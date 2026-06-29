## Why

lazygit can already annotate local branches with their GitHub PR status, but it
gives you no way to see PRs that aren't tied to one of your local branches —
the ones you authored, were asked to review, or are otherwise involved in. Today
you leave the TUI and open a browser (or `gh`) to answer "what needs my
attention?". This change brings that list inside lazygit as a first tab in the
branches window.

This is an explicit **proof of concept**: prove the tab can exist, render, and
gate its own visibility on tooling availability before investing in a polished
live data path.

## What Changes

- Add a new **Pull Requests** tab to the branches side window, positioned
  **second** — between `Local Branches` and `Remotes` (order becomes
  `Local Branches · Pull Requests · Remotes · Tags`).
- The tab renders a list of pull requests scoped to the current user: those
  where the user is the **author**, has a **review requested**, or is otherwise
  **involved**.
- **POC data is hardcoded.** The first cut ships a fixed, in-memory list of
  sample PRs so the tab, navigation, rendering, and visibility gating can be
  validated without a network/auth dependency.
- **Visibility is gated on the `gh` CLI.** The tab only appears when the GitHub
  CLI (`gh`) is found on `PATH`. When `gh` is absent the branches window keeps
  its current three tabs and nothing renumbers.
- Reuse the existing `models.GithubPullRequest` model rather than inventing a
  new PR type.

Out of scope for this POC (deferred): live `gh`/GraphQL fetching, refresh
wiring, caching, opening/checking-out a PR, per-PR keybindings, and GitHub
Enterprise host handling. The proposal deliberately stops at "tab exists, gated,
renders a list".

## Capabilities

### New Capabilities
- `pull-requests-tab`: a tab in the branches window that lists the current
  user's relevant pull requests (author / review-requested / involved), present
  only when the `gh` CLI is available, rendering from a hardcoded source in this
  POC.

### Modified Capabilities
<!-- None. The existing branch-level PR annotation behavior is untouched; no
     existing spec captures the branches-window tab set, so this is purely
     additive. -->

## Impact

- **New files**: a `PullRequestsContext` (`pkg/gui/context/`), its view wiring,
  presentation display-strings, and a small `gh`-availability check.
- **Touched files**:
  - `pkg/gui/gui.go` — `viewTabMap()`: insert the PRs tab at index 1 of the
    `branches` window (conditionally, when `gh` is present).
  - `pkg/gui/views.go` — declare/create the `pullRequests` view; tab title
    prefix bookkeeping.
  - `pkg/gui/context/setup.go` + context registration — add the new context to
    the tree as a `SIDE_CONTEXT` with `WindowName: "branches"`.
  - `pkg/i18n/english.go` — `PullRequestsTitle` (and any tab string).
- **Interactions to respect**: the tab system keys tabs by view name within a
  window (`viewTabMap`, `views.go:274`, `keybindings.go:368`); a
  conditionally-absent tab must not break tab cycling, the title-prefix loop, or
  focus. This mirrors the care already taken in the side-panel-visibility work.
- **Dependencies**: no new Go modules — `gh` is an external binary detected via
  `exec.LookPath`; `cli/go-gh` (already vendored) remains available for the
  later live path.
- **No CI/pipeline changes.**
