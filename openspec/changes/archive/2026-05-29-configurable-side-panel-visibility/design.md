## Context

Lazygit renders five side panels in a fixed vertical column: Status, Files,
Branches, Commits, Stash. The set is hard-coded in three independent places that
must stay in sync:

- `WindowHelper.SideWindows()` (`pkg/gui/controllers/helpers/window_helper.go:137`)
  returns the literal `[]string{"status","files","branches","commits","stash"}`.
  It drives both the jump-key bindings and the layout's child ordering.
- `WindowArrangementHelper.sidePanelChildren()`
  (`pkg/gui/controllers/helpers/window_arrangement_helper.go`) hard-codes the
  same window names (in several screen-mode branches) and assigns each a fixed
  size or weight (Status is fixed `Size: 3`; Files/Branches/Commits get
  `Weight: 1`; Stash is `Size: 3` unfocused / `Weight: 1` focused).
- `Gui.configureViewProperties()` (`pkg/gui/views.go:213-238`) hard-codes
  `jumpLabels[0..4]` onto specific views to render the `[n]` title badge.

The jump-key bindings live in
`JumpToSideWindowController.GetKeybindings()` (`jump_to_side_window_controller.go`),
which today `log.Fatal`s unless `len(JumpToBlock) == len(SideWindows())` (i.e.
exactly 5), then maps each window to the jump key at the same index.

Config reload path: `onUserConfigLoaded()` (`pkg/gui/gui.go:459`) runs on both
initial load and live reload, and calls `configureViewProperties()`. The window
arrangement is recomputed every render. **Keybindings are registered once at
startup and are not re-registered on reload.**

Default focus: `initialContext()` (`gui.go:692`) is `Files` by default
(`LocalCommits` or `Branches` for certain start args / filtering);
`defaultSideContext()` (`context_config.go:16`) is `Files`, or `LocalCommits`
when filtering is active. Status and Stash are never default focus targets.

## Goals / Non-Goals

**Goals:**
- Let users hide the Status, Commits, and Stash panels via persistent config,
  defaulting to today's behavior.
- Make a hidden panel consume zero layout space; redistribute via existing rules.
- Renumber jump keys and title badges to the visible panels, consistently.
- Guarantee focus never lands on a hidden panel.
- Make the visibility options reload-safe end to end (layout, badges, *and* the
  jump-key target mapping).

**Non-Goals:**
- Making the Files or Branches panel hideable (Files is the side-panel anchor;
  keeping Branches too guarantees ≥2 visible weighted panels and avoids
  degenerate single-panel layouts).
- A keybinding or menu to toggle visibility at runtime (config-only for now).
- Changing screen modes, accordion behavior, or the main/secondary views.
- Hiding individual tabs independently of their host window (hiding Commits also
  hides its Reflog tab; that is intended).

## Decisions

### D1: Single source of truth — make `SideWindows()` config-aware
`SideWindows()` becomes the one place that decides the visible set, filtering out
windows whose `gui.show*Panel` option is `false`. Every other site (layout,
badges, jump keys) derives its ordering/numbering from `SideWindows()` instead of
its own literal list. This collapses three hard-coded lists into one.

- *Alternative considered:* thread three booleans into each site independently.
  Rejected — triplicates the visibility logic and is exactly the drift the
  current code already risks.

The window-name → option mapping is fixed: `status`→`ShowStatusPanel`,
`commits`→`ShowCommitsPanel`, `stash`→`ShowStashPanel`; `files`/`branches`
unconditional.

### D2: `sidePanelChildren()` filters against the visible set
Each screen-mode branch in `sidePanelChildren()` is rewritten to build its child
list from `SideWindows()` (or to drop any window not in it) rather than naming
all five literally. The existing per-window sizing/weight rules are preserved for
the windows that remain; because Files/Branches/Commits are weighted, dropping
one automatically redistributes height to the others. Dropping Status (fixed
`Size: 3`) returns those rows to the weighted pool. Accordion / `expandFocused
SidePanel` math operates over the visible windows only.

### D3: Jump keys bind by index and resolve the target at press time
Replace the closure-over-window-name binding with a closure-over-index handler:
key at index `i` focuses `SideWindows()[i]` *evaluated when the key is pressed*.

- Registration maps over the configured `JumpToBlock` keys (not over windows), so
  all keys are registered once at startup and keep working after a live config
  reload that changes the visible set — no keybinding re-registration needed.
- The fatal check changes from `!= 5` to "fatal only if there are *fewer* jump
  keys than visible windows"; surplus keys resolve to out-of-range index and are
  inert no-ops.
- *Alternative considered:* keep mapping over windows and re-register keybindings
  on reload. Rejected — re-registering keybindings mid-session is not a supported
  path today and is far more invasive than resolving the index lazily.

### D4: Title badges derived from `SideWindows()` order
Rewrite `configureViewProperties()`'s badge block to iterate `SideWindows()` with
its index, mapping each window to its member views via a static
`window → []*View` table (`files`→Files/Worktrees/Submodules, etc.), and set
`TitlePrefix = jumpLabels[i]`. Views whose window is hidden get `""` (and are not
laid out anyway). Because this runs inside `configureViewProperties()`, badges
recompute on reload for free, staying consistent with D3.

### D5: Focus-safety guard for the Commits panel
Status/Stash need no guard (never default focus). The one hazard is `Commits`
being hidden while `defaultSideContext()`/`initialContext()` wants `LocalCommits`
(filtering mode, or a commits-oriented start arg). Decision: in those focus
selectors, if the chosen context's window is not in `SideWindows()`, fall back to
`Files`. Keep the guard centralized so future hideable panels inherit it.

- *Alternative considered:* force the Commits panel visible whenever filtering is
  active. Rejected as more surprising than falling back to Files; revisit only if
  users report wanting filtered commits while the panel is hidden.

### D6: Reload semantics — fully live
Layout (D2), badges (D4), and jump targets (D3) all recompute from current config
without restart, so these options live-reload cleanly and need **not** be added
to `checkForChangedConfigsThatDontAutoReload`. The spec's restart-prompt fallback
is therefore not exercised, but is left in the spec as the contractual floor.

## Risks / Trade-offs

- **Drift between the three derived sites** → Mitigation: D1 makes `SideWindows()`
  the sole authority; D2/D3/D4 consume it rather than re-listing windows. Add a
  unit test asserting badge index == jump-key index == layout order for a hidden
  panel.
- **`log.Fatal` regression for users with custom `JumpToBlock`** → A user who
  shortened `JumpToBlock` to <5 keys and hid no panels previously crashed; behavior
  is unchanged for them, but now hiding panels can make a short list legal.
  Mitigation: the new check only fatals when keys < visible windows; cover with a
  unit test.
- **Hiding Commits during filtering** (D5) → focus silently falls back to Files;
  could confuse a user who filtered by path. Mitigation: documented behavior;
  acceptable because hiding the commits panel while filtering commits is
  self-contradictory.
- **Mouse click / other code paths assuming 5 side windows** → Mitigation: audit
  references to the literal window names and `SideWindows()` length during
  implementation (tasks include a grep sweep).

## Migration Plan

Non-breaking: all three options default to `true`, reproducing current layout,
numbering, and focus exactly. No data or config migration. Rollback is reverting
the change; existing user configs without the keys are unaffected.

## Open Questions

- None blocking. Possible follow-up (out of scope): a runtime keybinding to
  toggle a panel, and/or making the badge index configurable per panel.
