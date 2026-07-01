package presentation

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/i18n"
)

// RenderPrOverview renders the PR's headline details (title, number, state, author,
// base <- head branches) plus its description body for the Overview tab's main view.
// The body goes through renderMarkdown (glow when present). renderMarkdown may be nil.
func RenderPrOverview(
	tr *i18n.TranslationSet,
	data *git_commands.PullRequestReviewData,
	width int,
	renderMarkdown markdownRenderer,
) string {
	if data == nil {
		return tr.PrReviewLoading
	}

	sb := &strings.Builder{}
	bodyWidth := max(width, 10)

	// Title + PR number.
	sb.WriteString(style.FgWhite.SetBold().Sprintf("%s ", data.Title))
	sb.WriteString(style.FgCyan.Sprintf("#%d", data.Number))
	sb.WriteString("\n")

	// State + author.
	sb.WriteString(colorizeReviewState(data.State))
	if data.Author != "" {
		sb.WriteString(style.FgWhite.Sprint(" by "))
		sb.WriteString(style.FgMagenta.Sprint("@" + data.Author))
	}
	sb.WriteString("\n")

	// base <- head branches.
	if data.BaseRefName != "" || data.HeadRefName != "" {
		sb.WriteString(style.FgYellow.Sprint(nonEmpty(data.BaseRefName, "?")))
		sb.WriteString(style.FgWhite.Sprint(" ← "))
		sb.WriteString(style.FgGreen.Sprint(nonEmpty(data.HeadRefName, "?")))
		sb.WriteString("\n")
	}

	// Description body.
	sb.WriteString("\n")
	sb.WriteString(style.FgCyan.SetBold().Sprint(tr.PrReviewDescriptionSection))
	sb.WriteString("\n")
	if strings.TrimSpace(data.Body) == "" {
		sb.WriteString(tr.PrReviewNoDescription)
		sb.WriteString("\n")
	} else {
		for _, line := range renderBodyLines(data.Body, bodyWidth, renderMarkdown) {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
