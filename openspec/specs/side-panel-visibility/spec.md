# side-panel-visibility Specification

## Purpose
TBD - created by archiving change configurable-side-panel-visibility. Update Purpose after archive.
## Requirements
### Requirement: Configurable visibility of Status, Commits, and Stash panels

Lazygit SHALL provide three boolean user-config options under `gui` —
`showStatusPanel`, `showCommitsPanel`, and `showStashPanel` — each defaulting to
`true`. When an option is `true`, its panel appears in the side-panel layout as
it does today. When an option is `false`, its panel SHALL be excluded from the
side-panel window list and SHALL NOT occupy any vertical space in the layout.

The Files and Branches panels SHALL NOT be configurable and SHALL always be
present.

#### Scenario: All options default to true (current behavior preserved)
- **WHEN** a user runs lazygit with no configuration for these options
- **THEN** all five side panels (Status, Files, Branches, Commits, Stash) are
  shown exactly as before this change

#### Scenario: Hiding the Status panel
- **WHEN** `gui.showStatusPanel` is `false`
- **THEN** the Status panel is not rendered in the side-panel column
- **AND** no blank space is left where the Status panel would have been

#### Scenario: Hiding the Commits panel also hides its Reflog tab
- **WHEN** `gui.showCommitsPanel` is `false`
- **THEN** the Commits panel and the Reflog tab that shares its window are not
  rendered in the side-panel column

#### Scenario: Hiding the Stash panel
- **WHEN** `gui.showStashPanel` is `false`
- **THEN** the Stash panel is not rendered in the side-panel column

#### Scenario: Hiding multiple panels at once
- **WHEN** `gui.showStatusPanel`, `gui.showCommitsPanel`, and
  `gui.showStashPanel` are all `false`
- **THEN** only the Files and Branches panels are rendered in the side-panel
  column, sharing the full side-panel height

### Requirement: Hidden panel space is redistributed to visible panels

When a panel is hidden, the vertical space it would have occupied SHALL be
redistributed to the remaining visible side panels using the existing panel
weighting/sizing rules, so the side-panel column always fills its full height.

#### Scenario: Status space goes to the weighted panels
- **WHEN** `gui.showStatusPanel` is `false`
- **THEN** the height previously reserved for the fixed-size Status panel is
  given to the remaining visible panels
- **AND** the relative sizing of the remaining panels follows the same rules
  that apply when Status is shown

#### Scenario: Redistribution still respects accordion / focused-panel expansion
- **WHEN** a panel is hidden **AND** `gui.expandFocusedSidePanel` is enabled
- **THEN** the focused visible panel still expands according to
  `gui.expandedSidePanelWeight`, computed over the visible panels only

### Requirement: Side-panel number-jump keys renumber to visible panels

The side-panel jump keys (default `1`–`5`) SHALL map, in order, to the side
panels that are currently visible, top to bottom. Hidden panels SHALL NOT
consume a jump number. Pressing a jump key SHALL never attempt to focus a hidden
panel.

#### Scenario: Renumbering when Status is hidden
- **WHEN** `gui.showStatusPanel` is `false`
- **AND** the user presses the first side-panel jump key (`1` by default)
- **THEN** focus moves to the Files panel (the first visible panel)

#### Scenario: Surplus jump keys are inert
- **WHEN** fewer side panels are visible than there are configured jump keys
- **THEN** the extra jump keys are bound to no panel and pressing one does
  nothing
- **AND** lazygit does not error or crash on startup due to the count mismatch

### Requirement: Panel title badges reflect renumbering

The bracketed jump-number badge shown in each side-panel title SHALL display the
panel's position among the visible panels, matching the key that jumps to it.

#### Scenario: Badge matches the active jump key
- **WHEN** `gui.showStatusPanel` is `false`
- **THEN** the Files panel title shows the badge `[1]`
- **AND** the Branches panel title shows the badge `[2]`

### Requirement: Focus never lands on a hidden panel

Lazygit SHALL NOT place focus on, or use as the default/fallback context, any
panel whose visibility option is `false` — including at startup and when focus
falls back after closing a view.

#### Scenario: Startup focus skips a hidden default panel
- **WHEN** the panel that would normally receive focus at startup is hidden by
  its visibility option
- **THEN** focus is placed on a visible panel instead

### Requirement: Visibility options honor config reload

Lazygit SHALL apply a change to any of the three visibility options when the
config is reloaded while running, without requiring the user to manually
re-trigger the layout. If applying the change live is not feasible, lazygit SHALL inform the
user that a restart is required, consistent with how other non-live-reloadable
config changes are handled.

#### Scenario: Toggling a panel on via config reload
- **WHEN** lazygit is running with `gui.showStashPanel: false`
- **AND** the user edits the config to set `gui.showStashPanel: true` and that
  config is reloaded
- **THEN** the Stash panel becomes visible without the user restarting lazygit,
  OR lazygit prompts the user that a restart is required

