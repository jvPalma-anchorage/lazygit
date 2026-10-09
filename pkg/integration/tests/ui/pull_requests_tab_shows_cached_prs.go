package ui

import (
	"os"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullRequestsTabShowsCachedPrs = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "The Pull Requests tab shows the PRs and overview cached on disk by an earlier run while gh can't be reached",
	ExtraCmdArgs: []string{},
	ExtraEnvVars: map[string]string{
		GH_AVAILABLE_OVERRIDE_ENV_VAR: "true",
		// A gh that fails every call, as when offline: anything shown can only
		// have come from the cache.
		"GH_PATH":                     ".git/fake-gh",
		"LAZYGIT_PR_REVIEW_CACHE_DIR": "prcache",
	},
	Skip: false,
	SetupConfig: func(cfg *config.AppConfig) {
		// The test starts lazygit in the repo, so the working directory is the
		// repo path the cache is keyed by.
		repoPath, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		if err := cfg.SaveCachedOpenPullRequests(repoPath, []config.CachedPullRequest{
			{Number: 7, Title: "Cached dark mode", State: "OPEN", HeadRefName: "dark-mode", Url: "https://github.com/own/rep/pull/7", HeadRepositoryOwner: "alice"},
		}); err != nil {
			panic(err)
		}
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file1", "content\n")
		shell.Commit("initial commit")
		shell.CreateFile(".git/fake-gh", "#!/bin/sh\nexit 1\n")
		shell.RunCommand([]string{"chmod", "+x", ".git/fake-gh"})
		shell.CreateFile("prcache/own/rep/7/meta.json", `{"fetchedAt": "2026-07-01T00:00:00Z", "headRefOid": "", "payload": {
			"Number": 7, "Title": "Cached dark mode", "State": "OPEN", "Author": "alice",
			"BaseRefName": "master", "HeadRefName": "dark-mode", "Body": "Cached description."
		}}`)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Branches().
			Focus()

		t.GlobalPress(keys.Universal.NextTab)
		t.Views().PullRequests().
			IsFocused().
			Lines(
				Contains("#7").Contains("Cached dark mode").IsSelected(),
			)
		t.Views().Main().
			Content(
				Contains("#7 - Cached dark mode").
					Contains("master ← dark-mode").
					Contains("Cached description."),
			)

		// The refresh behind the cached list fails, and says so.
		t.ExpectToast(Contains("Failed to load pull requests"))
	},
})
