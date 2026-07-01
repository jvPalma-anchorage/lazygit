package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewCommitsTab drives the Phase 5 PR-commits tab (panel [3] · Commits). It sets
// up a base ref and two PR commits on the head ref, then proves the Commits tab lists
// exactly the base..head commits (not the whole branch) and that hovering a commit
// renders that commit's changes (message + --stat file list + diff) in the main view.
var PrReviewCommitsTab = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: the Commits tab lists exactly the base..head commits and shows each commit's changes",
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

		// Base of the PR.
		shell.CreateFileAndAdd("base.txt", "base\n")
		shell.Commit("pr base commit")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/base", "HEAD"})

		// Two commits that make up the PR (base..head).
		shell.CreateFileAndAdd("fileA.txt", "AAA\n")
		shell.Commit("PR_COMMIT_ONE")
		shell.CreateFileAndAdd("fileB.txt", "BBB\n")
		shell.Commit("PR_COMMIT_TWO")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/head", "HEAD"})

		fixture := struct {
			Data  *git_commands.PullRequestReviewData
			Files []any
		}{
			Data: &git_commands.PullRequestReviewData{
				ID: "PR_7", Number: 7, Title: "Commits-tab PR", State: "OPEN", HeadRefOid: "head7",
			},
		}
		raw, err := json.Marshal(fixture)
		if err != nil {
			panic(err)
		}
		shell.CreateFile("pr_fixture.json", string(raw))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Boot focuses Files Changed; jump to PR Activity (panel [3], default tab
		// Conversation), then cycle to the Commits tab.
		t.GlobalPress(keys.Universal.JumpToBlock[2])
		t.GlobalPress(keys.Universal.NextTab) // Conversation -> Checks
		t.GlobalPress(keys.Universal.NextTab) // Checks -> Commits

		// The list shows exactly the two base..head commits — not the seed/base/init
		// commits (those are reachable from base, so excluded by base..head).
		t.Views().PrCommits().
			IsFocused().
			ContainsLines(Contains("PR_COMMIT_ONE")).
			ContainsLines(Contains("PR_COMMIT_TWO")).
			// base..head excludes every ancestor reachable from base: the base commit
			// AND the earlier seed/init commits.
			Content(DoesNotContainAnyOf("pr base commit", "seed", "init"))

		// Hovering a commit renders that commit's changes (its added file) in main.
		t.Views().PrCommits().NavigateToLine(Contains("PR_COMMIT_TWO"))
		t.Views().Main().Content(Contains("fileB"))
		t.Views().PrCommits().NavigateToLine(Contains("PR_COMMIT_ONE"))
		t.Views().Main().Content(Contains("fileA"))

		// Enter focuses the commit's changes in the main view; Escape returns to the
		// commit list (staying inside the review workspace).
		t.Views().PrCommits().PressEnter()
		t.Views().Main().IsFocused()
		t.Views().Main().PressEscape()
		t.Views().PrCommits().IsFocused()
	},
})
