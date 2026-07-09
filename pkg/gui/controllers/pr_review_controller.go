package controllers

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
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
		{
			Keys:            opts.GetKeys(config.Keybinding{"G"}),
			Handler:         self.toggleGenerated,
			Description:     self.c.Tr.PrReviewToggleGeneratedDescription,
			DisplayOnScreen: true,
		},
		{
			// Re-fetch the PR (refreshing review data and re-applying persisted
			// "viewed" marks against current blob OIDs, so a file changed by a new
			// commit drops its mark).
			// Re-fetch the PR; the Commits tab re-loads automatically once the new refs
			// are applied (see refreshScopedCommits), so there is no stale-endpoint race.
			Keys:            opts.GetKeys(opts.Config.Universal.Refresh),
			Handler:         func() error { self.context().Reload(); return nil },
			Description:     self.c.Tr.PrReviewRefreshDescription,
			DisplayOnScreen: true,
		},
		{
			Keys:            opts.GetKeys(config.Keybinding{"S"}),
			Handler:         self.submitReview,
			Description:     self.c.Tr.PrReviewSubmitDescription,
			DisplayOnScreen: true,
		},
	}
}

// submitReview submits the session's pending comments as ONE pull-request review
// (Phase 7): pick the event (comment / approve / request changes), enter an optional
// summary body, POST, then clear the queue and reload so the new threads render.
func (self *PrReviewController) submitReview() error {
	ctx := self.context()
	pending := ctx.PendingComments()

	menuTitle := fmt.Sprintf(self.c.Tr.PrReviewSubmitMenuTitle, len(pending))
	events := []struct {
		label string
		event string
	}{
		{self.c.Tr.PrReviewSubmitEventComment, "COMMENT"},
		{self.c.Tr.PrReviewSubmitEventApprove, "APPROVE"},
		{self.c.Tr.PrReviewSubmitEventRequestChanges, "REQUEST_CHANGES"},
	}

	items := make([]*types.MenuItem, 0, len(events))
	for _, e := range events {
		event := e.event
		items = append(items, &types.MenuItem{
			Label: e.label,
			OnPress: func() error {
				self.c.Prompt(types.PromptOpts{
					Title: self.c.Tr.PrReviewSubmitBodyTitle,
					HandleConfirm: func(body string) error {
						self.doSubmitReview(event, body)
						return nil
					},
				})
				return nil
			},
		})
	}

	return self.c.Menu(types.CreateMenuOptions{Title: menuTitle, Items: items})
}

