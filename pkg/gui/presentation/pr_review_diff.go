package presentation

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/patch"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/theme"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// ReviewRowKind classifies every rendered line of the PR-review buffer. It is the
// parallel selection map that replaces patch_exploring.State (which hard-assumes
// view-line == patch-line, an invariant the interleaved comment rows break, see
// design D4). Only ReviewRowDiff rows are selectable; comment and header rows are
// skipped by the range-select logic in the write phase.
type ReviewRowKind int

const (
	// ReviewRowDiff is an actual diff body line (addition / deletion / context).
	// Its RowPatchIdx maps back to the index into patch.Lines() so a selection can
	// be translated to a file line number.
	ReviewRowDiff ReviewRowKind = iota
	// ReviewRowComment is part of an interleaved review-thread block (header,
	// author label, body, or border). Non-selectable.
	ReviewRowComment
	// ReviewRowHeader is a structural row (file header, hunk header, patch header,
	// section title). Non-selectable.
	ReviewRowHeader
)

// markdownRenderer renders a markdown body to ANSI-styled text wrapped to width.
// It mirrors Gui.renderMarkdown; the presenter takes it as a parameter so the
// presentation package stays free of the glow/Gui dependency and is unit-testable.
type markdownRenderer func(body string, width int) string

// RenderedReviewDiff is the output of the inline presenter: the final ANSI buffer
// plus the parallel selection maps (one entry per line of Content). RowKind,
// RowPatchIdx and RowThreadID are always the same length as the number of lines in
// Content. RowThreadID carries the review thread's GraphQL node ID on every row of
// that thread's block ("" elsewhere), so the cursor can select a thread to reply to
// or resolve (Phase 11).
type RenderedReviewDiff struct {
	Content     string
	RowKind     []ReviewRowKind
	RowPatchIdx []int
	RowThreadID []string
}

// reviewDiffBuilder accumulates the rendered buffer and the parallel row maps in
// lock-step so they can never drift apart.
type reviewDiffBuilder struct {
	sb          strings.Builder
	rowKind     []ReviewRowKind
	rowPatchIdx []int
	rowThreadID []string
}

func (b *reviewDiffBuilder) add(content string, kind ReviewRowKind, patchIdx int) {
	b.addRow(content, kind, patchIdx, "")
}

// addThreadRow is add for rows belonging to a review-thread block: it tags the row
// with the thread's node ID so it is selectable as that thread.
func (b *reviewDiffBuilder) addThreadRow(content string, threadID string) {
	b.addRow(content, ReviewRowComment, -1, threadID)
}

func (b *reviewDiffBuilder) addRow(content string, kind ReviewRowKind, patchIdx int, threadID string) {
	b.sb.WriteString(content)
	b.sb.WriteString("\n")
	b.rowKind = append(b.rowKind, kind)
	b.rowPatchIdx = append(b.rowPatchIdx, patchIdx)
	b.rowThreadID = append(b.rowThreadID, threadID)
}

func (b *reviewDiffBuilder) result() *RenderedReviewDiff {
	return &RenderedReviewDiff{
		Content:     b.sb.String(),
		RowKind:     b.rowKind,
		RowPatchIdx: b.rowPatchIdx,
		RowThreadID: b.rowThreadID,
	}
}

const reviewCommentPrefix = "│ "

// DiffMode selects how the review diff is laid out (design D9). Unified is the
// v1-primary, always-working path; SideBySide renders old/new in parallel columns.
type DiffMode int

const (
	// DiffModeUnified is the standard single-column unified diff.
	DiffModeUnified DiffMode = iota
	// DiffModeSideBySide lays old (LEFT) and new (RIGHT) content in two columns.
	DiffModeSideBySide
)

// ReviewFileDiffOpts bundles the inputs to RenderReviewFileDiff. It is an options
// struct rather than a long positional list because the presenter now also takes
// the reviewed flag (D8 indicator) and the diff mode (D9).
type ReviewFileDiffOpts struct {
	Tr             *i18n.TranslationSet
	Path           string
	Diff           string
	Threads        []models.ReviewThread
	Width          int
	RenderMarkdown markdownRenderer
	// Reviewed drives the per-file reviewed indicator in the file header (D8).
	Reviewed bool
	// DiffMode selects unified vs side-by-side rendering (D9). Unified is the
	// guaranteed path; SideBySide is gated so it never affects the selection map.
	DiffMode DiffMode
}

