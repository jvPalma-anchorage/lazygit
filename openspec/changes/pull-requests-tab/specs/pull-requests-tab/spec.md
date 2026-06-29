## ADDED Requirements

### Requirement: Pull Requests tab presence is gated on the `gh` CLI

The system SHALL display a "Pull Requests" tab in the branches window only when
the GitHub CLI (`gh`) is available on the user's `PATH`. When `gh` is not
available, the branches window SHALL retain its existing tabs unchanged.

#### Scenario: gh is available

- **WHEN** lazygit starts and `gh` is found on `PATH`
- **THEN** the branches window shows a "Pull Requests" tab in addition to
  Local Branches, Remotes, and Tags

#### Scenario: gh is not available

- **WHEN** lazygit starts and `gh` is not found on `PATH`
- **THEN** the branches window shows exactly the existing tabs (Local Branches,
  Remotes, Tags) with no "Pull Requests" tab and no change to their order or
  numbering

### Requirement: Pull Requests tab is positioned second

When present, the "Pull Requests" tab SHALL appear as the second tab in the
branches window, immediately after Local Branches and before Remotes, yielding
the order: Local Branches, Pull Requests, Remotes, Tags.

#### Scenario: tab ordering with gh available

- **WHEN** the branches window is rendered with `gh` available
- **THEN** the tabs read left-to-right as Local Branches, Pull Requests,
  Remotes, Tags

### Requirement: Pull Requests tab lists the current user's relevant PRs

The "Pull Requests" tab SHALL list pull requests scoped to the current user:
those the user authored, those for which the user's review is requested, and
those in which the user is otherwise involved. In this proof-of-concept the list
SHALL be sourced from a fixed, hardcoded set of sample pull requests rather than
a live query.

#### Scenario: tab renders the hardcoded PR list

- **WHEN** the user focuses the Pull Requests tab
- **THEN** the panel renders one row per pull request from the hardcoded source,
  each showing at least the PR number and title

#### Scenario: empty list renders without error

- **WHEN** the hardcoded source contains no pull requests
- **THEN** the Pull Requests tab renders an empty list without crashing or
  affecting the other tabs

### Requirement: A hidden Pull Requests tab never receives focus

The system SHALL never give focus to the pull-requests context while the "Pull
Requests" tab is absent (because `gh` is unavailable): neither tab cycling nor
default-context resolution may land on it, and the branches window's default tab
SHALL be Local Branches.

#### Scenario: tab cycling skips the absent tab

- **WHEN** `gh` is unavailable and the user cycles tabs within the branches
  window
- **THEN** focus moves only among Local Branches, Remotes, and Tags and never to
  the pull-requests context

#### Scenario: default branches tab is Local Branches

- **WHEN** `gh` is unavailable and the branches window becomes active
- **THEN** the Local Branches tab is the focused tab
