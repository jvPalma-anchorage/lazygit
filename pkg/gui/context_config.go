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

func (gui *Gui) isSideWindowVisible(windowName string) bool {
	return lo.Contains(helpers.SideWindowNames(gui.c.UserConfig()), windowName)
}

// redirectFocusFromHiddenSideWindow moves focus to the Files panel when the side
// window we are currently focused on (or whose main view we're in) has just been
// hidden by a config reload. Pushing the Files side context also clears the
// hidden context off the stack, so later Escape/Pop and CurrentSide() calls
// can't resurrect it. It's a no-op when the current side window is still visible.
func (gui *Gui) redirectFocusFromHiddenSideWindow() {
	currentSideContext := gui.c.Context().CurrentSide()
	if !gui.isSideWindowVisible(currentSideContext.GetWindowName()) {
		gui.c.Context().Push(gui.State.Contexts.Files, types.OnFocusOpts{})
	}
}

func (gui *Gui) defaultSideContext() types.Context {
	if gui.State.Modes.Filtering.Active() {
		commitsContext := gui.State.Contexts.LocalCommits
		if gui.isSideWindowVisible(commitsContext.GetWindowName()) {
			return commitsContext
		}
	}

	return gui.State.Contexts.Files
}