// RenderReviewFileDiff parses one file's unified diff and renders it with review
// threads interleaved at their anchored lines (design D4). For each ADDITION or
// CONTEXT line it computes the new-file line via Patch.LineNumberOfLine and, when
// a RIGHT-side thread anchors there, emits the diff line followed by the thread's
// comment block; for each DELETION or CONTEXT line it computes the old-file line
// via Patch.OldLineNumberOfLine and emits LEFT-side threads there (D4 both-side
// display). Threads with no live anchor (outdated, or a line absent from the diff)
// are collected into a trailing section so no comment is silently dropped.
func RenderReviewFileDiff(opts ReviewFileDiffOpts) *RenderedReviewDiff {
	tr := opts.Tr
	width := opts.Width
	renderMarkdown := opts.RenderMarkdown
	b := &reviewDiffBuilder{}

	// File header (with the reviewed indicator) so a single-file buffer is
	// self-describing.
	marker := tr.PrReviewUnreviewedMarker
	if opts.Reviewed {
		marker = style.FgGreen.Sprint(tr.PrReviewReviewedMarker)
	}
	b.add(marker+" "+theme.DefaultTextColor.SetBold().Sprint(opts.Path), ReviewRowHeader, -1)

	// Partition this file's threads into RIGHT-anchored (by new-file line),
	// LEFT-anchored (by old-file line), and trailing (outdated / no live anchor).
	rightAnchored := map[int][]*models.ReviewThread{}
	leftAnchored := map[int][]*models.ReviewThread{}
	trailing := []*models.ReviewThread{}
	for i := range opts.Threads {
		th := &opts.Threads[i]
		if th.Path != opts.Path {
			continue
		}
		switch {
		case th.IsOutdated || th.Line <= 0:
			trailing = append(trailing, th)
		case th.Side == "RIGHT":
			rightAnchored[th.Line] = append(rightAnchored[th.Line], th)
		case th.Side == "LEFT":
			leftAnchored[th.Line] = append(leftAnchored[th.Line], th)
		default:
			trailing = append(trailing, th)
		}
	}

	// Track which anchored threads actually got emitted so any whose anchor line
	// is absent from the current diff still land in the trailing section.
	emitted := map[string]bool{}

	colorize := colorizeReviewDiffLine
	if opts.DiffMode == DiffModeSideBySide {
		colorize = func(line *patch.PatchLine) string {
			return colorizeReviewDiffLineSideBySide(line, width)
		}
	}

	p := patch.Parse(opts.Diff)
	lines := p.Lines()
	for idx, line := range lines {
		b.add(colorize(line), reviewRowKindForPatchLine(line.Kind), idx)

		if line.Kind == patch.ADDITION || line.Kind == patch.CONTEXT {
			newLine := p.LineNumberOfLine(idx)
			for _, th := range rightAnchored[newLine] {
				appendCommentBlock(b, tr, th, width, renderMarkdown)
				emitted[th.ID] = true
			}
		}
		if line.Kind == patch.DELETION || line.Kind == patch.CONTEXT {
			oldLine := p.OldLineNumberOfLine(idx)
			for _, th := range leftAnchored[oldLine] {
				appendCommentBlock(b, tr, th, width, renderMarkdown)
				emitted[th.ID] = true
			}
		}
	}

	// Any anchored thread whose line was not present in the diff is rescued into
	// the trailing section (no comment may be silently dropped, D4).
	for _, bucket := range []map[int][]*models.ReviewThread{rightAnchored, leftAnchored} {
		for _, ths := range bucket {
			for _, th := range ths {
				if !emitted[th.ID] {
					trailing = append(trailing, th)
				}
			}
		}
	}

	if len(trailing) > 0 {
		b.add("", ReviewRowHeader, -1)
		b.add(style.FgYellow.Sprint(tr.PrReviewOutdatedSection), ReviewRowHeader, -1)
		for _, th := range trailing {
			// Show the captured diff context verbatim (it can't anchor to the
			// current diff) followed by the conversation.
			if th.DiffHunk != "" {
				for _, hunkLine := range strings.Split(strings.TrimSuffix(th.DiffHunk, "\n"), "\n") {
					b.add(colorizeRawDiffLine(hunkLine), ReviewRowHeader, -1)
				}
			}
			appendCommentBlock(b, tr, th, width, renderMarkdown)
		}
	}

	return b.result()
}

// appendCommentBlock renders a single review thread immediately under its anchor:
// a header line (first comment's author + resolved badge), then each comment's
// author label and body, all tagged ReviewRowComment so selection skips them.
func appendCommentBlock(
	b *reviewDiffBuilder,
	tr *i18n.TranslationSet,
	th *models.ReviewThread,
	width int,
	renderMarkdown markdownRenderer,
) {
	badge := style.FgYellow.Sprint(tr.PrReviewUnresolvedBadge)
	if th.IsResolved {
		badge = style.FgGreen.Sprint(tr.PrReviewResolvedBadge)
	}

	author := "(unknown)"
	if len(th.Comments) > 0 && th.Comments[0].Author != "" {
		author = th.Comments[0].Author
	}

	b.addThreadRow(
		style.FgCyan.Sprint("┌─ ")+style.FgMagenta.Sprint("@"+author)+" · "+badge,
		th.ID,
	)

	bodyWidth := max(width-len([]rune(reviewCommentPrefix)), 10)
	border := style.FgCyan.Sprint(reviewCommentPrefix)

	for ci, c := range th.Comments {
		// The first comment's author is already on the header line; reply authors
		// each get their own label line.
		if ci > 0 {
			label := style.FgMagenta.Sprint("@" + nonEmpty(c.Author, "(unknown)"))
			if c.ReplyToID != 0 {
				label += " " + theme.DefaultTextColor.Sprint(tr.PrReviewReplyLabel)
			}
			b.addThreadRow(border+label, th.ID)
		}

		for _, bodyLine := range renderBodyLines(c.Body, bodyWidth, renderMarkdown) {
			b.addThreadRow(border+bodyLine, th.ID)
		}
	}

	b.addThreadRow(style.FgCyan.Sprint("└"+strings.Repeat("─", max(width-1, 1))), th.ID)
}

