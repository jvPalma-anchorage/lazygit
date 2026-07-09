package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewGroupedReview drives Phase 7's accumulate-then-submit flow up to the
// network boundary: `C` in the diff queues a comment locally (no gh call — proven by
// the toast arriving in fixture mode, where any gh invocation would fail), the queue
// counts up across files, and `S` in the tree opens the submit menu with the three
// review events. The POST itself is unit-tested against the fake runner.
var PrReviewGroupedReview = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: comments queue locally for a grouped review and S offers the submit events",
	ExtraCmdArgs: []string{"anchorlabsinc/anchorage", "777"},
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
				ID: "PR_777", Number: 777, Title: "Grouped review PR", State: "OPEN", HeadRefOid: "head",
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
		t.Views().PrReviewDiff().IsFocused()

		// Queue two comments; the toast counts up and nothing hits the network.
		t.GlobalPress(config.Keybinding{"C"})
		t.ExpectPopup().Prompt().
			Title(Equals("Comment (queued for grouped review)")).
			Type("first pending comment").
			Confirm()
		t.ExpectToast(Equals("Comment queued for review (1 pending)"))

		t.Views().PrReviewDiff().SelectNextItem()
		t.GlobalPress(config.Keybinding{"C"})
		t.ExpectPopup().Prompt().
			Title(Equals("Comment (queued for grouped review)")).
			Type("second pending comment").
			Confirm()
		t.ExpectToast(Equals("Comment queued for review (2 pending)"))

		// Back in the tree, S opens the submit menu carrying the pending count and
		// the three review events.
		t.Views().PrReviewDiff().PressEscape()
		t.Views().PrReview().IsFocused()
		t.GlobalPress(config.Keybinding{"S"})
		t.ExpectPopup().Menu().
			Title(Equals("Submit review (2 pending comments)")).
			ContainsLines(
				Contains("Comment"),
				Contains("Approve"),
				Contains("Request changes"),
			).
			Cancel()
	},
})
