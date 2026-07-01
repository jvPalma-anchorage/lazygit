package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewConversation drives the Phase 6 Conversation tab (panel [3] · Conversation).
// It proves the tab lists the PR's reviewers with their states (6.1) and that selecting
// a reviewer renders that reviewer's own detail — review summary, inline thread comment,
// and PR-body comment — filtered to them, in the main view (6.2).
var PrReviewConversation = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: the Conversation tab lists reviewers and shows the selected reviewer's comments",
	ExtraCmdArgs: []string{"anchorlabsinc/anchorage", "9000"},
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
				ID: "PR_9000", Number: 9000, Title: "Conversation PR", State: "OPEN", HeadRefOid: "head",
				Reviewers: []models.Reviewer{
					{Login: "alice", State: "APPROVED"},
					{Login: "bob", State: "CHANGES_REQUESTED"},
				},
				Reviews: []models.Review{
					{Author: "alice", State: "APPROVED", Body: "ALICE_REVIEW_BODY"},
					{Author: "bob", State: "CHANGES_REQUESTED", Body: "BOB_REVIEW_BODY"},
				},
				Threads: []models.ReviewThread{
					{
						Path: "a.go", Line: 5, IsResolved: false,
						Comments: []models.ReviewComment{{Author: "alice", Body: "ALICE_INLINE"}},
					},
				},
				IssueComments: []models.IssueComment{
					{Author: "bob", Body: "BOB_ISSUE"},
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
		// Jump to PR Activity (panel [3]); its default tab is Conversation.
		t.GlobalPress(keys.Universal.JumpToBlock[2])

		// 6.1: the reviewer list shows each reviewer with their state.
		t.Views().PrConversation().
			IsFocused().
			ContainsLines(Contains("alice")).
			ContainsLines(Contains("APPROVED")).
			ContainsLines(Contains("bob")).
			ContainsLines(Contains("CHANGES_REQUESTED"))

		// 6.2: selecting alice shows only alice's contributions.
		t.Views().PrConversation().NavigateToLine(Contains("alice"))
		t.Views().Main().
			Content(Contains("ALICE_REVIEW_BODY")).
			Content(Contains("ALICE_INLINE")).
			Content(DoesNotContain("BOB_REVIEW_BODY"))

		// Selecting bob switches to bob's contributions (review body + PR-body comment).
		t.Views().PrConversation().NavigateToLine(Contains("bob"))
		t.Views().Main().
			Content(Contains("BOB_REVIEW_BODY")).
			Content(Contains("BOB_ISSUE")).
			Content(DoesNotContain("ALICE_REVIEW_BODY"))
	},
})
