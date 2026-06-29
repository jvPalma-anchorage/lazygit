package presentation

import (
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
