package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// PrOverviewController drives the PR review workspace's Overview tab: when focused, it
// renders the pull request's headline details (title, number, state, author, base/head
// branches) and its description body into the main view.
type PrOverviewController struct {
	baseController
	c *ControllerCommon
}

var _ types.IController = &PrOverviewController{}

func NewPrOverviewController(c *ControllerCommon) *PrOverviewController {
	return &PrOverviewController{baseController: baseController{}, c: c}
}

func (self *PrOverviewController) Context() types.Context {
	return self.c.Contexts().PrOverview
}

func (self *PrOverviewController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{}
}

// GetOnRenderToMain renders the PR overview into the main view whenever the Overview
// tab is focused. The data comes from the live PR-review context.
func (self *PrOverviewController) GetOnRenderToMain() func() {
	return func() {
		review := self.c.Contexts().PrReview
		review.EnsureLoaded()

		content := presentation.RenderPrOverview(
			self.c.Tr,
			review.Data(),
			self.c.Views().Main.InnerWidth(),
			review.MarkdownRenderer(),
		)

		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: self.c.Tr.PrOverviewTitle,
				Task:  types.NewRenderStringWithoutScrollTask(content),
			},
		})
	}
}
