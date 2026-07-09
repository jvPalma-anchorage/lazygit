package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewChecksTab drives the Checks tab (task 8.2) via the LAZYGIT_PR_CHECKS_FIXTURE
// seam: checks group under their workflow parents, failures sort above passes above
// skips (importance order), and Enter on a parent collapses its children.
var PrReviewChecksTab = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: the Checks tab groups by workflow, sorts by importance, and collapses",
	ExtraCmdArgs: []string{"anchorlabsinc/anchorage", "888"},
	ExtraEnvVars: map[string]string{
		"LAZYGIT_PR_REVIEW_FIXTURE": "pr_fixture.json",
		"LAZYGIT_PR_CHECKS_FIXTURE": "pr_checks_fixture.json",
	},
	Skip:        false,
	SetupConfig: func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("seed", "seed\n")
		shell.Commit("init")

		fixture := struct {
			Data  *git_commands.PullRequestReviewData
			Files []*models.GithubPullRequestFile
		}{
			Data: &git_commands.PullRequestReviewData{
				ID: "PR_888", Number: 888, Title: "Checks PR", State: "OPEN", HeadRefOid: "head",
			},
			Files: []*models.GithubPullRequestFile{
				{Filename: "foo.txt", Status: "modified", Patch: "@@ -1,1 +1,2 @@\n line\n+more\n"},
			},
		}
		raw, err := json.Marshal(fixture)
		if err != nil {
			panic(err)
		}
		shell.CreateFile("pr_fixture.json", string(raw))

		// Deliberately shuffled: the pass check first, the failure last — the tab
		// must importance-sort them (failure first, skipped last). One workflow name
		// is intentionally long (real anchorage workflows run ~48 chars): the leaf
		// name must still render, not get clipped off the panel by column padding.
		checks := []git_commands.PrCheck{
			{Workflow: "A deliberately very long workflow parent name here", Name: "UNIT_TESTS_PASS", Bucket: "pass", State: "SUCCESS", CompletedAt: "2026-01-01T00:00:00Z"},
			{Workflow: "Lint", Name: "SKIPPED_CHECK", Bucket: "skipping", State: "SKIPPED"},
			{Workflow: "A deliberately very long workflow parent name here", Name: "BUILD_FAILURE", Bucket: "fail", State: "FAILURE", CompletedAt: "2026-01-02T00:00:00Z"},
		}
		rawChecks, err := json.Marshal(checks)
		if err != nil {
			panic(err)
		}
		shell.CreateFile("pr_checks_fixture.json", string(rawChecks))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Panel [3] then `]` to the Checks tab (Conversation is the default).
		t.GlobalPress(keys.Universal.JumpToBlock[2])
		t.Views().PrConversation().IsFocused()
		t.GlobalPress(keys.Universal.NextTab)

		// Importance order: CI (owning the failure) heads the list with the failure
		// above the pass; the skipped check's workflow trails.
		t.Views().PrChecks().
			IsFocused().
			ContainsLines(
				Contains("A deliberately very long workflow parent name here"),
				Contains("BUILD_FAILURE"),
				Contains("UNIT_TESTS_PASS"),
				Contains("Lint"),
				Contains("SKIPPED_CHECK"),
			)

		// Enter on the CI parent collapses its two checks.
		t.Views().PrChecks().NavigateToLine(Equals("▼ A deliberately very long workflow parent name here"))
		t.Views().PrChecks().PressEnter()
		t.Views().PrChecks().
			Content(DoesNotContain("BUILD_FAILURE")).
			Content(DoesNotContain("UNIT_TESTS_PASS")).
			Content(Contains("SKIPPED_CHECK"))

		// Enter again expands.
		t.Views().PrChecks().PressEnter()
		t.Views().PrChecks().Content(Contains("BUILD_FAILURE"))
	},
})
