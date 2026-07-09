package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewViewedState drives the Phase 4 persisted, content-invalidated viewed state.
// It marks two files viewed, refreshes (proving the marks survive a reload by being
// read back from AppState keyed by blob OID), then changes one file's content on the
// head ref and refreshes again — proving that file's mark auto-clears (its blob OID
// changed) while the untouched file stays viewed.
// viewedGreen is tcell's ANSI green (style.FgGreen), the color a viewed file's name
// is painted (mirroring lazygit's staged/unstaged coloring).
const viewedGreen = "#008000"

var PrReviewViewedState = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: viewed marks persist across reload and auto-clear when a file's content changes",
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

		shell.CreateFileAndAdd("src/foo.txt", "alpha\nbeta\n")
		shell.CreateFileAndAdd("src/bar.txt", "one\n")
		shell.Commit("pr base")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/base", "HEAD"})

		shell.CreateFileAndAdd("src/foo.txt", "alpha\nbeta\nFOO1\n")
		shell.CreateFileAndAdd("src/bar.txt", "one\nBAR1\n")
		shell.Commit("pr head")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/head", "HEAD"})

		fixture := struct {
			Data  *git_commands.PullRequestReviewData
			Files []any
		}{
			Data: &git_commands.PullRequestReviewData{
				ID: "PR_7", Number: 7, Title: "Viewed-state PR", State: "OPEN", HeadRefOid: "head7",
			},
		}
		raw, err := json.Marshal(fixture)
		if err != nil {
			panic(err)
		}
		shell.CreateFile("pr_fixture.json", string(raw))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Mark both files viewed; the filename turns green (the viewed state is now a
		// color, not a checkbox). This persists to AppState.
		t.Views().PrReview().NavigateToLine(Contains("foo.txt"))
		t.Views().PrReview().Press(keys.Universal.Select)
		t.Views().PrReview().ContainsColoredText(viewedGreen, "foo.txt")
		t.Views().PrReview().NavigateToLine(Contains("bar.txt"))
		t.Views().PrReview().Press(keys.Universal.Select)
		t.Views().PrReview().ContainsColoredText(viewedGreen, "bar.txt")

		// Refresh without changing anything: both marks are restored from AppState
		// (persistence — they would be lost if the marks were session-only).
		t.Views().PrReview().Press(keys.Universal.Refresh)
		t.Views().PrReview().
			ContainsColoredText(viewedGreen, "foo.txt").
			ContainsColoredText(viewedGreen, "bar.txt")

		// Change foo.txt's content on the head ref (a new commit), then refresh: foo's
		// blob OID changed so its mark auto-clears (name no longer green), but bar
		// (untouched) stays viewed.
		t.Shell().CreateFileAndAdd("src/foo.txt", "alpha\nbeta\nFOO1\nFOO2\n")
		t.Shell().Commit("update foo on head")
		t.Shell().RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/head", "HEAD"})

		t.Views().PrReview().Press(keys.Universal.Refresh)
		t.Views().PrReview().
			DoesNotContainColoredText(viewedGreen, "foo.txt").
			ContainsColoredText(viewedGreen, "bar.txt")
	},
})
