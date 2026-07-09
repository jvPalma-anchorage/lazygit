package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewOthersComments proves the two review-visibility fixes: a file with an
// unresolved review thread carries a 💬 marker in the tree, and simply hovering that
// file (no Enter) renders ANOTHER reviewer's inline comment in the main view — the
// pager preview is swapped for the thread-interleaving presenter whenever the file has
// comments, so other people's review comments are visible without any interaction.
var PrReviewOthersComments = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: a file with comments shows a 💬 marker and its inline threads (any author) render on hover",
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

		shell.CreateFileAndAdd("foo.txt", "alpha\nbeta\n")
		shell.CreateFileAndAdd("bar.txt", "quiet\n")
		shell.Commit("pr base")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/base", "HEAD"})

		// Head adds a line to foo.txt (line 3) — the anchor for the review thread — and
		// leaves bar.txt with its own change but no comments.
		shell.CreateFileAndAdd("foo.txt", "alpha\nbeta\nCOMMENTED_LINE\n")
		shell.CreateFileAndAdd("bar.txt", "quiet\nchange\n")
		shell.Commit("pr head")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/head", "HEAD"})

		fixture := struct {
			Data  *git_commands.PullRequestReviewData
			Files []any
		}{
			Data: &git_commands.PullRequestReviewData{
				ID: "PR_7", Number: 7, Title: "Comments PR", State: "OPEN", HeadRefOid: "head7",
				Threads: []models.ReviewThread{
					{
						ID: "T_OTHER", Path: "foo.txt", Line: 3, Side: "RIGHT", IsResolved: false,
						Comments: []models.ReviewComment{
							{DatabaseID: 11, Author: "someone-else", Body: "OTHER_PERSON_COMMENT"},
						},
					},
				},
			},
		}
		raw, err := json.Marshal(fixture)
		if err != nil {
			panic(err)
		}
		shell.CreateFile("pr_fixture.json", string(raw))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// The file with an unresolved thread carries the 💬 marker; the quiet file
		// does not.
		t.Views().PrReview().
			IsFocused().
			NavigateToLine(Contains("foo.txt")).
			SelectedLine(Contains("💬").Contains("foo.txt"))
		t.Views().PrReview().
			NavigateToLine(Contains("bar.txt")).
			SelectedLine(Contains("bar.txt").DoesNotContain("💬"))

		// Merely hovering foo.txt (no Enter) shows the OTHER reviewer's comment inline
		// in the main view — previously the pager preview showed no comments at all.
		t.Views().PrReview().NavigateToLine(Contains("foo.txt"))
		t.Views().Main().
			Content(Contains("COMMENTED_LINE")).
			Content(Contains("someone-else")).
			Content(Contains("OTHER_PERSON_COMMENT"))

		// A file without comments keeps the plain pager diff (no thread chrome).
		t.Views().PrReview().NavigateToLine(Contains("bar.txt"))
		t.Views().Main().
			Content(Contains("change")).
			Content(DoesNotContain("OTHER_PERSON_COMMENT"))
	},
})
