## 1. Config plumbing

- [x] 1.1 Add `ShowStatusPanel`, `ShowCommitsPanel`, `ShowStashPanel` bool fields to `GuiConfig` in `pkg/config/user_config.go` (with `yaml:"showStatusPanel"` etc. tags), placed near the other `show*` fields.
- [x] 1.2 Set all three defaults to `true` in the default-config constructor (alongside `ShowBottomLine`/`ShowFileTree`).
- [x] 1.3 Run `make generate` and confirm the generated config schema/docs pick up the three new options. (Note: generator targets `schema-master/config.json` and `docs-master/Config.md`, not the plain `schema/`+`docs/` copies.) Commit the regenerated files together with the config change.

## 2. Prep refactor — single source of truth (behavior-preserving)

- [x] 2.1 Refactor `Gui.configureViewProperties()` (`pkg/gui/views.go`) so the `[n]` title badges are assigned by iterating a `window → []*View` table in side-window order, instead of the hard-coded `jumpLabels[0..4]` assignments. With all panels visible this must produce byte-identical badges; verify no behavior change. (Implemented via a new instance-free `helpers.SideWindowNames()` free function — `configureViewProperties()` runs on the first config load *before* `resetHelpersAndControllers()`, so it cannot use the `WindowHelper` instance; `WindowHelper.SideWindows()` now delegates to that free function, keeping a single source of truth.)
- [x] 2.2 Refactor `JumpToSideWindowController.GetKeybindings()` (`pkg/gui/controllers/jump_to_side_window_controller.go`) to bind over the configured `JumpToBlock` keys by index, with the handler resolving the target via `SideWindows()[index]` at press time (out-of-range index = inert no-op). Keep the fatal check but only trigger it when `len(JumpToBlock) < len(SideWindows())`. With 5 panels + 5 keys, behavior is unchanged.
- [ ] 2.3 Commit tasks 2.1 and 2.2 as separate prep-refactor commits (each behavior-preserving, tree green).

## 3. Behavior — visibility filtering

- [x] 3.1 Make the side-window list config-aware in `pkg/gui/controllers/helpers/window_helper.go`: thread `UserConfig().Gui` into `SideWindowNames()` (the free function introduced in 2.1) so it omits `status`/`commits`/`stash` when their option is `false`; `files` and `branches` always included, order preserved. Update the two call sites to pass config: `WindowHelper.SideWindows()` (via `self.c.UserConfig()`) and `configureViewProperties()` in `views.go` (via `gui.c.UserConfig()`, available at startup).
- [x] 3.2 Update `WindowArrangementHelper.sidePanelChildren()` (`pkg/gui/controllers/helpers/window_arrangement_helper.go`) so every screen-mode branch builds its child list from the visible window set (drop any window not in `SideWindows()`), preserving the existing per-window size/weight rules and the accordion/`expandFocusedSidePanel` math over the visible windows only.
- [x] 3.3 Add the focus-safety guard (design D5): in `defaultSideContext()` / `initialContext()` (`pkg/gui/context_config.go`, `pkg/gui/gui.go`), fall back to the Files context when the otherwise-selected context's window is not in `SideWindows()` (covers Commits hidden during filtering or a commits start-arg). Implemented via `gui.isSideWindowVisible(windowName)`.

## 4. Sweep for hidden-assumptions

- [x] 4.1 Swept `pkg/gui` for literal window-name / 5-side-window assumptions (mouse routing, tab cycling, `SideWindows()` callers, presentation/cheatsheet) — no hard-coded lists or fixed-index assumptions remain; `side_window_controller`, `jump_to_side_window_controller`, and tab handling already use the config-aware list. The real exposure was focus paths that could land on a hidden side panel. Fixed centrally rather than per call site:
  - **Central focus guard** (`pkg/gui/context.go`): new `ContextMgr.focusableSideContext()` redirects a `SIDE_CONTEXT` whose window is hidden to the Files context, applied at the top of both `Push()` and `Replace()` before stack mutation. This single choke point covers every previously-found unconditional `LocalCommits` push (filtering, custom-patch ×2, merge/rebase, fixup), the status-controller click push, and the repo-restore path (whose context flows through `Push` at `gui.go:429`) — plus any future push. Safe because all side-context window names are canonical and files/branches are never hideable.
  - **Transient view gate** (`pkg/gui/layout.go`): a `SIDE_CONTEXT` transient view (e.g. `commitFiles` in the `commits` window) is forced `Visible=false` when its window is hidden, so it isn't re-shown after `setViewFromDimensions` hid it.
  - **Reload focus correction** (`pkg/gui/context_config.go` + reload block in `pkg/gui/gui.go`): `redirectFocusFromHiddenSideWindow()` pushes Files when `CurrentSide()`'s window was just hidden by a live config reload. Because a side-context push clears the stack, this also removes any stranded hidden context, making `Pop()` / `CurrentSide()` / `Current()` unable to resurrect it.

## 5. Tests

- [x] 5.1 Add `window_arrangement_helper_test.go` cases: each of Status/Commits/Stash hidden individually, all three hidden (only Files+Branches), and one case combined with `expandFocusedSidePanel`. Assert the ASCII layout shows the hidden panel gone and its space redistributed.
- [x] 5.2 Add a unit test asserting consistency for a hidden panel: the badge index, the jump-key index, and the layout order all agree (guards against the three-site drift called out in design Risks).
- [x] 5.3 Add a unit test for the jump controller: visible windows < jump keys does not fatal and surplus keys are inert; jump keys fewer than visible windows still fatals.
- [x] 5.4 Add an integration test (`pkg/integration/tests/`) that sets `showStatusPanel: false`, presses the first jump key, and asserts focus lands on Files (renumbering) and that Files' title badge is `[1]`. Run `make generate` to register it in `test_list.go`.

## 6. Docs & verification

- [x] 6.1 Confirmed the generated config schema/docs reflect all three options: `schema-master/config.json` (boolean, default true) and `docs-master/Config.md` (full descriptions). The plain `schema/config.json` + `docs/Config.md` do NOT contain them — expected, since the generator (`pkg/jsonschema/generate.go`, `generate_config_docs.go`) writes only to the `*-master` paths. No hand-edit needed.
- [x] 6.2 Green across the board: `make lint` → 0 issues; `make unit-test` → all packages ok; `go build ./...` ok; `gofumpt -l .` clean; new integration test passes headless. (Ran via a parallel verification workflow.)
- [ ] 6.3 Manual smoke check (`make run`) toggling each option + live config reload — **requires an interactive terminal; cannot be automated here.** Automation-covered portions: binary builds and launches (`--version` ok); the `hide_status_panel_renumbers_jump_keys` integration test exercises status-hidden badge `[1]`/`[2]` + jump-key renumbering; unit tests cover layout redistribution and `SideWindowNames` filtering. **Not yet covered by any automated test: the live-config-reload focus correction (`redirectFocusFromHiddenSideWindow`) and the half/full/squashed layout branches with hidden panels** — verify these by hand, or add tests (see follow-ups). Left unchecked for a human to perform.
