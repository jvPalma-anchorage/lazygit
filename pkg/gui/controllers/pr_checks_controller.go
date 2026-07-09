package controllers

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// PrChecksController drives the Checks tab (task 8.2): loading the checks tree on
// focus, collapsing/expanding workflow parents, and rendering a check's logs (with
// color forced on) into the main view on Enter.
type PrChecksController struct {
	baseController
	c *ControllerCommon
}

var _ types.IController = &PrChecksController{}

func NewPrChecksController(c *ControllerCommon) *PrChecksController {
	return &PrChecksController{baseController: baseController{}, c: c}
}

func (self *PrChecksController) Context() types.Context {
	return self.c.Contexts().PrChecks
}

func (self *PrChecksController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Keys:            opts.GetKeys(opts.Config.Universal.Confirm),
			Handler:         self.enter,
			Description:     self.c.Tr.PrChecksEnterDescription,
			DisplayOnScreen: true,
		},
	}
}

// GetOnRenderToMain loads the checks on first focus (the reliable hook) and shows
// the selected check's summary in the main view.
func (self *PrChecksController) GetOnRenderToMain() func() {
	return func() {
		ctx := self.c.Contexts().PrChecks
		ctx.Load()

		content := ""
		if row := ctx.SelectedRow(); row != nil && row.Check != nil {
			content = fmt.Sprintf("%s / %s\n%s · %s\n%s",
				row.Workflow, row.Check.Name, row.Check.State, row.Check.Bucket, row.Check.Link)
		}
		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: self.c.Tr.PrChecksTitle,
				Task:  types.NewRenderStringWithoutScrollTask(content),
			},
		})
	}
}

// enter toggles a workflow parent, or renders the selected check's logs through a
// pty (color forced on) when the check links to a GitHub Actions run.
func (self *PrChecksController) enter() error {
	ctx := self.c.Contexts().PrChecks
	row := ctx.SelectedRow()
	if row == nil {
		return nil
	}
	if row.Check == nil {
		ctx.ToggleCollapsed(row.Workflow)
		return nil
	}

	runID, jobID, ok := git_commands.CheckRunJobIDs(row.Check.Link)
	if !ok {
		// External CI (a details URL outside GitHub Actions) has no fetchable logs.
		self.c.ErrorToast(self.c.Tr.PrChecksNoLogs)
		return nil
	}

	owner, repo, _ := self.c.Contexts().PrReview.Target()
	cmdObj := self.c.Git().GitHub.ChecksLogCmdObj(owner, repo, runID, jobID)
	self.c.RenderToMainViews(types.RefreshMainOpts{
		Pair: self.c.MainViewPairs().Normal,
		Main: &types.ViewUpdateOpts{
			Title: self.c.Tr.PrChecksTitle,
			Task:  types.NewRunPtyTask(cmdObj.GetCmd()),
		},
	})
	return nil
}
