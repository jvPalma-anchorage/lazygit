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
				Labels: []models.PrLabel{
					{Name: "CHIP_BUG", Color: "d73a4a"},
				},
				// Timeline sources: a bot comment (older) and a human review
				// summary (newer) — the overview must show both, oldest first.
				IssueComments: []models.IssueComment{
					{Author: "release-bot[bot]", Body: "BOT_TIMELINE_ENTRY", CreatedAt: "2026-01-01T00:00:00Z"},
				},
				Reviews: []models.Review{
					{Author: "carol", State: "APPROVED", Body: "REVIEW_TIMELINE_ENTRY", SubmittedAt: "2026-02-01T00:00:00Z"},
				},
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
		// The Overview now lives in window [1]'s hover preview (Phase 13): jump to the
		// PR list and the launched (current) PR is selected, so its overview renders in
		// the main view.
		t.Views().PrReview().IsFocused()
		t.GlobalPress(keys.Universal.JumpToBlock[0])
		t.Views().PrList().IsFocused()

		// The main view shows the v2 layout: heading, state+author, branches, label
		// chip, description, and the oldest-first timeline including the bot entry.
		t.Views().Main().
			Content(Contains("#42 - OVERVIEW_TITLE")).
			Content(Contains("OPEN by @overviewauthor")).
			Content(Contains("main ← feature-x")).
			Content(Contains("CHIP_BUG")).
			Content(Contains("OVERVIEW_BODY")).
			Content(Contains("@release-bot[bot]")).
			Content(Contains("BOT_TIMELINE_ENTRY")).
			Content(Contains("REVIEW_TIMELINE_ENTRY"))
	},
})
