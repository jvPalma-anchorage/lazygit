package git_commands

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func TestReviewRefNames(t *testing.T) {
	assert.Equal(t, "refs/lazygit-review/42/head", ReviewHeadRef(42))
	assert.Equal(t, "refs/lazygit-review/42/base", ReviewBaseRef(42))
}

func TestReviewFetchCmdObjArgs(t *testing.T) {
	gh := NewGitHubCommands(buildGitCommon(commonDeps{}))

	// Fetch writes ONLY into the isolated namespace by explicit URL+OID, and never
	// touches FETCH_HEAD, tags, the working tree, the index, or any branch.
	assert.Equal(t,
		[]string{
			"git", "fetch", "--no-write-fetch-head", "--no-tags",
			"https://github.com/owner/repo.git",
			"+abc123:refs/lazygit-review/7/head",
		},
		gh.FetchReviewRefCmdObj("https://github.com/owner/repo.git", "abc123", ReviewHeadRef(7)).Args(),
	)

	assert.Equal(t,
		[]string{"git", "merge-base", "refs/lazygit-review/7/base", "refs/lazygit-review/7/head"},
		gh.ReviewMergeBaseCmdObj(7).Args(),
	)

	assert.Equal(t,
		[]string{"git", "diff", "--name-status", "--no-renames", "mergebaseoid", "refs/lazygit-review/7/head"},
		gh.ReviewChangedFilesCmdObj("mergebaseoid", ReviewHeadRef(7)).Args(),
	)

	assert.Equal(t,
		[]string{"git", "for-each-ref", "--format=%(refname) %(committerdate:unix)", "refs/lazygit-review/"},
		gh.PruneStaleReviewRefsListCmdObj().Args(),
	)

	assert.Equal(t,
		[]string{"git", "update-ref", "-d", "refs/lazygit-review/9/base"},
		gh.DeleteReviewRefCmdObj(ReviewBaseRef(9)).Args(),
	)
}

func TestReviewFileBlobOid(t *testing.T) {
	t.Run("returns the trimmed blob sha", func(t *testing.T) {
		runner := oscommands.NewFakeRunner(t).
			ExpectGitArgs([]string{"rev-parse", "refs/lazygit-review/7/head:pkg/a.go"}, "deadbeef\n", nil)
		gh := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))
		assert.Equal(t, "deadbeef", gh.ReviewFileBlobOid(ReviewHeadRef(7), "pkg/a.go"))
		runner.CheckForMissingCalls()
	})

	t.Run("returns empty when the path does not exist at the ref", func(t *testing.T) {
		runner := oscommands.NewFakeRunner(t).
			ExpectGitArgs([]string{"rev-parse", "refs/lazygit-review/7/head:gone.go"}, "", assert.AnError)
		gh := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))
		assert.Equal(t, "", gh.ReviewFileBlobOid(ReviewHeadRef(7), "gone.go"))
		runner.CheckForMissingCalls()
	})
}

func TestReviewMergeBase(t *testing.T) {
	t.Run("returns the trimmed merge-base", func(t *testing.T) {
		runner := oscommands.NewFakeRunner(t).
			ExpectGitArgs([]string{"merge-base", "refs/lazygit-review/3/base", "refs/lazygit-review/3/head"}, "deadbeef\n", nil)
		gh := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

		mb, err := gh.ReviewMergeBase(3)
		assert.NoError(t, err)
		assert.Equal(t, "deadbeef", mb)
		runner.CheckForMissingCalls()
	})

	t.Run("errors when refs share no history", func(t *testing.T) {
		runner := oscommands.NewFakeRunner(t).
			ExpectGitArgs([]string{"merge-base", "refs/lazygit-review/3/base", "refs/lazygit-review/3/head"}, "", assert.AnError)
		gh := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

		_, err := gh.ReviewMergeBase(3)
		assert.Error(t, err)
		runner.CheckForMissingCalls()
	})
}

func TestReviewChangedFiles(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs(
			[]string{"diff", "--name-status", "--no-renames", "mb", "refs/lazygit-review/1/head"},
			"M\tpkg/a.go\nA\tpkg/new.go\nD\tpkg/gone.go\n",
			nil,
		)
	gh := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

	files, err := gh.ReviewChangedFiles("mb", ReviewHeadRef(1))
	assert.NoError(t, err)
	assert.Equal(t, []ReviewChangedFile{
		{Status: "M", Path: "pkg/a.go"},
		{Status: "A", Path: "pkg/new.go"},
		{Status: "D", Path: "pkg/gone.go"},
	}, files)
	runner.CheckForMissingCalls()
}

func TestParseReviewChangedFiles(t *testing.T) {
	scenarios := []struct {
		name     string
		out      string
		expected []ReviewChangedFile
	}{
		{
			name:     "empty output",
			out:      "",
			expected: nil,
		},
		{
			name: "modified, added, deleted",
			out:  "M\ta.txt\nA\tb.txt\nD\tc.txt\n",
			expected: []ReviewChangedFile{
				{Status: "M", Path: "a.txt"},
				{Status: "A", Path: "b.txt"},
				{Status: "D", Path: "c.txt"},
			},
		},
		{
			name: "rename carries old + new path",
			out:  "R100\told/name.go\tnew/name.go\n",
			expected: []ReviewChangedFile{
				{Status: "R", OldPath: "old/name.go", Path: "new/name.go"},
			},
		},
		{
			name: "blank lines and CR are ignored",
			out:  "M\ta.txt\r\n\nA\tb.txt\n",
			expected: []ReviewChangedFile{
				{Status: "M", Path: "a.txt"},
				{Status: "A", Path: "b.txt"},
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, parseReviewChangedFiles(s.out))
		})
	}
}

func TestStaleReviewRefs(t *testing.T) {
	// cutoff = 1000: refs at or below 1000 are stale.
	out := "refs/lazygit-review/1/head 500\n" +
		"refs/lazygit-review/1/base 1000\n" +
		"refs/lazygit-review/2/head 2000\n" +
		"refs/lazygit-review/3/head notanumber\n"

	assert.Equal(t,
		[]string{"refs/lazygit-review/1/head", "refs/lazygit-review/1/base"},
		staleReviewRefs(out, 1000),
	)
}

func TestPruneStaleReviewRefs(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs(
			[]string{"for-each-ref", "--format=%(refname) %(committerdate:unix)", "refs/lazygit-review/"},
			"refs/lazygit-review/1/head 100\nrefs/lazygit-review/9/head 9999999999\n",
			nil,
		).
		// now=1_000_000, maxAge=30d → cutoff = 1_000_000 - 2_592_000 < 0; only ts<=cutoff
		// pruned. With now far in the future the old ref (100) is stale, the fresh one is not.
		ExpectGitArgs([]string{"update-ref", "-d", "refs/lazygit-review/1/head"}, "", nil)
	gh := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

	// now=10_000_000_000 makes ts=100 older than 30 days but ts=9999999999 still fresh.
	err := gh.PruneStaleReviewRefs(10_000_000_000, 30)
	assert.NoError(t, err)
	runner.CheckForMissingCalls()
}
