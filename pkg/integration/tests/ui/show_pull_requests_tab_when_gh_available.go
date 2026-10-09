package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ShowPullRequestsTabWhenGhAvailable = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "With gh available, the branches window has a Pull Requests tab second (Local Branches, Pull Requests, Remotes, Tags) listing the repo's open PRs from gh and previewing the selected one's overview",
	ExtraCmdArgs: []string{},
	ExtraEnvVars: map[string]string{
		// Force the gh gate on so the Pull Requests tab is present regardless of
		// whether the host has the gh CLI installed.
		GH_AVAILABLE_OVERRIDE_ENV_VAR: "true",
		// Stand in for gh with a script that answers `gh pr list`.
		"GH_PATH": ".git/fake-gh",
	},
	Skip:        false,
	SetupConfig: func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file1", "content\n")
		shell.Commit("initial commit")
		shell.CreateFile(".git/fake-gh", `#!/bin/sh
case "$1 $2" in
"pr list")
	echo '[{"number": 7, "title": "Add dark mode", "state": "OPEN", "isDraft": false, "headRefName": "dark-mode", "url": "https://github.com/own/rep/pull/7", "headRepositoryOwner": {"login": "alice"}}]' ;;
"api graphql")
	echo '{"data": {"repository": {"pullRequest": {"number": 7, "title": "Add dark mode", "state": "OPEN", "baseRefName": "master", "headRefName": "dark-mode", "body": "Adds a dark mode toggle.", "author": {"login": "alice"}}}}}' ;;
*)
	exit 1 ;;
esac
`)
		shell.RunCommand([]string{"chmod", "+x", ".git/fake-gh"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Focus the branches window; the default (first) tab is Local Branches.
		t.Views().Branches().
			Focus().
			IsFocused()

		// One tab to the right is the Pull Requests tab, which lists the open
		// PRs gh reports. This proves it is positioned second.
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().PullRequests().
			IsFocused().
			Lines(
				Contains("#7").Contains("Add dark mode").IsSelected(),
			)
		// The main view previews the PR's overview, as in review mode.
		t.Views().Main().
			Content(
				Contains("#7 - Add dark mode").
					Contains("@alice").
					Contains("master ← dark-mode").
					Contains("Adds a dark mode toggle."),
			)

		// The next tab is Remotes, confirming the Pull Requests tab sits between
		// Local Branches and Remotes.
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().Remotes().
			IsFocused()
	},
})
