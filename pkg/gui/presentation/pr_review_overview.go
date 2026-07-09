package presentation

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// RenderPrOverview renders the Overview tab's main-view content (spec "PR overview",
// Phase 12), in this order: a prominent `#number - title` heading; the state + author
// line; the `base ← head` branch line; the PR's labels as colored chips (GitHub label
// colors); a separator; the markdown-rendered description; a separator; and the
// conversation timeline — issue comments and submitted review summaries (bots
// included) merged oldest-first, each with author and timestamp. Empty blocks render
// without error. renderMarkdown may be nil (plain text).
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

	// Heading: `#42 - Title`, bold with an underline rule — the terminal's H1.
	heading := fmt.Sprintf("#%d - %s", data.Number, data.Title)
	sb.WriteString(style.FgWhite.SetBold().Sprint(heading))
	sb.WriteString("\n")
	sb.WriteString(style.FgBlue.Sprint(strings.Repeat("═", min(utils.StringWidth(heading), bodyWidth))))
	sb.WriteString("\n")

	// State + author.
	sb.WriteString(colorizeReviewState(data.State))
	if data.Author != "" {
		sb.WriteString(style.FgWhite.Sprint(" by "))
		sb.WriteString(style.FgMagenta.Sprint("@" + data.Author))
	}
	sb.WriteString("\n")

	// base ← head branches.
	if data.BaseRefName != "" || data.HeadRefName != "" {
		sb.WriteString(style.FgYellow.Sprint(nonEmpty(data.BaseRefName, "?")))
		sb.WriteString(style.FgWhite.Sprint(" ← "))
		sb.WriteString(style.FgGreen.Sprint(nonEmpty(data.HeadRefName, "?")))
		sb.WriteString("\n")
	}

	// Label chips in the label's own GitHub color.
	if len(data.Labels) > 0 {
		chips := make([]string, 0, len(data.Labels))
		for _, label := range data.Labels {
			chips = append(chips, labelChip(label))
		}
		sb.WriteString(strings.Join(chips, " "))
		sb.WriteString("\n")
	}

	separator := style.FgBlue.Sprint(strings.Repeat("─", bodyWidth))

	// Description.
	sb.WriteString(separator)
	sb.WriteString("\n\n")
	if strings.TrimSpace(data.Body) == "" {
		sb.WriteString(tr.PrReviewNoDescription)
		sb.WriteString("\n")
	} else {
		for _, line := range renderBodyLines(data.Body, bodyWidth, renderMarkdown) {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	// Timeline: issue comments + submitted review summaries, oldest first.
	sb.WriteString(separator)
	sb.WriteString("\n\n")
	sb.WriteString(style.FgCyan.SetBold().Sprint(tr.PrReviewTimelineSection))
	sb.WriteString("\n")
	entries := prTimelineEntries(data)
	if len(entries) == 0 {
		sb.WriteString(tr.PrReviewTimelineEmpty)
		sb.WriteString("\n")
	}
	for _, entry := range entries {
		sb.WriteString("\n")
		sb.WriteString(style.FgCyan.Sprint("● "))
		sb.WriteString(style.FgMagenta.Sprint("@" + entry.author))
		sb.WriteString(style.FgWhite.Sprint(" · " + formatReviewTimestamp(entry.when)))
		if entry.badge != "" {
			sb.WriteString(" " + entry.badge)
		}
		sb.WriteString("\n")
		body := strings.TrimSpace(entry.body)
		if body == "" {
			continue
		}
		for _, line := range renderBodyLines(body, bodyWidth-2, renderMarkdown) {
			sb.WriteString("  ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

type prTimelineEntry struct {
	author string
	when   string
	badge  string
	body   string
}

// prTimelineEntries merges the PR's issue comments and submitted reviews into one
// list sorted ascending by timestamp (RFC3339 strings compare correctly as strings;
// entries with no timestamp sink to the end). Bot authors flow through untouched.
func prTimelineEntries(data *git_commands.PullRequestReviewData) []prTimelineEntry {
	entries := make([]prTimelineEntry, 0, len(data.IssueComments)+len(data.Reviews))
	for _, c := range data.IssueComments {
		entries = append(entries, prTimelineEntry{author: c.Author, when: c.CreatedAt, body: c.Body})
	}
	for _, r := range data.Reviews {
		entries = append(entries, prTimelineEntry{
			author: r.Author,
			when:   r.SubmittedAt,
			badge:  colorizeReviewState(r.State),
			body:   r.Body,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if (entries[i].when == "") != (entries[j].when == "") {
			return entries[j].when == ""
		}
		return entries[i].when < entries[j].when
	})
	return entries
}

// formatReviewTimestamp renders an RFC3339 GitHub timestamp as a compact local
// date-time; anything unparseable is shown raw rather than dropped.
func formatReviewTimestamp(ts string) string {
	if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
		return parsed.Local().Format("2006-01-02 15:04")
	}
	return ts
}

// labelChip renders one PR label as a chip: the label name on its GitHub hex color,
// with a black/white foreground chosen by background luminance for readability. A
// malformed color (wrong length OR non-hex characters — gookit silently yields an
// empty color for the latter, losing the background entirely) falls back to a plain
// bold chip rather than unreadable text.
func labelChip(label models.PrLabel) string {
	hex := strings.TrimPrefix(label.Color, "#")
	if !isHexColor(hex) {
		return style.FgDefault.SetBold().Sprint(" " + label.Name + " ")
	}
	fg := style.FgBlack
	if relativeLuminance(hex) < 0.5 {
		fg = style.FgWhite
	}
	bg := style.New().SetBg(style.NewRGBColor(color.HEX(hex, true)))
	return bg.MergeStyle(fg).Sprint(" " + label.Name + " ")
}

// relativeLuminance approximates a hex color's perceived brightness in [0,1].
func relativeLuminance(hex string) float64 {
	rgb := color.HEX(hex, false)
	return (0.299*float64(rgb[0]) + 0.587*float64(rgb[1]) + 0.114*float64(rgb[2])) / 255
}

// isHexColor reports whether s is exactly six hexadecimal digits.
func isHexColor(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
