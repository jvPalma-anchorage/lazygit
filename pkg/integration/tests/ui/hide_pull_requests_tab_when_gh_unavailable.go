package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var HidePullRequestsTabWhenGhUnavailable = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "With gh unavailable, the branches window shows only Local Branches, Remotes and Tags; tab cycling never lands on the pull-requests context",
	ExtraCmdArgs: []string{},
	ExtraEnvVars: map[string]string{
		// Force the gh gate off so the Pull Requests tab is absent.
		GH_AVAILABLE_OVERRIDE_ENV_VAR: "false",
	},
	Skip:        false,
	SetupConfig: func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file1", "content\n")
		shell.Commit("initial commit")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Focusing the branches window lands on Local Branches (the default tab).
		t.Views().Branches().
			Focus().
			IsFocused()

		// With the Pull Requests tab absent, one tab to the right is Remotes
		// (not pull-requests), proving the tab is not present and cycling skips
		// the pull-requests context.
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().Remotes().
			IsFocused()

		// One more tab is Tags; cycling wraps back to Local Branches — at no
		// point is the pull-requests context focused.
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().Tags().
			IsFocused()

		t.GlobalPress(keys.Universal.NextTab)
		t.Views().Branches().
			IsFocused()
	},
})
