package controllers

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// PrListController drives window [1], the PR browser (Phase 13): [ / ] switch the
// section tabs, hovering a PR previews its overview in the main panel, enter focuses
// the main panel to scroll that overview, and SPACE starts the review of the selected
// PR (loading it into the content and activity windows).
type PrListController struct {
	baseController
	c *ControllerCommon
}

var _ types.IController = &PrListController{}

func NewPrListController(c *ControllerCommon) *PrListController {
	return &PrListController{baseController: baseController{}, c: c}
}

func (self *PrListController) Context() types.Context {
	return self.c.Contexts().PrList
}

func (self *PrListController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		// [ / ] cycle the section tabs (overriding the global tab handler, which
		// no-ops here since prList is not in viewTabMap).
		{
			Keys:    opts.GetKeys(opts.Config.Universal.NextTab),
			Handler: func() error { self.c.Contexts().PrList.NextSection(); return nil },
		},
		{
			Keys:    opts.GetKeys(opts.Config.Universal.PrevTab),
			Handler: func() error { self.c.Contexts().PrList.PrevSection(); return nil },
		},
		{
			Keys:            opts.GetKeys(opts.Config.Universal.Confirm),
			Handler:         self.focusOverview,
			Description:     self.c.Tr.PrListFocusOverviewDescription,
			DisplayOnScreen: true,
		},
		{
			Keys:            opts.GetKeys(opts.Config.Universal.Select),
			Handler:         self.startReview,
			Description:     self.c.Tr.PrListStartReviewDescription,
			DisplayOnScreen: true,
		},
	}
}

// GetOnRenderToMain loads the browser on first render and previews the selected PR's
// overview in the main panel. For the currently-loaded PR (and any PR whose data has
// been fetched) it shows the full overview; otherwise a lightweight header while the
// background fetch runs.
func (self *PrListController) GetOnRenderToMain() func() {
	return func() {
		ctx := self.c.Contexts().PrList
		ctx.Load()

		content := ""
		if entry := ctx.SelectedEntry(); entry != nil {
			if data := ctx.OverviewDataForSelected(); data != nil {
				content = presentation.RenderPrOverview(
					self.c.Tr,
					data,
					self.c.Views().Main.InnerWidth(),
					self.c.Contexts().PrReview.MarkdownRenderer(),
				)
			} else {
				content = self.lightHeader(entry)
			}
		}

		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: self.c.Tr.PrOverviewTitle,
				Task:  types.NewRenderStringWithoutScrollTask(content),
			},
		})
	}
}

// lightHeader is the placeholder overview shown from the list row's own fields while
// the full data is still being fetched.
func (self *PrListController) lightHeader(entry *git_commands.PrListEntry) string {
	return fmt.Sprintf("#%d %s\n%s/%s · %s\n\n%s",
		entry.Number, entry.Title, entry.Owner, entry.Repo, entry.State, self.c.Tr.PrReviewLoading)
}

// focusOverview (enter) moves focus into the main panel so the user can scroll the
// overview; Esc returns to the list.
func (self *PrListController) focusOverview() error {
	if self.c.Contexts().PrList.SelectedEntry() == nil {
		return nil
	}
	self.c.Context().Push(self.c.Contexts().Normal, types.OnFocusOpts{})
	return nil
}

// startReview (SPACE) loads the selected PR into the review workspace: retargeting the
// content/activity windows at it and jumping to the content window. Selecting the
// already-loaded PR just jumps.
func (self *PrListController) startReview() error {
	entry := self.c.Contexts().PrList.SelectedEntry()
	if entry == nil {
		return nil
	}

	review := self.c.Contexts().PrReview
	owner, repo, number := review.Target()
	if !(entry.Owner == owner && entry.Repo == repo && entry.Number == number) {
		review.Retarget(entry.Owner, entry.Repo, entry.Number)
	}
	self.c.Context().Push(review, types.OnFocusOpts{})
	return nil
}
