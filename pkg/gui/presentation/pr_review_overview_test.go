package presentation

import (
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func overviewFixture() *git_commands.PullRequestReviewData {
	return &git_commands.PullRequestReviewData{
		Number: 42, Title: "Add widgets", State: "OPEN", Author: "alice",
		BaseRefName: "main", HeadRefName: "feature-x",
		Body: "DESCRIPTION_BODY",
		Labels: []models.PrLabel{
			{Name: "bug", Color: "d73a4a"},
			{Name: "frontend", Color: "f9e2af"},
		},
		IssueComments: []models.IssueComment{
			{Author: "human", Body: "SECOND_ENTRY", CreatedAt: "2026-02-01T00:00:00Z"},
			{Author: "ci-bot[bot]", Body: "FIRST_ENTRY", CreatedAt: "2026-01-01T00:00:00Z"},
		},
		Reviews: []models.Review{
			{Author: "rev", State: "APPROVED", Body: "THIRD_ENTRY", SubmittedAt: "2026-03-01T00:00:00Z"},
		},
	}
}

func TestRenderPrOverviewLayoutOrder(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	out := utils.Decolorise(RenderPrOverview(tr, overviewFixture(), 80, nil))

	// Every block present, in the specified order.
	blocks := []string{
		"#42 - Add widgets",
		"OPEN by @alice",
		"main ← feature-x",
		"bug", "frontend",
		"DESCRIPTION_BODY",
		tr.PrReviewTimelineSection,
		"FIRST_ENTRY",
	}
	lastIdx := -1
	for _, block := range blocks {
		idx := strings.Index(out, block)
		assert.GreaterOrEqual(t, idx, 0, "missing block %q", block)
		assert.Greater(t, idx, lastIdx, "block %q out of order", block)
		lastIdx = idx
	}
}

func TestRenderPrOverviewTimelineIsChronologicalAndIncludesBots(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	out := utils.Decolorise(RenderPrOverview(tr, overviewFixture(), 80, nil))

	// Oldest first: the bot's January comment precedes the February human comment,
	// which precedes the March review summary.
	first := strings.Index(out, "FIRST_ENTRY")
	second := strings.Index(out, "SECOND_ENTRY")
	third := strings.Index(out, "THIRD_ENTRY")
	assert.True(t, first >= 0 && second >= 0 && third >= 0)
	assert.Less(t, first, second)
	assert.Less(t, second, third)

	// The bot's entry is attributed, not filtered.
	assert.Contains(t, out, "@ci-bot[bot]")
	// Timestamps render compactly.
	assert.Contains(t, out, "2026-01-01")
	// The review entry carries its state badge.
	assert.Contains(t, out, "APPROVED")
}

func TestRenderPrOverviewEmptyBlocksRenderWithoutError(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	data := &git_commands.PullRequestReviewData{Number: 7, Title: "Bare", State: "MERGED"}
	out := utils.Decolorise(RenderPrOverview(tr, data, 80, nil))

	assert.Contains(t, out, "#7 - Bare")
	assert.Contains(t, out, tr.PrReviewNoDescription)
	assert.Contains(t, out, tr.PrReviewTimelineEmpty)
}

func TestLabelChipPicksReadableForeground(t *testing.T) {
	// Dark background → white text; light background → black text. We can't easily
	// assert ANSI codes portably, but relativeLuminance drives the choice directly.
	assert.Less(t, relativeLuminance("d73a4a"), 0.5, "GitHub red is dark → white fg")
	assert.Greater(t, relativeLuminance("f9e2af"), 0.5, "pale yellow is light → black fg")

	// A malformed color still renders the name rather than exploding — including
	// 6-char garbage that is length-valid but not hex (gookit would silently
	// produce an empty color).
	for _, bad := range []string{"zz", "gggggg", ""} {
		out := utils.Decolorise(labelChip(models.PrLabel{Name: "odd", Color: bad}))
		assert.Contains(t, out, "odd")
	}
	assert.False(t, isHexColor("gggggg"))
	assert.True(t, isHexColor("d73a4a"))
}

func TestPrTimelineEntriesSinksUndatedToEnd(t *testing.T) {
	data := &git_commands.PullRequestReviewData{
		IssueComments: []models.IssueComment{
			{Author: "undated", Body: "u", CreatedAt: ""},
			{Author: "dated", Body: "d", CreatedAt: "2026-01-01T00:00:00Z"},
		},
	}
	entries := prTimelineEntries(data)
	assert.Equal(t, "dated", entries[0].author)
	assert.Equal(t, "undated", entries[1].author)
}
