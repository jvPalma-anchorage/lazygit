package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewPrList drives window [1] (tasks 8.0 + 8.1): the launched PR heads the list
// under "Current" (with no gh-dash dependency), the gh-dash-style sections render as
// data underneath (via the LAZYGIT_PR_LIST_FIXTURE seam — no network), section
// headers are not openable, and Enter on the already-loaded PR jumps focus to the PR
// Content window.
var PrReviewPrList = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: window [1] lists the launched PR plus gh-dash sections and opens PRs into the workspace",
	ExtraCmdArgs: []string{"anchorlabsinc/anchorage", "555"},
	ExtraEnvVars: map[string]string{
		"LAZYGIT_PR_REVIEW_FIXTURE": "pr_fixture.json",
		"LAZYGIT_PR_LIST_FIXTURE":   "pr_list_fixture.json",
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
				ID: "PR_555", Number: 555, Title: "LAUNCHED_PR_TITLE", State: "OPEN", HeadRefOid: "head",
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

		sections := []struct {
			Title string
			Prs   []git_commands.PrListEntry
		}{
			{
				Title: "Needs My Review",
				Prs: []git_commands.PrListEntry{
					{Owner: "anchorlabsinc", Repo: "anchorage", Number: 601, Title: "SECTION_PR_ONE", State: "OPEN", Author: "alice"},
					{Owner: "anchorlabsinc", Repo: "anchorage", Number: 602, Title: "SECTION_PR_TWO", State: "OPEN", Author: "bob"},
				},
			},
		}
		rawSections, err := json.Marshal(sections)
		if err != nil {
			panic(err)
		}
		shell.CreateFile("pr_list_fixture.json", string(rawSections))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Jump to window [1]. Sections are TABS now (Phase 13): the active tab is
		// "Current" (the launched PR), and its title bar carries the section tabs.
		t.GlobalPress(keys.Universal.JumpToBlock[0])
		t.Views().PrList().
			IsFocused().
			// Sections are TABS (Phase 13): the list shows ONLY the active ("Current")
			// section — the launched PR, not the other section's PRs.
			Content(Contains("#555").Contains("LAUNCHED_PR_TITLE")).
			Content(DoesNotContain("SECTION_PR_ONE"))

		// `]` switches to the "Needs My Review" tab, whose PRs now show (and Current's
		// launched PR does not).
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().PrList().
			Content(Contains("#601").Contains("SECTION_PR_ONE")).
			Content(Contains("#602").Contains("SECTION_PR_TWO")).
			Content(DoesNotContain("LAUNCHED_PR_TITLE"))

		// `[` returns to Current.
		t.GlobalPress(keys.Universal.PrevTab)
		t.Views().PrList().Content(Contains("LAUNCHED_PR_TITLE"))

		// enter on a PR focuses the main panel so its overview can be scrolled; Esc
		// returns to the list.
		t.Views().PrList().NavigateToLine(Contains("#555"))
		t.Views().PrList().PressEnter()
		t.Views().Main().IsFocused()
		t.Views().Main().PressEscape()
		t.Views().PrList().IsFocused()

		// SPACE on the launched (current) PR starts its review — focus moves to the
		// content window. (It is already loaded, so this is just a jump.)
		t.Views().PrList().Press(keys.Universal.Select)
		t.Views().PrReview().IsFocused()
	},
})
