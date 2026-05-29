package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var HideStatusPanelRenumbersJumpKeys = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "With the status panel hidden, the first jump key and the [1] badge map to the Files panel",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Gui.ShowStatusPanel = false
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file1", "content\n")
		shell.Commit("initial commit")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// With status hidden, the badges renumber: Files becomes [1] and
		// Branches becomes [2] (rather than [2]/[3]).
		t.Views().Files().
			TitlePrefix(Equals("[1]"))
		t.Views().Branches().
			TitlePrefix(Equals("[2]"))

		// Move focus elsewhere (status is hidden, so the second jump key now
		// targets Branches), then the first jump key should return to Files.
		t.GlobalPress(keys.Universal.JumpToBlock[1])
		t.Views().Branches().IsFocused()

		t.GlobalPress(keys.Universal.JumpToBlock[0])
		t.Views().Files().IsFocused()
	},
})