// renderBodyLines turns a markdown body into display lines wrapped to width. When
// glow produced ANSI output (escape codes present) we trust its own wrapping and
// only split on newlines: WrapViewLinesToWidth counts escape bytes as width and
// would corrupt the colours. Plain (no-glow) bodies are wrapped normally.
func renderBodyLines(body string, width int, renderMarkdown markdownRenderer) []string {
	rendered := body
	if renderMarkdown != nil {
		rendered = renderMarkdown(body, width)
	}

	if strings.ContainsRune(rendered, '\x1b') {
		return strings.Split(strings.TrimSuffix(rendered, "\n"), "\n")
	}

	lines, _, _ := utils.WrapViewLinesToWidth(true, false, rendered, width, 4)
	return lines
}

func colorizeReviewDiffLine(line *patch.PatchLine) string {
	switch line.Kind {
	case patch.PATCH_HEADER:
		return theme.DefaultTextColor.SetBold().Sprint(line.Content)
	case patch.HUNK_HEADER:
		return style.FgCyan.Sprint(line.Content)
	case patch.ADDITION:
		return style.FgGreen.Sprint(line.Content)
	case patch.DELETION:
		return style.FgRed.Sprint(line.Content)
	default:
		return theme.DefaultTextColor.Sprint(line.Content)
	}
}

// colorizeReviewDiffLineSideBySide renders one diff body line as two columns
// (old | new). It keeps exactly one rendered row per patch line so the parallel
// selection map (rowKind/rowPatchIdx) stays byte-for-byte aligned and unified-mode
// selection logic is unaffected (design D9: side-by-side never breaks unified).
// This is the minimal v1 side-by-side: it truncates rather than wraps long lines.
func colorizeReviewDiffLineSideBySide(line *patch.PatchLine, width int) string {
	switch line.Kind {
	case patch.PATCH_HEADER:
		return theme.DefaultTextColor.SetBold().Sprint(line.Content)
	case patch.HUNK_HEADER:
		return style.FgCyan.Sprint(line.Content)
	default:
		// Body lines (addition/deletion/context) are rendered as two columns below.
	}

	colWidth := max((width-3)/2, 4)

	// Strip the leading +/-/space diff marker; the column itself conveys the side.
	text := line.Content
	if len(text) > 0 && (text[0] == '+' || text[0] == '-' || text[0] == ' ') {
		text = text[1:]
	}

	var left, right string
	switch line.Kind {
	case patch.ADDITION:
		right = "+ " + text
	case patch.DELETION:
		left = "- " + text
	default: // context: same content on both sides
		left = "  " + text
		right = "  " + text
	}

	leftCell := colorColumn(line.Kind, padOrTruncate(left, colWidth), false)
	rightCell := colorColumn(line.Kind, padOrTruncate(right, colWidth), true)
	return leftCell + style.FgCyan.Sprint(" │ ") + rightCell
}

// colorColumn colours a side-by-side column: deletions red on the left, additions
// green on the right, everything else default.
func colorColumn(kind patch.PatchLineKind, text string, isRight bool) string {
	switch {
	case kind == patch.ADDITION && isRight:
		return style.FgGreen.Sprint(text)
	case kind == patch.DELETION && !isRight:
		return style.FgRed.Sprint(text)
	default:
		return theme.DefaultTextColor.Sprint(text)
	}
}

// padOrTruncate right-pads (or truncates) a plain string to exactly width runes.
// It operates on the pre-colour text so ANSI escapes never skew the width.
func padOrTruncate(s string, width int) string {
	r := []rune(s)
	if len(r) > width {
		return string(r[:width])
	}
	return s + strings.Repeat(" ", width-len(r))
}

// colorizeRawDiffLine colours a verbatim diffHunk line (from an outdated thread)
// the same way as the live diff, keying off the leading character.
func colorizeRawDiffLine(line string) string {
	switch {
	case strings.HasPrefix(line, "@@"):
		return style.FgCyan.Sprint(line)
	case strings.HasPrefix(line, "+"):
		return style.FgGreen.Sprint(line)
	case strings.HasPrefix(line, "-"):
		return style.FgRed.Sprint(line)
	default:
		return theme.DefaultTextColor.Sprint(line)
	}
}

func reviewRowKindForPatchLine(kind patch.PatchLineKind) ReviewRowKind {
	switch kind {
	case patch.ADDITION, patch.DELETION, patch.CONTEXT:
		return ReviewRowDiff
	default:
		return ReviewRowHeader
	}
}

func nonEmpty(s string, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
