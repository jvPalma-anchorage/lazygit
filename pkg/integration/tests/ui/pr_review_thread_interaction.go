package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewThreadInteraction drives Phase 11: inside the diff view, an inline review
// thread is a selectable anchor — the cursor lands on it, `c` there opens the REPLY
// prompt (not the new-comment prompt), and `t` on a plain diff row explains that a
// thread must be selected. The writes themselves shell out to gh, so this test stops
// at the prompt (write construction is unit-tested against the fake runner).
var PrReviewThreadInteraction = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: inline threads are selectable anchors with a reply prompt and a resolve guard",
	ExtraCmdArgs: []string{"anchorlabsinc/anchorage", "31337"},
	ExtraEnvVars: map[string]string{"LAZYGIT_PR_REVIEW_FIXTURE": "pr_fixture.json"},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("seed", "seed\n")
		shell.Commit("init")

		fixture := struct {
			Data  *git_commands.PullRequestReviewData
			Files []*models.GithubPullRequestFile
		}{
			Data: &git_commands.PullRequestReviewData{
				ID: "PR_1", Number: 31337, Title: "Thread PR", State: "OPEN", HeadRefOid: "head",
				Threads: []models.ReviewThread{
					{
						ID: "T_1", Path: "foo.txt", Line: 2, Side: "RIGHT",
						Comments: []models.ReviewComment{
							{DatabaseID: 99, Author: "alice", Body: "THREAD_BODY"},
						},
					},
				},
			},
			Files: []*models.GithubPullRequestFile{
				{
					Filename: "foo.txt",
					Status:   "modified",
					Patch:    "@@ -1,2 +1,3 @@\n line one\n+inserted line\n line two\n",
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
		t.Views().PrReview().
			IsFocused().
			Lines(Contains("foo.txt"))
		t.Views().PrReview().PressEnter()

		// The diff renders with the thread interleaved under its anchor line.
		t.Views().PrReviewDiff().
			IsFocused().
			Content(Contains("inserted line")).
			Content(Contains("@alice")).
			Content(Contains("THREAD_BODY"))

		// On a plain diff row, `t` explains a thread must be selected first.
		t.GlobalPress(config.Keybinding{"t"})
		t.ExpectToast(Contains("Move the cursor onto a comment thread first"))

		// Arrow down walks from "line one" over "inserted line" ONTO the thread block
		// anchored under it (Phase 11: plain moves land on threads), where `c` opens
		// the REPLY prompt, not the new-comment prompt.
		t.Views().PrReviewDiff().SelectNextItem().SelectNextItem()
		t.GlobalPress(config.Keybinding{"c"})
		t.ExpectPopup().Prompt().
			Title(Equals("Reply to thread")).
			Cancel()

		// Back on a diff row (up), `c` opens the plain new-comment prompt again.
		t.Views().PrReviewDiff().SelectPreviousItem()
		t.GlobalPress(config.Keybinding{"c"})
		t.ExpectPopup().Prompt().
			Title(Equals("Review comment (RIGHT side: added/context lines)")).
			Cancel()
	},
})
