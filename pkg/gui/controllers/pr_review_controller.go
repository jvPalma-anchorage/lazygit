package controllers

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// PrReviewController drives the PR-review file-tree side panel: it inherits list
// navigation from ListControllerTrait, renders the cursor file's inline diff (with
// review threads) into the main view via GetOnRenderToMain, and binds the
// per-file actions (toggle reviewed, toggle diff mode, view description).
type PrReviewController struct {
	baseController
	*ListControllerTrait[*filetree.CommitFileNode]
	c *ControllerCommon
}

var _ types.IController = &PrReviewController{}

func NewPrReviewController(c *ControllerCommon) *PrReviewController {
	return &PrReviewController{
		baseController: baseController{},
		c:              c,
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().PrReview,
			c.Contexts().PrReview.GetSelected,
			c.Contexts().PrReview.GetSelectedItems,
		),
	}
}

func (self *PrReviewController) context() *context.PrReviewContext {
	return self.c.Contexts().PrReview
}

func (self *PrReviewController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Keys:            opts.GetKeys(opts.Config.Universal.Confirm),
			Handler:         self.enterDiff,
			Description:     self.c.Tr.PrReviewEnterDiffDescription,
			DisplayOnScreen: true,
		},
		{
			Keys:            opts.GetKeys(opts.Config.Universal.Select),
			Handler:         func() error { self.context().ToggleReviewedForSelectedFile(); return nil },
			Description:     self.c.Tr.PrReviewToggleReviewedDescription,
			DisplayOnScreen: true,
		},
		{
			Keys:            opts.GetKeys(config.Keybinding{"t"}),
			Handler:         func() error { self.context().ToggleDiffMode(); return nil },
			Description:     self.c.Tr.PrReviewToggleDiffModeDescription,
			DisplayOnScreen: true,
		},
		{
			Keys:            opts.GetKeys(config.Keybinding{"d"}),
			Handler:         self.showDescription,
			Description:     self.c.Tr.PrReviewDescriptionDescription,
			DisplayOnScreen: true,
		},
	}
}

// GetOnRenderToMain renders the selected file's inline diff into the main view and
// the PR's global comments + reviewers into the secondary view. Fires on focus and
// on every cursor move within the tree (the list lifecycle calls this).
func (self *PrReviewController) GetOnRenderToMain() func() {
	return func() {
		ctx := self.context()
		// Kick off the one-time data fetch here: this hook reliably fires on boot
		// (the initial "Loading" render proves it), whereas the context's on-focus
		// callback does not for the initial boot context. EnsureLoaded is idempotent.
		ctx.EnsureLoaded()
		mainContent := ctx.RenderSelectedFileDiff(self.c.Views().Main.InnerWidth())
		secondaryContent := ctx.RenderConversation(self.c.Views().Secondary.InnerWidth())

		// Only split the panel for the conversation when there is one; otherwise the
		// diff gets the full main view.
		var secondary *types.ViewUpdateOpts
		if strings.TrimSpace(secondaryContent) != "" {
			secondary = &types.ViewUpdateOpts{
				Title: self.c.Tr.PrReviewConversationTitle,
				Task:  types.NewRenderStringWithoutScrollTask(secondaryContent),
			}
		}

		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: self.c.Tr.PrReviewTitle,
				Task:  types.NewRenderStringWithoutScrollTask(mainContent),
			},
			Secondary: secondary,
		})
	}
}

// enterDiff opens the selected file in the focusable diff context, where the user
// can move a cursor over diff lines, select a range, and post a review comment. The
// diff context renders via this closure (at its own view width) so it needs no
// direct dependency on the file-tree context.
func (self *PrReviewController) enterDiff() error {
	tree := self.context()

	// On a directory node, Enter collapses/expands it (native filetree behaviour)
	// rather than opening a diff.
	if node := tree.GetSelected(); node != nil && !node.IsFile() {
		tree.ToggleCollapsed(node.GetInternalPath())
		self.c.PostRefreshUpdate(tree)
		return nil
	}

	if tree.SelectedFile() == nil {
		return nil
	}

	diffCtx := self.c.Contexts().PrReviewDiff
	self.c.Context().Push(diffCtx, types.OnFocusOpts{})
	diffCtx.PrepareForEntry()
	return nil
}

// showDescription opens the PR title + body (rendered through glow when present)
// in a dismissible alert panel (design D10).
func (self *PrReviewController) showDescription() error {
	ctx := self.context()
	if ctx.Data() == nil {
		return nil
	}

	title, body := ctx.RenderedDescription(ctx.GetView().InnerWidth())
	if strings.TrimSpace(body) == "" {
		body = self.c.Tr.PrReviewNoDescription
	}
	if strings.TrimSpace(title) == "" {
		title = self.c.Tr.PrReviewTitle
	}
	self.c.Alert(title, body)
	return nil
}
