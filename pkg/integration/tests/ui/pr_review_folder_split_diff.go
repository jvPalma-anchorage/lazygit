package ui

import (
	"encoding/json"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewFolderSplitDiff proves that a folder's aggregate diff splits like lazygit's
// unstaged/staged view once some of its files are marked viewed: the unviewed files'
// diffs render in the top (main) pane and the viewed files' diffs in the bottom
// (secondary) pane. "Viewed" plays the role of "staged".
var PrReviewFolderSplitDiff = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: a folder's diff splits into unviewed (top) and viewed (bottom) panes",
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
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/base", "HEAD"})

		// Three files under one folder, each with distinctive content.
		shell.CreateFileAndAdd("src/foo.txt", "FOO_CONTENT\n")
		shell.CreateFileAndAdd("src/bar.txt", "BAR_CONTENT\n")
		shell.CreateFileAndAdd("src/baz.txt", "BAZ_CONTENT\n")
		shell.Commit("pr head")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/head", "HEAD"})

		fixture := struct {
			Data  *git_commands.PullRequestReviewData
			Files []any
		}{
			Data: &git_commands.PullRequestReviewData{
				ID: "PR_7", Number: 7, Title: "Split-diff PR", State: "OPEN", HeadRefOid: "head7",
			},
		}
		raw, err := json.Marshal(fixture)
		if err != nil {
			panic(err)
		}
		shell.CreateFile("pr_fixture.json", string(raw))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Mark only foo.txt as viewed.
		t.Views().PrReview().
			IsFocused().
			NavigateToLine(Contains("foo.txt"))
		t.Views().PrReview().Press(keys.Universal.Select)
		// foo.txt's name is now green; its parent folder "src" turns yellow (partially
		// viewed — foo viewed, bar/baz not).
		t.Views().PrReview().
			ContainsColoredText("#008000", "foo.txt").
			ContainsColoredText("#808000", "src")

		// Selecting the folder now splits the diff: unviewed (bar, baz) on top, viewed
		// (foo) on the bottom.
		t.Views().PrReview().NavigateToLine(Contains("src"))

		t.Views().Main().
			Content(Contains("BAR_CONTENT")).
			Content(Contains("BAZ_CONTENT")).
			Content(DoesNotContain("FOO_CONTENT"))

		t.Views().Secondary().
			IsVisible().
			Content(Contains("FOO_CONTENT")).
			Content(DoesNotContain("BAR_CONTENT"))

		// SPACE on the folder marks EVERY file under it viewed as a group: the folder
		// (and all its files) turn green.
		t.Views().PrReview().NavigateToLine(Contains("src"))
		t.Views().PrReview().Press(keys.Universal.Select)
		t.Views().PrReview().
			ContainsColoredText("#008000", "src").
			ContainsColoredText("#008000", "foo.txt").
			ContainsColoredText("#008000", "bar.txt").
			ContainsColoredText("#008000", "baz.txt").
			DoesNotContainColoredText("#808000", "src")

		// SPACE again on the fully-viewed folder unviews the whole group.
		t.Views().PrReview().Press(keys.Universal.Select)
		t.Views().PrReview().
			DoesNotContainColoredText("#008000", "foo.txt").
			DoesNotContainColoredText("#008000", "bar.txt").
			DoesNotContainColoredText("#008000", "baz.txt")
	},
})
