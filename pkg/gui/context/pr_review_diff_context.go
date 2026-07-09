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

	rendered *presentation.RenderedReviewDiff
	// path and patch are the cursor file's path and raw unified patch, sourced
	// mode-agnostically (gh-API patch or a plain local `git diff`) from the tree.
	// SelectedCommentTarget maps line numbers from this exact patch string.
	path        string
	patch       string
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

// Invalidate drops the cached render so the next onRender re-pulls the diff (and any
// newly-loaded review threads). Used after a comment post + data reload so the new
// comment appears inline without the user re-entering the file.
func (self *PrReviewDiffContext) Invalidate() {
	self.rendered = nil
	self.cachedWidth = -1
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
		self.rendered, self.path, self.patch = tree.SelectedFileDiff(width)
		self.cachedWidth = width
		if self.rendered == nil {
			self.c.SetViewContent(view, "")
			return
		}
		self.c.SetViewContent(view, self.rendered.Content)
		if self.selectedRow < 0 || self.selectedRow >= len(self.rendered.RowKind) ||
			(self.rendered.RowKind[self.selectedRow] != presentation.ReviewRowDiff &&
				self.rendered.RowKind[self.selectedRow] != presentation.ReviewRowComment) {
			self.selectedRow = self.firstDiffRow()
			self.rangeAnchor = self.selectedRow
		}
	}
	self.applySelectionToView()
}

// MoveSelection moves the cursor to the next/prev selectable row (delta ±1). A plain
// move lands on DIFF rows and on review-thread (comment) rows — threads are selectable
// anchors for reply/resolve (Phase 11). An extending move (range select) only walks
// DIFF rows, since a comment range is meaningless across a thread block; extending
// from a thread row collapses to a plain move.
func (self *PrReviewDiffContext) MoveSelection(delta int, extend bool) {
	if extend && self.rowThreadID(self.selectedRow) != "" {
		extend = false
	}
	next := nextSelectableRowIn(self.rows(), self.selectedRow, delta, !extend)
	if next == -1 {
		return
	}
	self.selectedRow = next
	if !extend || self.rowThreadID(next) != "" {
		self.rangeAnchor = next
	}
	self.applySelectionToView()
	self.c.Render()
}

// rowThreadID returns the review-thread node ID the given view row belongs to, or ""
// when the row is not part of a thread block.
func (self *PrReviewDiffContext) rowThreadID(row int) string {
	if self.rendered == nil || row < 0 || row >= len(self.rendered.RowThreadID) {
		return ""
	}
	return self.rendered.RowThreadID[row]
}

// SelectedThread returns the review thread the cursor is on, or nil when the cursor
// is on a diff row. The thread is looked up live in the tree context's data so its
// IsResolved state reflects the latest reload.
func (self *PrReviewDiffContext) SelectedThread() *models.ReviewThread {
	id := self.rowThreadID(self.selectedRow)
	if id == "" {
		return nil
	}
	tree := self.treeContext()
	if tree == nil || tree.Data() == nil {
		return nil
	}
	threads := tree.Data().Threads
	for i := range threads {
		if threads[i].ID == id {
			return &threads[i]
		}
	}
	return nil
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

// nextSelectableRowIn walks from `from` in the direction of delta and returns the
// first row that is selectable: always DIFF rows, plus thread (comment) rows when
// includeThreads is set. -1 when there is nothing further in that direction. Pure so
// the navigation semantics are unit-testable.
func nextSelectableRowIn(rows []presentation.ReviewRowKind, from int, delta int, includeThreads bool) int {
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
		if includeThreads && rows[i] == presentation.ReviewRowComment {
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
	if self.rendered == nil || self.patch == "" || self.path == "" || self.selectedRow < 0 || self.rangeAnchor < 0 {
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

	startFileLine, endFileLine, ok := reviewCommentRange(self.patch, self.rendered.RowPatchIdx[startRow], self.rendered.RowPatchIdx[endRow])
	if !ok {
		return "", 0, 0, self.c.Tr.PrReviewCommentLeftSideUnsupported
	}
	return self.path, startFileLine, endFileLine, ""
}

// reviewCommentRange maps two patch-line indices (the selection endpoints) to the
// GitHub new-file (RIGHT-side) line numbers for an add-comment call. It parses the
// unified patch — which may be a full `git diff` (with header) or a hunk-only gh-API
// patch; the @@-derived line numbers are identical either way, matching GitHub's
// pulls/{n}/files coordinates. ok is false when either endpoint is a deletion line
// (LEFT side, which v1 cannot comment on). The returned range is start<=end.
//
// Only the endpoints are required to be RIGHT-side: GitHub anchors a multi-line
// comment by its start_line..line RIGHT-side line numbers, and deletions lying
// between them are irrelevant to the API. Rejecting interior deletions would block
// commenting on any modified block (every modification is a delete + an add), so it
// is deliberately NOT done.
func reviewCommentRange(patchStr string, startPatchIdx int, endPatchIdx int) (int, int, bool) {
	p := patch.Parse(patchStr)
	lines := p.Lines()
	startFileLine, ok1 := rightSideFileLine(p, lines, startPatchIdx)
	endFileLine, ok2 := rightSideFileLine(p, lines, endPatchIdx)
	if !ok1 || !ok2 {
		return 0, 0, false
	}
	if startFileLine > endFileLine {
		startFileLine, endFileLine = endFileLine, startFileLine
	}
	return startFileLine, endFileLine, true
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