func (self *PrReviewController) doSubmitReview(event string, body string) {
	ctx := self.context()
	owner, repo, number := ctx.Target()
	data := ctx.Data()
	if data == nil {
		return
	}
	headOid := data.HeadRefOid
	pending := ctx.PendingComments()

	self.c.OnWorker(func(_ gocui.Task) error {
		err := self.c.Git().GitHub.SubmitReview(owner, repo, number, headOid, event, body, pending)

		self.c.OnUIThread(func() error {
			if err != nil {
				self.c.ErrorToast(fmt.Sprintf(self.c.Tr.PrReviewCommentFailed, err.Error()))
				return nil
			}
			ctx.ClearPendingComments()
			self.c.Toast(self.c.Tr.PrReviewSubmitted)
			ctx.Reload()
			return nil
		})
		return nil
	})
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

		// Local-ref read model (DW2 / Phase 2): render the selected file's (or
		// folder's) diff through the user's pager (delta) via a pty task, exactly as
		// the commit-files panel does, instead of the bespoke inline presenter.
		if ctx.LocalRefsActive() {
			self.renderLocalRefDiff(ctx)
			return
		}

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

// renderLocalRefDiff renders the cursor node's diff (a file, or the aggregate of all
// changed files under a folder — fixing the "no changed files" folder bug) between
// the PR's merge-base and head, through the user's pager. A directory selection uses
// every file under it, which is bounded by the tree's render cap.
func (self *PrReviewController) renderLocalRefDiff(ctx *context.PrReviewContext) {
	node := ctx.GetSelected()
	if node == nil {
		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: self.c.Tr.PrReviewTitle,
				Task:  types.NewRenderStringWithoutScrollTask(self.c.Tr.PrReviewNoChangedFiles),
			},
		})
		return
	}

	from, to := ctx.LocalDiffEndpoints()

	subtitle := ""
	if gen := ctx.HiddenGeneratedCount(); gen > 0 {
		subtitle = fmt.Sprintf(self.c.Tr.PrReviewGeneratedHiddenNote, gen)
	}

	// A file that carries review threads renders through the inline presenter so
	// EVERY reviewer's comments are visible on hover (not just the current user's,
	// and without having to enter the file) — the pager can't interleave threads.
	// Files without threads and folders keep the clean pager (delta) rendering.
	if node.IsFile() && ctx.SelectedFileHasThreads() {
		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title:    self.c.Tr.PrReviewTitle,
				SubTitle: subtitle,
				Task:     types.NewRenderStringWithoutScrollTask(ctx.RenderSelectedFileInlineDiff(self.c.Views().Main.InnerWidth())),
			},
		})
		return
	}

	// Partition the selection's files into unviewed and viewed. When a folder holds
	// both, split the main panel — unviewed diffs on top, viewed diffs on the bottom —
	// mirroring lazygit's unstaged/staged split, with "viewed" playing the staged role.
	var unviewed, viewed []string
	for _, p := range pathsForReviewDiff(node) {
		if ctx.IsPathViewed(p) {
			viewed = append(viewed, p)
		} else {
			unviewed = append(unviewed, p)
		}
	}

	diffTask := func(paths []string) types.UpdateTask {
		cmdObj := self.c.Git().WorkingTree.ShowFileDiffCmdObj(from, to, false, paths, false)
		return types.NewRunPtyTask(cmdObj.GetCmd())
	}

	if len(unviewed) > 0 && len(viewed) > 0 {
		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title:    self.c.Tr.PrReviewUnviewedTitle,
				SubTitle: subtitle,
				Task:     diffTask(unviewed),
			},
			Secondary: &types.ViewUpdateOpts{
				Title: self.c.Tr.PrReviewViewedTitle,
				Task:  diffTask(viewed),
			},
		})
		return
	}

	// A single file, or a folder whose files are all on the same side: one pane.
	self.c.RenderToMainViews(types.RefreshMainOpts{
		Pair: self.c.MainViewPairs().Normal,
		Main: &types.ViewUpdateOpts{
			Title:    self.c.Tr.PrReviewTitle,
			SubTitle: subtitle,
			Task:     diffTask(append(unviewed, viewed...)),
		},
	})
}

// pathsForReviewDiff returns the diff pathspec for a tree node: the single file path
// for a file node, or every changed file under a directory node (the aggregate diff).
func pathsForReviewDiff(node *filetree.CommitFileNode) []string {
	if node.IsFile() {
		return []string{node.GetPath()}
	}
	var paths []string
	_ = node.ForEachFile(func(file *models.CommitFile) error {
		paths = append(paths, file.Path)
		return nil
	})
	return paths
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

	// On a file node, open the focusable diff surface. SelectedIsFile works in both
	// the gh-API and local-ref modes (unlike SelectedFile, which reads the API map).
	if !tree.SelectedIsFile() {
		return nil
	}

	diffCtx := self.c.Contexts().PrReviewDiff
	self.c.Context().Push(diffCtx, types.OnFocusOpts{})
	diffCtx.PrepareForEntry()
	return nil
}

// toggleGenerated shows/hides generated files (lockfiles, *.generated.*, etc.) in the
// tree and toasts the new state.
func (self *PrReviewController) toggleGenerated() error {
	if self.context().ToggleShowGenerated() {
		self.c.Toast(self.c.Tr.PrReviewGeneratedShown)
	} else {
		self.c.Toast(self.c.Tr.PrReviewGeneratedHidden)
	}
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
