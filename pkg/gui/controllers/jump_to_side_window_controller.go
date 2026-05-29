package controllers

import (
	"log"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

type JumpToSideWindowController struct {
	baseController
	c           *ControllerCommon
	nextTabFunc func() error
}

func NewJumpToSideWindowController(
	c *ControllerCommon,
	nextTabFunc func() error,
) *JumpToSideWindowController {
	return &JumpToSideWindowController{
		baseController: baseController{},
		c:              c,
		nextTabFunc:    nextTabFunc,
	}
}

func (self *JumpToSideWindowController) Context() types.Context {
	return nil
}

// enoughJumpKeys reports whether there are at least as many jump-to-block keys
// as visible side windows. Each visible window needs a key to be reachable;
// surplus keys are harmless (they resolve to an out-of-range index and are
// inert), but a deficit would leave a window unreachable, which is a misconfig.
func enoughJumpKeys(jumpKeyCount int, sideWindowCount int) bool {
	return jumpKeyCount >= sideWindowCount
}

func (self *JumpToSideWindowController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	windows := self.c.Helpers().Window.SideWindows()

	if !enoughJumpKeys(len(opts.Config.Universal.JumpToBlock), len(windows)) {
		log.Fatal("Jump to block keybindings cannot be set. At least as many keybindings as side windows must be supplied.")
	}

	return lo.Map(opts.Config.Universal.JumpToBlock, func(jumpKey config.Keybinding, index int) *types.Binding {
		return &types.Binding{
			ViewName: "",
			// by default the keys are 1, 2, 3, etc
			Keys:    opts.GetKeys(jumpKey),
			Handler: opts.Guards.NoPopupPanel(self.goToSideWindowByIndex(index)),
		}
	})
}

func (self *JumpToSideWindowController) goToSideWindowByIndex(index int) func() error {
	return func() error {
		windows := self.c.Helpers().Window.SideWindows()
		if index >= len(windows) {
			return nil
		}
		window := windows[index]

		sideWindowAlreadyActive := self.c.Helpers().Window.CurrentWindow() == window
		if sideWindowAlreadyActive && self.c.UserConfig().Gui.SwitchTabsWithPanelJumpKeys {
			return self.nextTabFunc()
		}

		context := self.c.Helpers().Window.GetContextForWindow(window)

		self.c.Context().Push(context, types.OnFocusOpts{})
		return nil
	}
}
