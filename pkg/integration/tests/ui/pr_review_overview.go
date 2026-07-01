package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewOverview drives the Overview tab of the PR Content window: focusing it
// renders the pull request's headline details (title, number, state, author, base/head
// branches) and its description body into the main view.
var PrReviewOverview = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: the Overview tab renders the PR title, branches, author and description",
	ExtraCmdArgs: []string{"anchorlabsinc/anchorage", "42"},
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
				ID: "PR_42", Number: 42, Title: "OVERVIEW_TITLE", State: "OPEN",
				Author: "overviewauthor", BaseRefName: "main", HeadRefName: "feature-x",
				Body: "OVERVIEW_BODY text", HeadRefOid: "head",
			},
			Files: []*models.GithubPullRequestFile{
				{Filename: "a.go", Status: "modified", Patch: "@@ -1,1 +1,1 @@\n-old\n+new\n"},
			},
		}
		raw, err := json.Marshal(fixture)
		if err != nil {
			panic(err)
		}
		shell.CreateFile("pr_fixture.json", string(raw))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Boot focuses Files Changed; `[` (previous tab) switches to the Overview tab.
		t.Views().PrReview().IsFocused()
		t.GlobalPress(keys.Universal.PrevTab)
		t.Views().PrOverview().IsFocused()

		// The main view shows the PR's headline details and description.
		t.Views().Main().
			Content(Contains("OVERVIEW_TITLE")).
			Content(Contains("#42")).
			Content(Contains("main")).
			Content(Contains("feature-x")).
			Content(Contains("overviewauthor")).
			Content(Contains("OVERVIEW_BODY"))
	},
})
