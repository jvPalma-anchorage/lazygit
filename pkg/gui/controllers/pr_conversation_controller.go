package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// PrConversationController drives the PR review workspace's Conversation tab: it loads
// the reviewer list and renders the selected reviewer's detail (review summary, inline
// thread comments with per-thread state, and PR-body comments) into the main view.
// List navigation is attached generically (AllList).
type PrConversationController struct {
	baseController
	c *ControllerCommon
}

var _ types.IController = &PrConversationController{}

func NewPrConversationController(c *ControllerCommon) *PrConversationController {
	return &PrConversationController{baseController: baseController{}, c: c}
}

func (self *PrConversationController) Context() types.Context {
	return self.context()
}

func (self *PrConversationController) context() *context.PrConversationContext {
	return self.c.Contexts().PrConversation
}

// GetOnRenderToMain loads the reviewers (idempotent, fires reliably on focus) and
// renders the selected reviewer's detail through the markdown renderer into main.
func (self *PrConversationController) GetOnRenderToMain() func() {
	return func() {
		self.context().Load()

		reviewer := self.context().SelectedReviewer()
		review := self.c.Contexts().PrReview
		data := review.Data()

		var content string
		if reviewer == nil || data == nil {
			content = self.c.Tr.PrReviewNoReviewers
		} else {
			content = presentation.RenderReviewerDetail(
				self.c.Tr,
				*reviewer,
				data.Reviews,
				data.Threads,
				data.IssueComments,
				self.c.Views().Main.InnerWidth(),
				review.MarkdownRenderer(),
			)
		}

		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: self.c.Tr.PrReviewConversationTitle,
				Task:  types.NewRenderStringWithoutScrollTask(content),
			},
		})
	}
}
