package controllers

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// PullRequestsController drives the Pull Requests tab of the branches window:
// it previews the selected PR's overview in the main view, and on Enter opens
// the PR in review mode.
type PullRequestsController struct {
	baseController
	*ListControllerTrait[*models.GithubPullRequest]
	c *ControllerCommon
}

var _ types.IController = &PullRequestsController{}

func NewPullRequestsController(c *ControllerCommon) *PullRequestsController {
	return &PullRequestsController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().PullRequests,
			c.Contexts().PullRequests.GetSelected,
			c.Contexts().PullRequests.GetSelectedItems,
		),
		c: c,
	}
}

func (self *PullRequestsController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.Universal.GoInto),
			Handler:           self.withItem(self.review),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.ReviewPullRequest,
			DisplayOnScreen:   true,
		},
	}
}

func (self *PullRequestsController) GetOnRenderToMain() func() {
	return func() {
		content := self.c.Tr.NoPullRequests
		if pr := self.context().GetSelected(); pr != nil {
			if data := self.context().OverviewDataForSelected(); data != nil {
				content = presentation.RenderPrOverview(
					self.c.Tr,
					data,
					self.c.Views().Main.InnerWidth(),
					self.c.Contexts().PrReview.MarkdownRenderer(),
				)
			} else {
				// A placeholder from the list row's own fields while the full
				// overview is being fetched.
				content = fmt.Sprintf("%s\n%s:%s\n\n%s",
					presentation.FormatPullRequestHeader(pr, self.c.Tr), pr.UserName(), pr.BranchName(), self.c.Tr.PrReviewLoading)
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

// review opens the pull request in review mode by running lazygit on it as a
// subprocess, exactly as `lazygit -p <repo> <owner>/<repo> <number>` would.
// Quitting review mode returns here.
func (self *PullRequestsController) review(pr *models.GithubPullRequest) error {
	lazygitPath, err := os.Executable()
	if err != nil {
		return err
	}

	args, ok := reviewCommandArgs(lazygitPath, self.c.Git().RepoPaths.WorktreePath(), pr)
	if !ok {
		return errors.New(self.c.Tr.PullRequestRepoUnknown)
	}

	return self.c.RunSubprocessAndRefresh(self.c.OS().Cmd.New(args))
}

func reviewCommandArgs(lazygitPath string, repoPath string, pr *models.GithubPullRequest) ([]string, bool) {
	owner, repo, ok := pr.BaseRepo()
	if !ok {
		return nil, false
	}

	return []string{lazygitPath, "-p", repoPath, owner + "/" + repo, strconv.Itoa(pr.Number)}, true
}

func (self *PullRequestsController) context() *context.PullRequestsContext {
	return self.c.Contexts().PullRequests
}
