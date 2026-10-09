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

// isSideWindowVisible reports whether the window is part of the side panel
// layout. It reads the Gui-level isReviewMode flag (not gui.State) so it is
// correct even before the repo state exists.
func (gui *Gui) isSideWindowVisible(windowName string) bool {
	return lo.Contains(helpers.SideWindowNames(gui.c.UserConfig(), gui.isReviewMode), windowName)
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
