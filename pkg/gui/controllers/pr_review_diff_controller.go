package controllers

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// PrReviewDiffController drives the focusable diff view the user enters from the
// file tree: moving a cursor over selectable diff rows (skipping comment rows),
// extending a line range with space / shift-arrows, and posting a review comment
// over that range via gh (design D5). Escape returns to the file tree.
type PrReviewDiffController struct {
	baseController
	c *ControllerCommon
}

var _ types.IController = &PrReviewDiffController{}

func NewPrReviewDiffController(c *ControllerCommon) *PrReviewDiffController {
	return &PrReviewDiffController{baseController: baseController{}, c: c}
}

func (self *PrReviewDiffController) Context() types.Context {
	return self.context()
}

func (self *PrReviewDiffController) context() *context.PrReviewDiffContext {
	return self.c.Contexts().PrReviewDiff
}

func (self *PrReviewDiffController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Tag:     "navigation",
			Keys:    opts.GetKeys(opts.Config.Universal.PrevItem),
			Handler: func() error { self.context().MoveSelection(-1, false); return nil },
		},
		{
			Tag:     "navigation",
			Keys:    opts.GetKeys(opts.Config.Universal.NextItem),
			Handler: func() error { self.context().MoveSelection(1, false); return nil },
		},
		{
			Tag:         "navigation",
			Keys:        opts.GetKeys(opts.Config.Universal.RangeSelectUp),
			Handler:     func() error { self.context().MoveSelection(-1, true); return nil },
			Description: self.c.Tr.RangeSelectUp,
		},
		{
			Tag:         "navigation",
			Keys:        opts.GetKeys(opts.Config.Universal.RangeSelectDown),
			Handler:     func() error { self.context().MoveSelection(1, true); return nil },
			Description: self.c.Tr.RangeSelectDown,
		},
		{
			// space adds the next line to the selection range (select lines to comment on).
			Keys:            opts.GetKeys(opts.Config.Universal.Select),
			Handler:         func() error { self.context().MoveSelection(1, true); return nil },
			Description:     self.c.Tr.RangeSelectDown,
			DisplayOnScreen: true,
		},
		{
			Keys:            opts.GetKeys(config.Keybinding{"c"}),
			Handler:         self.addComment,
			Description:     self.c.Tr.PrReviewAddCommentDescription,
			DisplayOnScreen: true,
		},
		{
			Keys:            opts.GetKeys(opts.Config.Universal.Return),
			Handler:         func() error { self.c.Context().Pop(); return nil },
			Description:     self.c.Tr.ReturnToFilesPanel,
			DisplayOnScreen: true,
		},
	}
}

func (self *PrReviewDiffController) addComment() error {
	path, startLine, line, errMsg := self.context().SelectedCommentTarget()
	if errMsg != "" {
		self.c.ErrorToast(errMsg)
		return nil
	}

	self.c.Prompt(types.PromptOpts{
		Title: self.c.Tr.PrReviewAddCommentTitle,
		HandleConfirm: func(body string) error {
			if strings.TrimSpace(body) == "" {
				return nil
			}
			self.submitComment(path, startLine, line, body)
			return nil
		},
	})

	return nil
}

// submitComment POSTs the review comment off the UI thread (the gh call must not
// block the gocui main loop), then hops back to toast the result and, on success,
// re-fetch the review data so the new comment renders inline (task 5.5).
func (self *PrReviewDiffController) submitComment(path string, startLine int, line int, body string) {
	tree := self.c.Contexts().PrReview
	owner, repo, number := tree.Target()
	data := tree.Data()
	if data == nil {
		return
	}
	headOid := data.HeadRefOid

	self.c.OnWorker(func(_ gocui.Task) error {
		token := self.c.Git().GitHub.GetAuthToken("github.com")
		err := self.c.Git().GitHub.AddReviewComment(owner, repo, number, headOid, path, body, startLine, line, token)

		self.c.OnUIThread(func() error {
			if err != nil {
				self.c.ErrorToast(fmt.Sprintf(self.c.Tr.PrReviewCommentFailed, err.Error()))
				return nil
			}
			self.c.Toast(self.c.Tr.PrReviewCommentAdded)
			tree.Reload()
			return nil
		})
		return nil
	})
}
