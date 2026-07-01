package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewWorkspaceLayout proves the Phase 1 layout foundation of the PR review
// workspace: booted in review mode, lazygit owns its own three side windows
// (PR list, PR content, PR activity) with their own tabs and panel numbers. Tab
// cycling and panel jumps stay INSIDE the review workspace and never fall back to
// the normal Files/Branches/Commits windows (the bug #3 the workspace fixes).
//
// It deliberately asserts only the layout/navigation, not GitHub data — the panels
// are Phase 1 stubs.
var PrReviewWorkspaceLayout = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review mode owns its own windows/tabs/panel numbers; [/] cycling, 1/2/3 jumps and Esc never escape to the normal windows",
	ExtraCmdArgs: []string{"jesseduffield/lazygit", "1"},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file1", "content\n")
		shell.Commit("initial commit")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Boot focuses the review workspace (the Files Changed tab of the PR Content
		// window), never the normal Files window.
		t.Views().PrReview().
			IsFocused().
			Title(Equals("PR Review"))

		// Left/Right arrows (PrevBlock/NextBlock) cycle the review side panels:
		// from PR Content, Right → PR Activity, Left → back to PR Content.
		t.GlobalPress(keys.Universal.NextBlock)
		t.Views().PrConversation().IsFocused()
		t.GlobalPress(keys.Universal.PrevBlock)
		t.Views().PrReview().IsFocused()

		// `]` cycles WITHIN the PR Content window's tabs (Overview - Files Changed);
		// it does NOT escape to the normal Worktrees/Submodules tabs.
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().PrOverview().
			IsFocused().
			Title(Equals("Overview"))

		// Wrapping back lands on Files Changed again, confirming the window has
		// exactly its two review tabs.
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().PrReview().IsFocused()

		// `[` (previous tab) cycles the other way, still inside PR Content — it does
		// not escape backwards into a normal window either.
		t.GlobalPress(keys.Universal.PrevTab)
		t.Views().PrOverview().IsFocused()
		t.GlobalPress(keys.Universal.PrevTab)
		t.Views().PrReview().IsFocused()

		// `2` jumps to the PR Content window (panel [2]); its currently-on-top tab is
		// Files Changed.
		t.GlobalPress(keys.Universal.JumpToBlock[1])
		t.Views().PrReview().IsFocused()

		// `3` jumps to the PR Activity window (panel [3]); its default tab is
		// Conversation.
		t.GlobalPress(keys.Universal.JumpToBlock[2])
		t.Views().PrConversation().
			IsFocused().
			Title(Equals("Conversation"))

		// `]` inside PR Activity cycles Conversation - Checks - Commits, all review
		// windows — never the normal commits/reflog tabs.
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().PrChecks().
			IsFocused().
			Title(Equals("Checks"))
		t.GlobalPress(keys.Universal.NextTab)
		t.Views().PrCommits().
			IsFocused().
			Title(Equals("Commits"))

		// `1` jumps to the PR list window (panel [1]).
		t.GlobalPress(keys.Universal.JumpToBlock[0])
		t.Views().PrList().IsFocused()

		// Escape stays inside the review workspace: the default side context is the
		// PR list, never the normal Files window.
		t.GlobalPress(keys.Universal.Return)
		t.Views().PrList().IsFocused()
	},
})
