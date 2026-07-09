package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewLocalRefDiff drives the Phase 2 local-ref read model headlessly. The test
// repo plays the role of a fetched checkout: it creates the PR's base and head as
// real local commits under refs/lazygit-review/<pr>/{base,head} (exactly what the
// network fetch would write), then boots review mode with LAZYGIT_PR_REVIEW_LOCAL_REFS
// set so the changed-file tree and diffs are built from `git diff base..head` instead
// of the gh API. It proves: the tree lists the changed files (2.5); selecting a file
// renders that file's diff (2.6); and selecting a folder renders the AGGREGATE diff of
// every file under it rather than "no changed files" (bug #1).
var PrReviewLocalRefDiff = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review (local-ref model): the file tree and per-file / per-folder diffs come from git diff base..head",
	ExtraCmdArgs: []string{"owner/repo", "7"},
	ExtraEnvVars: map[string]string{
		"LAZYGIT_PR_REVIEW_FIXTURE":    "pr_fixture.json",
		"LAZYGIT_PR_REVIEW_LOCAL_REFS": "7",
	},
	Skip:        false,
	SetupConfig: func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("seed", "seed\n")
		shell.Commit("init")

		// Base of the PR: one file under a folder.
		shell.CreateFileAndAdd("src/foo.txt", "alpha\nbeta\n")
		shell.Commit("pr base")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/base", "HEAD"})

		// Head of the PR: modify foo.txt and add a new file under the same folder.
		shell.CreateFileAndAdd("src/foo.txt", "alpha\nbeta\nFOO_ADDED\n")
		shell.CreateFileAndAdd("src/bar.txt", "BAR_NEW\n")
		shell.Commit("pr head")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/head", "HEAD"})

		// Minimal PR metadata so the panel has data (the tree/diff come from refs).
		fixture := struct {
			Data  *git_commands.PullRequestReviewData
			Files []any
		}{
			Data: &git_commands.PullRequestReviewData{
				ID: "PR_7", Number: 7, Title: "Local-ref PR", State: "OPEN", HeadRefOid: "head7",
			},
		}
		raw, err := json.Marshal(fixture)
		if err != nil {
			panic(err)
		}
		shell.CreateFile("pr_fixture.json", string(raw))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// 2.5: the changed-file tree lists both files from `git diff --name-status`.
		t.Views().PrReview().
			IsFocused().
			ContainsLines(Contains("foo.txt")).
			ContainsLines(Contains("bar.txt"))

		// Bug #1: selecting the folder renders the AGGREGATE diff of every file under
		// it (both the modified foo.txt and the added bar.txt), not "no changed files".
		t.Views().PrReview().NavigateToLine(Contains("src"))
		t.Views().Main().
			Content(Contains("FOO_ADDED")).
			Content(Contains("BAR_NEW"))

		// 2.6: selecting a single file renders that file's diff.
		t.Views().PrReview().NavigateToLine(Contains("foo.txt"))
		t.Views().Main().Content(Contains("FOO_ADDED"))

		// Phase 3 (3.5): `space` toggles the "viewed" state on the file, which colors
		// its name green (unviewed → not green → viewed → green).
		t.Views().PrReview().DoesNotContainColoredText("#008000", "foo.txt")
		t.Views().PrReview().Press(keys.Universal.Select)
		t.Views().PrReview().ContainsColoredText("#008000", "foo.txt")

		// Phase 3 (3.1/3.5): Enter on a file opens the focusable diff surface, sourced
		// from the local `git diff` (no-op before — the bug found in manual testing).
		// The selection cursor lands on a diff row, and `space` extends the range.
		t.Views().PrReview().PressEnter()
		t.Views().PrReviewDiff().
			IsFocused().
			Content(Contains("FOO_ADDED"))
		t.Views().PrReviewDiff().Press(keys.Universal.Select) // extend the line range
		// Esc returns to the file tree without leaving the review workspace.
		t.Views().PrReviewDiff().PressEscape()
		t.Views().PrReview().IsFocused()
	},
})
