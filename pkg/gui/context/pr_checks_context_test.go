package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// The importance sort interleaves workflows (CI-fail, Lint-fail, CI-pass); each
// workflow's checks must still render contiguously under their own parent.
func TestGroupChecksRowsKeepsWorkflowsContiguous(t *testing.T) {
	checks := []git_commands.PrCheck{
		{Workflow: "CI", Name: "ci-fail", Bucket: "fail"},
		{Workflow: "Lint", Name: "lint-fail", Bucket: "fail"},
		{Workflow: "CI", Name: "ci-pass", Bucket: "pass"},
	}

	rows := groupChecksRows(checks, "Other", map[string]bool{})

	labels := []string{}
	for _, row := range rows {
		if row.Check != nil {
			labels = append(labels, row.Workflow+"/"+row.Check.Name)
		} else {
			labels = append(labels, row.Workflow)
		}
	}
	assert.Equal(t, []string{"CI", "CI/ci-fail", "CI/ci-pass", "Lint", "Lint/lint-fail"}, labels)
}

func TestGroupChecksRowsCollapsedAndUngrouped(t *testing.T) {
	checks := []git_commands.PrCheck{
		{Workflow: "", Name: "loose", Bucket: "pass"},
		{Workflow: "CI", Name: "ci", Bucket: "pass"},
	}

	rows := groupChecksRows(checks, "Other", map[string]bool{"CI": true})

	labels := []string{}
	for _, row := range rows {
		if row.Check != nil {
			labels = append(labels, row.Workflow+"/"+row.Check.Name)
		} else {
			labels = append(labels, row.Workflow)
		}
	}
	// The empty workflow falls under the ungrouped title; collapsed CI hides its child.
	assert.Equal(t, []string{"Other", "Other/loose", "CI"}, labels)
}

func TestCheckRowLineKeepsNameWithIcon(t *testing.T) {
	// A leaf renders as a SINGLE cell holding both the bucket glyph and the check
	// name — never split into columns (which the aligner would pad by the longest
	// workflow name, clipping the leaf name off-screen).
	leaf := &PrCheckRow{Workflow: "Some very long workflow name that would pad columns", Check: &git_commands.PrCheck{Name: "the-check-name", Bucket: "pass"}}
	line := utils.Decolorise(checkRowLine(leaf, false))
	assert.Contains(t, line, "the-check-name")
	// The name sits right after the icon: the visible run is short, independent of
	// any sibling workflow's width.
	assert.Less(t, len(line), 30)

	// A nameless check falls back to its workflow rather than rendering blank.
	nameless := &PrCheckRow{Workflow: "Trunk Merge Queue", Check: &git_commands.PrCheck{Name: "", Workflow: "Trunk Merge Queue", Bucket: "pass"}}
	assert.Contains(t, utils.Decolorise(checkRowLine(nameless, false)), "Trunk Merge Queue")

	// A collapsed parent shows the ▶ affordance.
	parent := &PrCheckRow{Workflow: "CI"}
	assert.Contains(t, utils.Decolorise(checkRowLine(parent, true)), "▶ CI")
	assert.Contains(t, utils.Decolorise(checkRowLine(parent, false)), "▼ CI")
}
