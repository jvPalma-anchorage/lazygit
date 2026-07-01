package ui

import (
	"encoding/json"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// PrReviewFileFiltering proves two things about the Files Changed tree:
//   - it is NOT capped at 10 files (a 15-file PR shows all 15), and
//   - generated files (lockfiles, *.generated.*) are hidden by default and revealed
//     with `G`.
var PrReviewFileFiltering = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "PR review: the file tree shows all files (no 10-file cap) and hides generated files until toggled",
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

		// 15 source files (more than the old cap of 10) plus two generated files.
		for i := range 15 {
			shell.CreateFileAndAdd(fmt.Sprintf("file%02d.go", i), fmt.Sprintf("package p%02d\n", i))
		}
		shell.CreateFileAndAdd("yarn.lock", "# lockfile\n")
		shell.CreateFileAndAdd("schema.generated.ts", "// codegen\n")
		shell.Commit("pr head")
		shell.RunCommand([]string{"git", "update-ref", "refs/lazygit-review/7/head", "HEAD"})

		fixture := struct {
			Data  *git_commands.PullRequestReviewData
			Files []any
		}{
			Data: &git_commands.PullRequestReviewData{
				ID: "PR_7", Number: 7, Title: "File-filtering PR", State: "OPEN", HeadRefOid: "head7",
			},
		}
		raw, err := json.Marshal(fixture)
		if err != nil {
			panic(err)
		}
		shell.CreateFile("pr_fixture.json", string(raw))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// No 10-file cap: the 11th and 15th files are present.
		t.Views().PrReview().
			IsFocused().
			ContainsLines(Contains("file00.go")).
			ContainsLines(Contains("file10.go")).
			ContainsLines(Contains("file14.go")).
			// Generated files are hidden by default.
			Content(DoesNotContain("yarn.lock")).
			Content(DoesNotContain("schema.generated.ts"))

		// `G` reveals the generated files.
		t.GlobalPress(config.Keybinding{"G"})
		t.ExpectToast(Equals("Showing generated files"))
		t.Views().PrReview().
			ContainsLines(Contains("yarn.lock")).
			ContainsLines(Contains("schema.generated.ts"))

		// `G` again hides them.
		t.GlobalPress(config.Keybinding{"G"})
		t.ExpectToast(Equals("Hiding generated files"))
		t.Views().PrReview().Content(DoesNotContain("yarn.lock"))
	},
})
