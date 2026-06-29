package context

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/patch"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// PrReviewDiffContext is the focusable main-window context the user enters (from
// the file tree, via Enter) to select a line range and add a review comment. It
// owns its own view-line<->patch-line map (RowKind/RowPatchIdx from the inline
// presenter) so the interleaved comment rows don't corrupt selection — the reason
// it can't reuse the generic PatchExplorerContext (design D4/D5). It renders the
// file the tree cursor is on, fetched live via ContextForKey so it survives the
// context-tree rebuilds that happen during startup.
type PrReviewDiffContext struct {
	*SimpleContext
	c *ContextCommon

	rendered    *presentation.RenderedReviewDiff
	file        *models.GithubPullRequestFile
	cachedWidth int

	// selectedRow is the view line index of the cursor (always a DIFF row, or -1);
	// rangeAnchor is the other end of the selection range.
	selectedRow int
	rangeAnchor int
}

var _ types.Context = (*PrReviewDiffContext)(nil)

func NewPrReviewDiffContext(c *ContextCommon) *PrReviewDiffContext {
	self := &PrReviewDiffContext{c: c, cachedWidth: -1, selectedRow: -1, rangeAnchor: -1}

	baseContext := NewBaseContext(NewBaseContextOpts{
		Kind:             types.MAIN_CONTEXT,
		View:             c.Views().PrReviewDiff,
		WindowName:       "main",
		Key:              PR_REVIEW_DIFF_CONTEXT_KEY,
		Focusable:        true,
		HighlightOnFocus: true,
		// The diff view is hidden until the user enters it, so it has no width when
		// first focused. This makes the layout call HandleRender once the view is
		// sized, so the content renders at the right width instead of staying blank.
		NeedsRerenderOnWidthChange: types.NEEDS_RERENDER_ON_WIDTH_CHANGE_WHEN_WIDTH_CHANGES,
	})
	self.SimpleContext = NewSimpleContext(baseContext)
	self.SimpleContext.SetHandleRenderFunc(self.onRender)

	return self
}

// PrepareForEntry is called by the controller right after pushing this context. It
// forces a fresh render of the file the tree cursor is now on and resets the
// selection. On first entry the view has no width yet, so the actual render happens
// when the layout sizes the view (NeedsRerenderOnWidthChange); on re-entry (a
// different file) the view already has width, so this renders immediately.
func (self *PrReviewDiffContext) PrepareForEntry() {
	self.cachedWidth = -1
	self.rendered = nil
	self.selectedRow = -1
	self.rangeAnchor = -1
	self.onRender()
}

// treeContext returns the live file-tree context (fetched by key, so it is always
// the current instance even after a context-tree rebuild).
func (self *PrReviewDiffContext) treeContext() *PrReviewContext {
	ctx, ok := self.c.ContextForKey(PR_REVIEW_CONTEXT_KEY).(*PrReviewContext)
	if !ok {
		return nil
	}
	return ctx
}

// onRender (re)builds the diff content at the current view width and applies the
// selection. It is width-safe: before the view is laid out (width 0) it no-ops, and
// the layout's width-change rerender calls it again once the view is sized.
func (self *PrReviewDiffContext) onRender() {
	view := self.c.Views().PrReviewDiff
	if view == nil {
		return
	}
	width := view.InnerWidth()
	if width <= 0 {
		return
	}

	tree := self.treeContext()
	if tree == nil {
		return
	}

	if self.rendered == nil || self.cachedWidth != width {
		self.rendered, self.file = tree.RenderedSelectedDiff(width)
		self.cachedWidth = width
		if self.rendered == nil {
			self.c.SetViewContent(view, "")
			return
		}
		self.c.SetViewContent(view, self.rendered.Content)
		if self.selectedRow < 0 || self.selectedRow >= len(self.rendered.RowKind) ||
			self.rendered.RowKind[self.selectedRow] != presentation.ReviewRowDiff {
			self.selectedRow = self.firstDiffRow()
			self.rangeAnchor = self.selectedRow
		}
	}
	self.applySelectionToView()
}

// MoveSelection moves the cursor to the next/prev DIFF row (delta ±1), skipping
// comment and header rows. extend holds the range anchor so a multi-line range grows.
func (self *PrReviewDiffContext) MoveSelection(delta int, extend bool) {
	next := self.nextDiffRow(self.selectedRow, delta)
	if next == -1 {
		return
	}
	self.selectedRow = next
	if !extend {
		self.rangeAnchor = next
	}
	self.applySelectionToView()
	self.c.Render()
}

