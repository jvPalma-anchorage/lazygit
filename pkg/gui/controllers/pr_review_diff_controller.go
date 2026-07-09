package controllers

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
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
			Handler:         self.addCommentOrReply,
			Description:     self.c.Tr.PrReviewAddCommentDescription,
			DisplayOnScreen: true,
		},
		{
			Keys:            opts.GetKeys(config.Keybinding{"t"}),
			Handler:         self.toggleThreadResolved,
			Description:     self.c.Tr.PrReviewToggleResolveDescription,
			DisplayOnScreen: true,
		},
		{
			Keys:            opts.GetKeys(config.Keybinding{"C"}),
			Handler:         self.queueComment,
			Description:     self.c.Tr.PrReviewQueueCommentDescription,
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

// addCommentOrReply is context-sensitive on the cursor row: on a review-thread block
// it replies to that thread; on a diff row it posts a new comment over the selected
// line range (the original Phase 3 flow).
func (self *PrReviewDiffController) addCommentOrReply() error {
	if thread := self.context().SelectedThread(); thread != nil {
		return self.replyToThread(thread)
	}

	path, startLine, line, errMsg := self.context().SelectedCommentTarget()
	if errMsg != "" {
		self.c.ErrorToast(errMsg)
		return nil
	}

	var openPrompt func(initial string)
	openPrompt = func(initial string) {
		self.c.Prompt(types.PromptOpts{
			Title:          self.c.Tr.PrReviewAddCommentTitle,
			InitialContent: initial,
			HandleConfirm: func(body string) error {
				if strings.TrimSpace(body) == "" {
					return nil
				}
				self.submitComment(path, startLine, line, body, func() { openPrompt(body) })
				return nil
			},
		})
	}
	openPrompt("")

	return nil
}

// queueComment accumulates a comment on the selected line range into the session's
// pending review (Phase 7) — no network; the batch is submitted as ONE review from
// the file tree (`S`).
func (self *PrReviewDiffController) queueComment() error {
	path, startLine, line, errMsg := self.context().SelectedCommentTarget()
	if errMsg != "" {
		self.c.ErrorToast(errMsg)
		return nil
	}

	self.c.Prompt(types.PromptOpts{
		Title: self.c.Tr.PrReviewQueueCommentTitle,
		HandleConfirm: func(body string) error {
			if strings.TrimSpace(body) == "" {
				return nil
			}
			count := self.c.Contexts().PrReview.AddPendingComment(git_commands.PendingReviewComment{
				Path: path, StartLine: startLine, Line: line, Body: body,
			})
			self.c.Toast(fmt.Sprintf(self.c.Tr.PrReviewCommentQueued, count))
			return nil
		},
	})
	return nil
}

// replyToThread prompts for a reply body and posts it to the thread via the REST
// replies endpoint (keyed by the thread's root comment).
func (self *PrReviewDiffController) replyToThread(thread *models.ReviewThread) error {
	if len(thread.Comments) == 0 || thread.Comments[0].DatabaseID == 0 {
		self.c.ErrorToast(self.c.Tr.PrReviewThreadNotReplyable)
		return nil
	}
	rootCommentID := thread.Comments[0].DatabaseID

	var openPrompt func(initial string)
	openPrompt = func(initial string) {
		self.c.Prompt(types.PromptOpts{
			Title:          self.c.Tr.PrReviewReplyTitle,
			InitialContent: initial,
			HandleConfirm: func(body string) error {
				if strings.TrimSpace(body) == "" {
					return nil
				}
				self.submitWrite(func() error {
					owner, repo, number := self.c.Contexts().PrReview.Target()
					return self.c.Git().GitHub.ReplyToReviewComment(owner, repo, number, rootCommentID, body)
				}, self.c.Tr.PrReviewReplyAdded, func() { openPrompt(body) })
				return nil
			},
		})
	}
	openPrompt("")

	return nil
}

// toggleThreadResolved flips the resolved state of the thread under the cursor via
// the GraphQL resolve/unresolve mutation.
func (self *PrReviewDiffController) toggleThreadResolved() error {
	thread := self.context().SelectedThread()
	if thread == nil {
		self.c.ErrorToast(self.c.Tr.PrReviewNoThreadSelected)
		return nil
	}

	resolved := !thread.IsResolved
	toast := self.c.Tr.PrReviewThreadResolved
	if !resolved {
		toast = self.c.Tr.PrReviewThreadUnresolved
	}
	threadID := thread.ID
	self.submitWrite(func() error {
		return self.c.Git().GitHub.SetReviewThreadResolved(threadID, resolved)
	}, toast, nil)
	return nil
}

// submitWrite runs a review write (reply / resolve) off the UI thread, then toasts
// the outcome and reloads so the thread display reflects it. On failure the error is
// surfaced AND reopen (when non-nil) re-opens the originating prompt with the typed
// body intact — the spec's "input is not lost" clause.
func (self *PrReviewDiffController) submitWrite(write func() error, successToast string, reopen func()) {
	tree := self.c.Contexts().PrReview
	self.c.OnWorker(func(_ gocui.Task) error {
		err := write()

		self.c.OnUIThread(func() error {
			if err != nil {
				self.c.ErrorToast(fmt.Sprintf(self.c.Tr.PrReviewCommentFailed, err.Error()))
				if reopen != nil {
					reopen()
				}
				return nil
			}
			self.c.Toast(successToast)
			diffCtx := self.context()
			tree.ReloadThen(func() {
				diffCtx.Invalidate()
				self.c.PostRefreshUpdate(diffCtx)
			})
			return nil
		})
		return nil
	})
}

// submitComment POSTs the review comment off the UI thread (the gh call must not
// block the gocui main loop), then hops back to toast the result and, on success,
// re-fetch the review data so the new comment renders inline (task 5.5).
func (self *PrReviewDiffController) submitComment(path string, startLine int, line int, body string, reopen func()) {
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
				if reopen != nil {
					reopen()
				}
				return nil
			}
			self.c.Toast(self.c.Tr.PrReviewCommentAdded)
			// Reload the review data, then re-render the focused diff surface so the
			// just-posted comment appears inline (the data reload alone does not
			// repaint the active diff view, which caches its render).
			diffCtx := self.context()
			tree.ReloadThen(func() {
				diffCtx.Invalidate()
				self.c.PostRefreshUpdate(diffCtx)
			})
			return nil
		})
		return nil
	})
}
