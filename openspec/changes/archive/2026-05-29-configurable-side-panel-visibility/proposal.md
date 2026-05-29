## Why

The left-hand side panels (Status, Files, Branches, Commits, Stash) are fixed:
every user gets all five regardless of whether they use them. Users who never
look at the Status, Commits, or Stash panel pay for them in vertical screen
space that could go to the panels they actually use. There is no way to opt out
of a panel today — only transient mechanisms (screen modes, accordion) that
don't persist a deliberate "I never want this panel" choice.

## What Changes

- Add three user-config booleans under `gui`, all defaulting to `true`:
  - `showStatusPanel` — when `false`, the Status panel is removed from the side
    panel layout.
  - `showCommitsPanel` — when `false`, the Commits panel (and its Reflog tab) is
    removed from the side panel layout.
  - `showStashPanel` — when `false`, the Stash panel is removed from the side
    panel layout.
- A hidden panel is excluded from the side-panel window list entirely; its
  vertical space is redistributed to the remaining visible panels by the
  existing weighting logic (no empty gap is left behind).
- Side-panel number-jump keys (`1`–`5`) renumber to the *visible* panels in
  top-to-bottom order. With `showStatusPanel: false`, `1` jumps to Files. The
  `[n]` badge rendered in each panel title reflects the same renumbering.
- Focus and the default/fallback context never land on a hidden panel.
- Config reload is honored: toggling any of these in the config file while
  lazygit is running takes effect on the next layout (or, if that is not
  feasible, the user is prompted to restart — to be settled in design).

The Files panel is intentionally **not** configurable: it is the anchor of the
side panel and at least one weighted panel must always remain.

## Capabilities

### New Capabilities
- `side-panel-visibility`: User-configurable show/hide of the Status, Commits,
  and Stash side panels, including the resulting layout redistribution, number-key
  renumbering, title-badge renumbering, and focus-safety guarantees.

### Modified Capabilities
<!-- No existing specs in openspec/specs/; nothing to modify. -->

## Impact

- **Config**: new fields + defaults in `pkg/config/user_config.go`; regenerated
  JSON schema (`make generate`) and config docs.
- **Layout**: `pkg/gui/controllers/helpers/window_helper.go` (`SideWindows()`),
  `pkg/gui/controllers/helpers/window_arrangement_helper.go` (`sidePanelChildren`).
- **Navigation**: `pkg/gui/controllers/jump_to_side_window_controller.go`
  (the binding-count equality check must become a zip over visible windows),
  and the title-prefix mapping in `pkg/gui/views.go`.
- **Focus**: initial/fallback context selection must skip hidden windows.
- **Tests**: `window_arrangement_helper_test.go` cases for each panel hidden and
  combinations; an integration test covering hidden-panel navigation/renumbering.
- No external API or dependency changes. Non-breaking: defaults preserve current
  behavior exactly.
