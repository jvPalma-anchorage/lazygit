package context

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// PrConversationContext is the PR review workspace's Conversation tab (window
// prActivity). It lists the pull request's reviewers with a state icon; selecting one
// shows that reviewer's review summary, inline thread comments, and PR-body comments in
// the main view (rendered by PrConversationController). The reviewer data comes from
// the live PR-review (Files Changed) context, which owns the fetched review data.
type PrConversationContext struct {
	*ListViewModel[*models.Reviewer]
	*ListContextTrait

	c *ContextCommon

	reviewers   []*models.Reviewer
	loadStarted bool
}

var _ types.IListContext = (*PrConversationContext)(nil)

func NewPrConversationContext(c *ContextCommon) *PrConversationContext {
	self := &PrConversationContext{c: c}

	viewModel := NewListViewModel(func() []*models.Reviewer { return self.reviewers })
	self.ListViewModel = viewModel

	getDisplayStrings := func(_ int, _ int) [][]string {
		if len(self.reviewers) == 0 {
			return [][]string{{style.FgYellow.Sprint(c.Tr.PrReviewNoReviewers)}}
		}
		return lo.Map(self.reviewers, func(r *models.Reviewer, _ int) []string {
			return []string{reviewStateIcon(r.State), style.FgMagenta.Sprint("@" + r.Login), colorizeReviewerState(r.State)}
		})
	}

	baseContext := NewBaseContext(NewBaseContextOpts{
		Kind:       types.SIDE_CONTEXT,
		View:       c.Views().PrConversation,
		WindowName: "prActivity",
		Key:        PR_CONVERSATION_CONTEXT_KEY,
		Focusable:  true,
	})

	self.ListContextTrait = &ListContextTrait{
		Context: NewSimpleContext(baseContext),
		ListRenderer: ListRenderer{
			list:              viewModel,
			getDisplayStrings: getDisplayStrings,
		},
		c: c,
	}

	return self
}

func (self *PrConversationContext) reviewContext() *PrReviewContext {
	ctx, ok := self.c.ContextForKey(PR_REVIEW_CONTEXT_KEY).(*PrReviewContext)
	if !ok {
		return nil
	}
	return ctx
}

// Load populates the reviewer list from the PR-review data the first time it is
// available. Driven from the controller's render hook (which fires reliably on focus).
func (self *PrConversationContext) Load() {
	if self.loadStarted {
		return
	}
	review := self.reviewContext()
	if review == nil || review.Data() == nil {
		return
	}
	self.loadStarted = true

	data := review.Data()
	self.reviewers = lo.Map(data.Reviewers, func(r models.Reviewer, _ int) *models.Reviewer {
		reviewer := r
		return &reviewer
	})
	self.c.PostRefreshUpdate(self)
}

// Reload clears the load guard so the next render re-populates (e.g. after a refresh).
func (self *PrConversationContext) Reload() {
	self.loadStarted = false
	self.Load()
}

// SelectedReviewer returns the reviewer the cursor is on, or nil.
func (self *PrConversationContext) SelectedReviewer() *models.Reviewer {
	return self.GetSelected()
}

// reviewStateIcon returns a short glyph for a review state, so the reviewer list reads
// at a glance.
func reviewStateIcon(state string) string {
	switch state {
	case "APPROVED":
		return style.FgGreen.Sprint("✓")
	case "CHANGES_REQUESTED":
		return style.FgRed.Sprint("✗")
	case "COMMENTED":
		return style.FgCyan.Sprint("💬")
	case "PENDING":
		return style.FgYellow.Sprint("○")
	default:
		return " "
	}
}

// colorizeReviewerState mirrors the presenter's state colouring for the list column.
func colorizeReviewerState(state string) string {
	switch state {
	case "APPROVED":
		return style.FgGreen.Sprint(state)
	case "CHANGES_REQUESTED":
		return style.FgRed.Sprint(state)
	case "PENDING":
		return style.FgYellow.Sprint(state)
	default:
		return style.FgWhite.Sprint(state)
	}
}
