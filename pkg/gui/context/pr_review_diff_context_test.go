package context

import (
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/patch"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/stretchr/testify/assert"
)

// patchIdxOf returns the index into patch.Lines() of the first line containing substr.
func patchIdxOf(t *testing.T, patchStr string, substr string) int {
	t.Helper()
	for i, l := range patch.Parse(patchStr).Lines() {
		if strings.Contains(l.Content, substr) {
			return i
		}
	}
	t.Fatalf("no patch line containing %q", substr)
	return -1
}

// TestReviewCommentRange covers the local-diff → GitHub line mapping (tasks 3.2/3.3),
// including the divergence guard: a full `git diff` (with diff --git/---/+++ header,
// as produced locally) and the hunk-only gh-API patch must yield the SAME new-file
// (RIGHT-side) line numbers, matching GitHub's pulls/{n}/files coordinates.
func TestReviewCommentRange(t *testing.T) {
	hunkOnly := "@@ -1,2 +1,3 @@\n" +
		" context_a\n" +
		"+added_b\n" +
		" context_c\n"

	// The same change as a full local `git diff` (header lines must not shift lines).
	fullGitDiff := "diff --git a/file.go b/file.go\n" +
		"index 1111111..2222222 100644\n" +
		"--- a/file.go\n" +
		"+++ b/file.go\n" +
		hunkOnly

	t.Run("single added line maps to its new-file line (hunk-only)", func(t *testing.T) {
		idx := patchIdxOf(t, hunkOnly, "added_b")
		start, end, ok := reviewCommentRange(hunkOnly, idx, idx)
		assert.True(t, ok)
		assert.Equal(t, 2, start)
		assert.Equal(t, 2, end)
	})

	t.Run("full git diff yields the same line numbers as the hunk-only patch", func(t *testing.T) {
		idx := patchIdxOf(t, fullGitDiff, "added_b")
		start, end, ok := reviewCommentRange(fullGitDiff, idx, idx)
		assert.True(t, ok)
		assert.Equal(t, 2, start, "the diff --git header must not shift the new-file line")
		assert.Equal(t, 2, end)
	})

	t.Run("multi-line range spans both added lines", func(t *testing.T) {
		diff := "@@ -1,2 +1,4 @@\n" +
			" context_a\n" +
			"+added_b\n" +
			"+added_c\n" +
			" context_d\n"
		startIdx := patchIdxOf(t, diff, "added_b")
		endIdx := patchIdxOf(t, diff, "added_c")
		start, end, ok := reviewCommentRange(diff, startIdx, endIdx)
		assert.True(t, ok)
		assert.Equal(t, 2, start)
		assert.Equal(t, 3, end)
	})

	t.Run("endpoints normalise to start<=end regardless of selection direction", func(t *testing.T) {
		diff := "@@ -1,2 +1,4 @@\n" +
			" context_a\n" +
			"+added_b\n" +
			"+added_c\n" +
			" context_d\n"
		startIdx := patchIdxOf(t, diff, "added_b")
		endIdx := patchIdxOf(t, diff, "added_c")
		// Selected bottom-to-top: end index passed first.
		start, end, ok := reviewCommentRange(diff, endIdx, startIdx)
		assert.True(t, ok)
		assert.Equal(t, 2, start)
		assert.Equal(t, 3, end)
	})

	t.Run("a deletion-only (LEFT-side) selection is rejected", func(t *testing.T) {
		diff := "@@ -1,3 +1,2 @@\n" +
			" context_a\n" +
			"-removed_b\n" +
			" context_c\n"
		idx := patchIdxOf(t, diff, "removed_b")
		_, _, ok := reviewCommentRange(diff, idx, idx)
		assert.False(t, ok)
	})

	// Rename / --no-renames divergence guard (task 3.3): the local review diff is
	// taken with --no-renames, so a renamed file appears as a brand-new "added" file
	// (and a separate deletion of the old path) rather than GitHub's rename hunk. The
	// added side's new-file line numbers still come from the @@ header, so they match
	// GitHub's pulls/{n}/files RIGHT-side coordinates despite the format divergence.
	t.Run("added-file diff (--no-renames rename) maps to new-file line numbers", func(t *testing.T) {
		diff := "diff --git a/new.go b/new.go\n" +
			"new file mode 100644\n" +
			"index 0000000..1234567\n" +
			"--- /dev/null\n" +
			"+++ b/new.go\n" +
			"@@ -0,0 +1,3 @@\n" +
			"+newline1\n" +
			"+newline2\n" +
			"+newline3\n"
		startIdx := patchIdxOf(t, diff, "newline1")
		midIdx := patchIdxOf(t, diff, "newline2")
		start, end, ok := reviewCommentRange(diff, startIdx, midIdx)
		assert.True(t, ok)
		assert.Equal(t, 1, start)
		assert.Equal(t, 2, end)
	})
}

func TestIsGeneratedReviewFile(t *testing.T) {
	generated := []string{
		"yarn.lock", "package-lock.json", "pnpm-lock.yaml", "go.sum",
		"app/yarn.lock", "vendor/pkg/go.sum", "Cargo.lock",
		"api/service.pb.go", "models/user_generated.go", "queries.gen.go",
		"schema.generated.ts", "components/Button.test.tsx.snap",
		"dist/app.min.js", "dist/app.min.css",
	}
	for _, p := range generated {
		assert.True(t, isGeneratedReviewFile(p), "expected %q to be generated", p)
	}

	source := []string{
		"pkg/gui/gui.go", "src/index.ts", "README.md", "go.mod",
		"components/Button.tsx", "a/b/c.py", "generated_docs.md",
	}
	for _, p := range source {
		assert.False(t, isGeneratedReviewFile(p), "expected %q to be source", p)
	}
}

func TestNextSelectableRowIncludesThreadsOnPlainMove(t *testing.T) {
	// rows: header, diff, comment, comment, diff
	rows := []presentation.ReviewRowKind{
		presentation.ReviewRowHeader,
		presentation.ReviewRowDiff,
		presentation.ReviewRowComment,
		presentation.ReviewRowComment,
		presentation.ReviewRowDiff,
	}

	// A plain move from the diff row lands on the thread block's first row.
	assert.Equal(t, 2, nextSelectableRowIn(rows, 1, 1, true))
	// A range-extending move skips the whole thread block to the next diff row.
	assert.Equal(t, 4, nextSelectableRowIn(rows, 1, 1, false))
	// Moving up from the last diff row lands on the thread block's last row.
	assert.Equal(t, 3, nextSelectableRowIn(rows, 4, -1, true))
	// Headers are never selectable: moving up from the first diff row goes nowhere.
	assert.Equal(t, -1, nextSelectableRowIn(rows, 1, -1, true))
}
