package presentation

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/theme"
)

// RenderReviewConversation renders the PR's global (issue-level) comments and the
// reviewers + their latest states as a plain ANSI string for a side section
// (task 4.7). Authors come from the models; bodies go through renderMarkdown
// (glow when present, plain otherwise). renderMarkdown may be nil.
func RenderReviewConversation(
	tr *i18n.TranslationSet,
	issueComments []models.IssueComment,
	reviewers []models.Reviewer,
	width int,
	renderMarkdown markdownRenderer,
) string {
	sb := &strings.Builder{}

	sb.WriteString(style.FgCyan.SetBold().Sprint(tr.PrReviewReviewersSection))
	sb.WriteString("\n")
	if len(reviewers) == 0 {
		sb.WriteString(theme.DefaultTextColor.Sprint("—"))
		sb.WriteString("\n")
	}
	for _, r := range reviewers {
		sb.WriteString(style.FgMagenta.Sprint("@" + r.Login))
		sb.WriteString(" ")
		sb.WriteString(colorizeReviewState(r.State))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(style.FgCyan.SetBold().Sprint(tr.PrReviewGlobalCommentsSection))
	sb.WriteString("\n")
	bodyWidth := max(width, 10)
	for _, c := range issueComments {
		sb.WriteString(style.FgMagenta.Sprint("@" + nonEmpty(c.Author, "(unknown)")))
		sb.WriteString("\n")
		for _, line := range renderBodyLines(c.Body, bodyWidth, renderMarkdown) {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderReviewerDetail renders one reviewer's contribution for the Conversation tab's
// main view (Phase 6): their review summary body + state, their inline thread comments
// (with per-thread resolved/unresolved/outdated state), and their PR-body comments.
// Everything is filtered to the given reviewer. Bodies go through renderMarkdown (glow
// when present). renderMarkdown may be nil.
func RenderReviewerDetail(
	tr *i18n.TranslationSet,
	reviewer models.Reviewer,
	reviews []models.Review,
	threads []models.ReviewThread,
	issueComments []models.IssueComment,
	width int,
	renderMarkdown markdownRenderer,
) string {
	sb := &strings.Builder{}
	bodyWidth := max(width, 10)
	login := reviewer.Login

	sb.WriteString(style.FgMagenta.SetBold().Sprint("@" + nonEmpty(login, "(unknown)")))
	sb.WriteString(" ")
	sb.WriteString(colorizeReviewState(reviewer.State))
	sb.WriteString("\n")

	rendered := false

	// Review summary bodies submitted by this reviewer.
	summarySection := false
	for _, r := range reviews {
		if r.Author != login || strings.TrimSpace(r.Body) == "" {
			continue
		}
		if !summarySection {
			sb.WriteString("\n")
			sb.WriteString(style.FgCyan.SetBold().Sprint(tr.PrReviewReviewSummarySection))
			sb.WriteString("\n")
			summarySection = true
		}
		for _, line := range renderBodyLines(r.Body, bodyWidth, renderMarkdown) {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		rendered = true
	}

	// Inline thread comments the reviewer wrote, grouped by thread with its state.
	inlineSection := false
	for _, t := range threads {
		theirs := commentsByAuthor(t.Comments, login)
		if len(theirs) == 0 {
			continue
		}
		if !inlineSection {
			sb.WriteString("\n")
			sb.WriteString(style.FgCyan.SetBold().Sprint(tr.PrReviewInlineCommentsSection))
			sb.WriteString("\n")
			inlineSection = true
		}
		sb.WriteString(style.FgYellow.Sprint(threadLocation(t)))
		sb.WriteString(" ")
		sb.WriteString(threadStateBadge(tr, t))
		sb.WriteString("\n")
		for _, c := range theirs {
			for _, line := range renderBodyLines(c.Body, bodyWidth, renderMarkdown) {
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
		rendered = true
	}

	// PR-body (issue-level) comments by this reviewer.
	issueSection := false
	for _, c := range issueComments {
		if c.Author != login {
			continue
		}
		if !issueSection {
			sb.WriteString("\n")
			sb.WriteString(style.FgCyan.SetBold().Sprint(tr.PrReviewGlobalCommentsSection))
			sb.WriteString("\n")
			issueSection = true
		}
		for _, line := range renderBodyLines(c.Body, bodyWidth, renderMarkdown) {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		rendered = true
	}

	if !rendered {
		sb.WriteString("\n")
		sb.WriteString(theme.DefaultTextColor.Sprint(tr.PrReviewReviewerNoActivity))
		sb.WriteString("\n")
	}

	return sb.String()
}

// commentsByAuthor returns the comments authored by login.
func commentsByAuthor(comments []models.ReviewComment, login string) []models.ReviewComment {
	result := make([]models.ReviewComment, 0, len(comments))
	for _, c := range comments {
		if c.Author == login {
			result = append(result, c)
		}
	}
	return result
}

// threadLocation is the "path:line" (or just "path" for an outdated/unanchored
// thread) label for a thread header.
func threadLocation(t models.ReviewThread) string {
	if t.Line > 0 {
		return fmt.Sprintf("%s:%d", t.Path, t.Line)
	}
	return t.Path
}

// threadStateBadge shows a thread's resolution (resolved green / unresolved red) and,
// when the anchored code has moved, an additional outdated badge — the two are
// orthogonal, so a resolved-and-outdated thread surfaces both.
func threadStateBadge(tr *i18n.TranslationSet, t models.ReviewThread) string {
	badge := style.FgRed.Sprint(tr.PrReviewUnresolvedBadge)
	if t.IsResolved {
		badge = style.FgGreen.Sprint(tr.PrReviewResolvedBadge)
	}
	if t.IsOutdated {
		badge += " " + style.FgYellow.Sprint(tr.PrReviewOutdatedBadge)
	}
	return badge
}

// colorizeReviewState colours a review state badge: approvals green, requested
// changes red, pending/requested yellow, everything else default.
func colorizeReviewState(state string) string {
	switch state {
	case "APPROVED":
		return style.FgGreen.Sprint(state)
	case "CHANGES_REQUESTED":
		return style.FgRed.Sprint(state)
	case "PENDING":
		return style.FgYellow.Sprint(state)
	default:
		return theme.DefaultTextColor.Sprint(state)
	}
}
