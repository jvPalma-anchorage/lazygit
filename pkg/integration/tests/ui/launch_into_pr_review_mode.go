package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// LaunchIntoPrReviewMode proves the `lazygit OWNER/REPO PR_NUMBER` boot path:
// the two bare positionals are detected as PR review mode, the dedicated PR
// review context becomes the focused boot context (instead of the normal Files
// context), and its view renders with the review-mode title.
//
// What it deliberately does NOT assert: the changed-file list, the inline diff,
// review threads, reviewers, or the conversation section. All of those come from
// a live GitHub fetch (FetchPRReviewData / FetchPRChangedFiles over net/http to
// github.com), which cannot succeed in the hermetic integration-test sandbox.
// Without that data the context only ever renders the loading placeholder (and
// then a load error once the unauthenticated fetch fails), so file/diff/thread
// assertions are intentionally out of scope here. See the package notes in the
// pr-review-mode change for the live-GitHub gaps.
var LaunchIntoPrReviewMode = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Launching with `OWNER/REPO PR_NUMBER` positionals boots directly into PR review mode with the review context focused",
	// The two bare positionals select PR review mode at boot.
	ExtraCmdArgs: []string{"jesseduffield/lazygit", "1"},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		// Review mode only requires that the cwd is a git work tree (gh-dash
		// `cd`s into the checkout before launching); it does not require the
		// remote to match OWNER/REPO. A minimal repo is enough to pass the boot
		// guard in app.go's ensureReviewModeRepo.
		shell.CreateFileAndAdd("file1", "content\n")
		shell.Commit("initial commit")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// The PR review view is the focused context on boot (not Files), with the
		// review-mode title. This is the host-independent, data-independent signal
		// that the OWNER/REPO PR_NUMBER CLI grammar routed into review mode.
		t.Views().PrReview().
			IsFocused().
			Title(Equals("PR Review"))
	},
})
