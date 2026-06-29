package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ShowPullRequestsTabWhenGhAvailable = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "With gh available, the branches window has a Pull Requests tab second (Local Branches, Pull Requests, Remotes, Tags) rendering the hardcoded PRs",
	ExtraCmdArgs: []string{},
	ExtraEnvVars: map[string]string{
		// Force the gh gate on so the Pull Requests tab is present regardless of
		// whether the host has the gh CLI installed.
		GH_AVAILABLE_OVERRIDE_ENV_VAR: "true",
	},
	Skip:        false,
	SetupConfig: func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file1", "content\n")
		shell.Commit("initial commit")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Focus the branches window; the default (first) tab is Local Branches.
		t.Views().Branches().
			Focus().
			IsFocused()

		// One tab to the right is the Pull Requests tab, which renders the
		// hardcoded sample rows. This proves it is positioned second.
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().PullRequests().
			IsFocused().
			Content(Contains("[SAMPLE]"))

		// The next tab is Remotes, confirming the Pull Requests tab sits between
		// Local Branches and Remotes.
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().Remotes().
			IsFocused()
	},
})
