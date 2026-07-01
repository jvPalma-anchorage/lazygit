package gui

import (
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

func (gui *Gui) contextTree() *context.ContextTree {
	contextCommon := &context.ContextCommon{
		IGuiCommon: gui.c.IGuiCommon,
		Common:     gui.c.Common,
	}
	return context.NewContextTree(contextCommon)
}

// sideWindowNames returns the visible side-panel windows for the current mode: the
// dedicated PR review windows when booted into review mode, otherwise the normal
// config-gated set. It reads the Gui-level isReviewMode flag (not gui.State) so it
// is correct even when reached on the first config load (configureViewProperties),
// which runs before the repo state exists.
func (gui *Gui) sideWindowNames() []string {
	if gui.isReviewMode {
		return helpers.ReviewSideWindowNames()
	}
	return helpers.SideWindowNames(gui.c.UserConfig())
}

func (gui *Gui) isSideWindowVisible(windowName string) bool {
	return lo.Contains(gui.sideWindowNames(), windowName)
}

// redirectFocusFromHiddenSideWindow moves focus to the Files panel when the side
// window we are currently focused on (or whose main view we're in) has just been
// hidden by a config reload. Pushing the Files side context also clears the
// hidden context off the stack, so later Escape/Pop and CurrentSide() calls
// can't resurrect it. It's a no-op when the current side window is still visible.
func (gui *Gui) redirectFocusFromHiddenSideWindow() {
	currentSideContext := gui.c.Context().CurrentSide()
	if !gui.isSideWindowVisible(currentSideContext.GetWindowName()) {
		gui.c.Context().Push(gui.defaultSideContext(), types.OnFocusOpts{})
	}
}

func (gui *Gui) defaultSideContext() types.Context {
	// In PR review mode the normal side windows are suppressed, so the default
	// (and Escape/pop fallback) side context is the PR list, never Files.
	if gui.isReviewMode {
		return gui.State.Contexts.PrList
	}

	if gui.State.Modes.Filtering.Active() {
		commitsContext := gui.State.Contexts.LocalCommits
		if gui.isSideWindowVisible(commitsContext.GetWindowName()) {
			return commitsContext
		}
	}

	return gui.State.Contexts.Files
}
