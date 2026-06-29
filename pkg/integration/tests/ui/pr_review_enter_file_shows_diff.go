package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewEnterFileShowsDiff drives PR review mode headlessly with injected fixture
// data (LAZYGIT_PR_REVIEW_FIXTURE, so no live GitHub) to prove that pressing Enter
// on a changed file renders that file's diff in the focusable diff view rather than
// leaving it blank.
var PrReviewEnterFileShowsDiff = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: Enter on a changed file shows its diff in the diff view",
	ExtraCmdArgs: []string{"anchorlabsinc/anchorage", "215339"},
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
				ID:         "PR_1",
				Number:     215339,
				Title:      "Fixture PR",
				Body:       "Body",
				State:      "OPEN",
				HeadRefOid: "deadbeef",
			},
			Files: []*models.GithubPullRequestFile{
				{
					Filename:  "foo.txt",
					Status:    "modified",
					Patch:     "@@ -1,2 +1,3 @@\n line one\n+inserted line\n line two\n",
					Additions: 1,
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
		// The file tree lists the changed file.
		t.Views().PrReview().
			IsFocused().
			Lines(Contains("foo.txt"))

		// Enter opens the file's diff in the diff view (the bug under test: blank).
		t.Views().PrReview().PressEnter()

		t.Views().PrReviewDiff().
			Content(Contains("inserted line"))
	},
})
