package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// PrCommitsController drives the PR review workspace's Commits tab: it renders the
// cursor commit (message + changed-file stat + diffs, through the user's pager) into
// the main view, and on Enter focuses the main view so the user can scroll that
// commit's changes. List navigation is attached generically (AllList).
type PrCommitsController struct {
	baseController
	c *ControllerCommon
}

var _ types.IController = &PrCommitsController{}

func NewPrCommitsController(c *ControllerCommon) *PrCommitsController {
	return &PrCommitsController{baseController: baseController{}, c: c}
}

func (self *PrCommitsController) Context() types.Context {
	return self.context()
}

func (self *PrCommitsController) context() *context.PrCommitsContext {
	return self.c.Contexts().PrCommits
}

func (self *PrCommitsController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Keys:            opts.GetKeys(opts.Config.Universal.Confirm),
			Handler:         self.enter,
			Description:     self.c.Tr.PrReviewViewCommitDescription,
			DisplayOnScreen: true,
		},
	}
}

// GetOnRenderToMain renders the selected PR commit (message + --stat file list +
// diffs) into the main view through the user's pager, exactly as the normal commits
// panel shows a commit. Fires on focus and on every cursor move.
func (self *PrCommitsController) GetOnRenderToMain() func() {
	return func() {
		// Load the base..head commits here: this hook fires reliably on focus,
		// unlike the context's on-focus callback. Load is idempotent.
		self.context().Load()

		commit := self.context().SelectedCommit()
		var task types.UpdateTask
		if commit == nil {
			task = types.NewRenderStringTask(self.c.Tr.PrReviewNoCommits)
		} else {
			cmdObj := self.c.Git().Commit.ShowCmdObj(commit.Hash(), nil)
			task = types.NewRunPtyTask(cmdObj.GetCmd())
		}

		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: self.c.Tr.PrCommitsTitle,
				Task:  task,
			},
		})
	}
}

// enter focuses the main view so the user can scroll the selected commit's changes
// (its --stat file list and per-file diffs). Escape returns to the commit list.
func (self *PrCommitsController) enter() error {
	if self.context().SelectedCommit() == nil {
		return nil
	}
	self.c.Context().Push(self.c.Contexts().Normal, types.OnFocusOpts{})
	return nil
}
