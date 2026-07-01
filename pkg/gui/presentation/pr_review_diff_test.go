package presentation

import (
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func testTr() *i18n.TranslationSet {
	return i18n.EnglishTranslationSet()
}

// rowOfFirstLineContaining returns the index of the first buffer line that
// contains substr, or -1.
func rowOfFirstLineContaining(content string, substr string) int {
	for i, line := range strings.Split(content, "\n") {
		if strings.Contains(line, substr) {
			return i
		}
	}
	return -1
}

func TestRenderReviewFileDiff_AnchorsRightSideThread(t *testing.T) {
	// REST `patch` fields have no diff --git/---/+++ header; they start at @@.
	diff := "@@ -1,2 +1,3 @@\n" +
		" context_a\n" +
		"+added_b\n" +
		" context_c\n"

	threads := []models.ReviewThread{
		{
			Path:       "file.go",
			Side:       "RIGHT",
			Line:       2, // new-file line of "+added_b"
			IsResolved: true,
			Comments: []models.ReviewComment{
				{Author: "alice", Body: "looks good"},
				{Author: "bob", Body: "agreed", ReplyToID: 1},
			},
		},
	}

	rendered := RenderReviewFileDiff(ReviewFileDiffOpts{
		Tr:      testTr(),
		Path:    "file.go",
		Diff:    diff,
		Threads: threads,
		Width:   80,
	})

	// The maps stay in lock-step with the buffer.
	assert.Equal(t, len(rendered.RowKind), len(rendered.RowPatchIdx))
	assert.Equal(t, strings.Count(rendered.Content, "\n"), len(rendered.RowKind))

	// The comment block renders under its anchor with author + resolved badge.
	assert.Contains(t, rendered.Content, "@alice")
	assert.Contains(t, rendered.Content, "looks good")
	assert.Contains(t, rendered.Content, "@bob")
	assert.Contains(t, rendered.Content, "agreed")
	assert.Contains(t, rendered.Content, testTr().PrReviewResolvedBadge)

	// The comment block immediately follows the anchored diff line.
	anchorRow := rowOfFirstLineContaining(rendered.Content, "added_b")
	assert.GreaterOrEqual(t, anchorRow, 0)
	assert.Equal(t, ReviewRowDiff, rendered.RowKind[anchorRow])
	assert.Equal(t, ReviewRowComment, rendered.RowKind[anchorRow+1])

	// Comment rows are non-selectable (patch index -1).
	commentRow := rowOfFirstLineContaining(rendered.Content, "looks good")
	assert.Equal(t, ReviewRowComment, rendered.RowKind[commentRow])
	assert.Equal(t, -1, rendered.RowPatchIdx[commentRow])

	// A diff row carries its index into patch.Lines() so a selection can map back
	// to a file line. "+added_b" is patch line index 2 (idx 0 is the hunk header).
	assert.Equal(t, 2, rendered.RowPatchIdx[anchorRow])
}

func TestRenderReviewFileDiff_OutdatedThreadInTrailingSection(t *testing.T) {
	diff := "@@ -1,1 +1,1 @@\n context_a\n"

	threads := []models.ReviewThread{
		{
			Path:       "file.go",
			Side:       "RIGHT",
			IsOutdated: true, // no live anchor; Line stays 0
			DiffHunk:   "@@ -10,2 +10,2 @@\n-gone_line\n+replacement",
			Comments: []models.ReviewComment{
				{Author: "carol", Body: "this changed"},
			},
		},
	}

	rendered := RenderReviewFileDiff(ReviewFileDiffOpts{
		Tr:      testTr(),
		Path:    "file.go",
		Diff:    diff,
		Threads: threads,
		Width:   80,
	})

	// The outdated thread is not anchored inline; it lands in the trailing section
	// with its diffHunk shown verbatim.
	assert.Contains(t, rendered.Content, testTr().PrReviewOutdatedSection)
	assert.Contains(t, rendered.Content, "gone_line")
	assert.Contains(t, rendered.Content, "this changed")
	assert.Contains(t, rendered.Content, "@carol")
}

func TestRenderReviewFileDiff_AnchorsLeftSideDeletionThread(t *testing.T) {
	diff := "@@ -1,3 +1,2 @@\n" +
		" context_a\n" +
		"-deleted_b\n" +
		" context_c\n"

	threads := []models.ReviewThread{
		{
			Path: "file.go",
			Side: "LEFT",
			Line: 2, // old-file line of "-deleted_b"
			Comments: []models.ReviewComment{
				{Author: "alice", Body: "why remove this?"},
			},
		},
	}

	rendered := RenderReviewFileDiff(ReviewFileDiffOpts{
		Tr:      testTr(),
		Path:    "file.go",
		Diff:    diff,
		Threads: threads,
		Width:   80,
	})

	// The maps stay in lock-step with the buffer.
	assert.Equal(t, len(rendered.RowKind), len(rendered.RowPatchIdx))
	assert.Equal(t, strings.Count(rendered.Content, "\n"), len(rendered.RowKind))

	// The LEFT-side thread is anchored inline (not dropped into the trailing
	// outdated section).
	assert.Contains(t, rendered.Content, "why remove this?")
	assert.NotContains(t, rendered.Content, testTr().PrReviewOutdatedSection)

	// The comment block immediately follows the deletion line it anchors to.
	anchorRow := rowOfFirstLineContaining(rendered.Content, "deleted_b")
	assert.GreaterOrEqual(t, anchorRow, 0)
	assert.Equal(t, ReviewRowDiff, rendered.RowKind[anchorRow])
	assert.Equal(t, ReviewRowComment, rendered.RowKind[anchorRow+1])
}

func TestRenderReviewFileDiff_ReviewedMarker(t *testing.T) {
	diff := "@@ -1,1 +1,1 @@\n context_a\n"

	reviewed := RenderReviewFileDiff(ReviewFileDiffOpts{
		Tr: testTr(), Path: "file.go", Diff: diff, Width: 80, Reviewed: true,
	})
	unreviewed := RenderReviewFileDiff(ReviewFileDiffOpts{
		Tr: testTr(), Path: "file.go", Diff: diff, Width: 80, Reviewed: false,
	})

	assert.Contains(t, reviewed.Content, testTr().PrReviewReviewedMarker)
	assert.Contains(t, unreviewed.Content, testTr().PrReviewUnreviewedMarker)
}

func TestRenderReviewFileDiff_SideBySidePreservesSelectionMap(t *testing.T) {
	diff := "@@ -1,2 +1,2 @@\n context_a\n-deleted_b\n+added_b\n"

	unified := RenderReviewFileDiff(ReviewFileDiffOpts{
		Tr: testTr(), Path: "file.go", Diff: diff, Width: 80, DiffMode: DiffModeUnified,
	})
	sideBySide := RenderReviewFileDiff(ReviewFileDiffOpts{
		Tr: testTr(), Path: "file.go", Diff: diff, Width: 80, DiffMode: DiffModeSideBySide,
	})

	// Side-by-side keeps exactly one row per patch line, so the selection map is
	// identical to unified (design D9: side-by-side never breaks selection).
	assert.Equal(t, unified.RowKind, sideBySide.RowKind)
	assert.Equal(t, unified.RowPatchIdx, sideBySide.RowPatchIdx)
	// And the buffer stays a column layout (a separator between old and new).
	assert.Contains(t, sideBySide.Content, "│")
}

func TestRenderReviewConversation_ReviewersAndComments(t *testing.T) {
	comments := []models.IssueComment{{Author: "dave", Body: "ship it"}}
	reviewers := []models.Reviewer{
		{Login: "erin", State: "APPROVED"},
		{Login: "frank", State: "PENDING"},
	}

	out := RenderReviewConversation(testTr(), comments, reviewers, 80, nil)

	assert.Contains(t, out, "@erin")
	assert.Contains(t, out, "APPROVED")
	assert.Contains(t, out, "@frank")
	assert.Contains(t, out, "PENDING")
	assert.Contains(t, out, "@dave")
	assert.Contains(t, out, "ship it")
}

func TestRenderReviewerDetail_FiltersToReviewer(t *testing.T) {
	reviewer := models.Reviewer{Login: "alice", State: "CHANGES_REQUESTED"}
	reviews := []models.Review{
		{Author: "alice", State: "CHANGES_REQUESTED", Body: "needs_work"},
		{Author: "bob", State: "APPROVED", Body: "bob_review"},
	}
	threads := []models.ReviewThread{
		{
			Path: "a.go", Line: 5, IsResolved: false,
			Comments: []models.ReviewComment{
				{Author: "alice", Body: "alice_inline"},
				{Author: "bob", Body: "bob_reply"},
			},
		},
		{
			Path: "b.go", Line: 9, IsResolved: true,
			Comments: []models.ReviewComment{{Author: "bob", Body: "bob_only"}},
		},
	}
	issueComments := []models.IssueComment{
		{Author: "alice", Body: "alice_issue"},
		{Author: "bob", Body: "bob_issue"},
	}

	out := RenderReviewerDetail(testTr(), reviewer, reviews, threads, issueComments, 80, nil)

	// The selected reviewer's own contributions appear.
	assert.Contains(t, out, "@alice")
	assert.Contains(t, out, "CHANGES_REQUESTED")
	assert.Contains(t, out, "needs_work")
	assert.Contains(t, out, "a.go:5")
	assert.Contains(t, out, "alice_inline")
	assert.Contains(t, out, "alice_issue")

	// Other reviewers' contributions are filtered out.
	assert.NotContains(t, out, "bob_review")
	assert.NotContains(t, out, "bob_reply")
	assert.NotContains(t, out, "bob_only")
	assert.NotContains(t, out, "b.go") // alice did not comment on b.go
	assert.NotContains(t, out, "bob_issue")
}

func TestRenderReviewerDetail_NoActivity(t *testing.T) {
	reviewer := models.Reviewer{Login: "carol", State: "PENDING"}
	out := RenderReviewerDetail(testTr(), reviewer, nil, nil, nil, 80, nil)
	assert.Contains(t, out, "@carol")
	assert.Contains(t, out, testTr().PrReviewReviewerNoActivity)
}

func TestRenderReviewerDetail_ResolvedAndOutdatedShowsBoth(t *testing.T) {
	reviewer := models.Reviewer{Login: "alice", State: "COMMENTED"}
	threads := []models.ReviewThread{{
		Path: "a.go", Line: 5, IsResolved: true, IsOutdated: true,
		Comments: []models.ReviewComment{{Author: "alice", Body: "note"}},
	}}
	out := RenderReviewerDetail(testTr(), reviewer, nil, threads, nil, 80, nil)
	assert.Contains(t, out, testTr().PrReviewResolvedBadge)
	assert.Contains(t, out, testTr().PrReviewOutdatedBadge)
}