func (self *PrReviewDiffContext) rows() []presentation.ReviewRowKind {
	if self.rendered == nil {
		return nil
	}
	return self.rendered.RowKind
}

func (self *PrReviewDiffContext) firstDiffRow() int {
	for i, kind := range self.rows() {
		if kind == presentation.ReviewRowDiff {
			return i
		}
	}
	return -1
}

func (self *PrReviewDiffContext) nextDiffRow(from int, delta int) int {
	rows := self.rows()
	if delta == 0 || from < 0 {
		return from
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	for i := from + step; i >= 0 && i < len(rows); i += step {
		if rows[i] == presentation.ReviewRowDiff {
			return i
		}
	}
	return -1
}

// applySelectionToView scrolls the cursor row into view and sets the gocui
// range-select anchor + cursor so the highlighted range matches [rangeAnchor, selectedRow].
func (self *PrReviewDiffContext) applySelectionToView() {
	view := self.c.Views().PrReviewDiff
	if view == nil || self.selectedRow < 0 {
		return
	}

	viewHeight := view.InnerHeight()
	_, origin := view.Origin()
	numLines := view.ViewLinesHeight()
	newOrigin := clampReviewOrigin(self.selectedRow, origin, viewHeight, numLines)
	view.SetOriginY(newOrigin)

	view.SetRangeSelectStart(self.rangeAnchor)
	view.SetCursorY(self.selectedRow - newOrigin)
}

// SelectedCommentTarget translates the current [rangeAnchor, selectedRow] DIFF-row
// range into GitHub add-comment parameters: the file path and the new-file start/end
// line numbers (RIGHT side). Returns a non-empty errMsg when nothing is selected or
// when either endpoint is a deletion (LEFT-side) line, which v1 cannot comment on.
func (self *PrReviewDiffContext) SelectedCommentTarget() (path string, startLine int, line int, errMsg string) {
	if self.rendered == nil || self.file == nil || self.selectedRow < 0 || self.rangeAnchor < 0 {
		return "", 0, 0, self.c.Tr.PrReviewNoDiffSelected
	}

	startRow := min(self.rangeAnchor, self.selectedRow)
	endRow := max(self.rangeAnchor, self.selectedRow)
	rowKind := self.rendered.RowKind
	if startRow < 0 || endRow >= len(rowKind) {
		return "", 0, 0, self.c.Tr.PrReviewNoDiffSelected
	}
	if rowKind[startRow] != presentation.ReviewRowDiff || rowKind[endRow] != presentation.ReviewRowDiff {
		return "", 0, 0, self.c.Tr.PrReviewNoDiffSelected
	}

	p := patch.Parse(self.file.Patch)
	lines := p.Lines()
	startFileLine, ok1 := rightSideFileLine(p, lines, self.rendered.RowPatchIdx[startRow])
	endFileLine, ok2 := rightSideFileLine(p, lines, self.rendered.RowPatchIdx[endRow])
	if !ok1 || !ok2 {
		return "", 0, 0, self.c.Tr.PrReviewCommentLeftSideUnsupported
	}

	if startFileLine > endFileLine {
		startFileLine, endFileLine = endFileLine, startFileLine
	}

	return self.file.Filename, startFileLine, endFileLine, ""
}

// rightSideFileLine returns the new-file line number for a patch line, rejecting
// deletion lines (which live on the LEFT side and have no new-file line).
func rightSideFileLine(p *patch.Patch, lines []*patch.PatchLine, idx int) (int, bool) {
	if idx < 0 || idx >= len(lines) {
		return 0, false
	}
	if lines[idx].Kind == patch.DELETION {
		return 0, false
	}
	return p.LineNumberOfLine(idx), true
}

// clampReviewOrigin returns a vertical origin that keeps the selected row visible
// without recentring when it is already on screen.
func clampReviewOrigin(selected int, origin int, viewHeight int, numLines int) int {
	if viewHeight <= 0 || viewHeight >= numLines {
		return 0
	}
	if selected < origin {
		return selected
	}
	if selected >= origin+viewHeight {
		return selected - viewHeight + 1
	}
	return origin
}
