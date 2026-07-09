package context

import (
	"encoding/json"
	"os"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// PrChecksContext is the PR review workspace's Checks tab (window prActivity): the
// pull request's checks normalized into a two-level tree — workflow parents with
// their checks as children — sorted by importance (failure first, skipped last) then
// recency (task 8.2). Parents collapse/expand; entering a check renders its logs in
// the main view. Data comes from `gh pr checks` (or the LAZYGIT_PR_CHECKS_FIXTURE
// seam in tests).
type PrChecksContext struct {
	*ListViewModel[*PrCheckRow]
	*ListContextTrait

	c *ContextCommon

	checks      []git_commands.PrCheck
	rows        []*PrCheckRow
	collapsed   map[string]bool
	loadStarted bool
	loadErr     error
}

// PrCheckRow is one visible line of the checks tree: a workflow parent
// (Check == nil) or a check leaf.
type PrCheckRow struct {
	Workflow string
	Check    *git_commands.PrCheck
}

// ID implements HasID for the list view-model.
func (r *PrCheckRow) ID() string {
	if r.Check != nil {
		return r.Workflow + "/" + r.Check.Name + "/" + r.Check.Link
	}
	return "workflow:" + r.Workflow
}

var _ types.IListContext = (*PrChecksContext)(nil)

func NewPrChecksContext(c *ContextCommon) *PrChecksContext {
	self := &PrChecksContext{c: c, collapsed: map[string]bool{}}

	viewModel := NewListViewModel(func() []*PrCheckRow { return self.rows })
	self.ListViewModel = viewModel

	// SINGLE column per row (not icon | name): with the icon and name in separate
	// table columns, the aligner padded column 0 to the widest workflow-parent name
	// (~48 chars for anchorage's slack workflows), pushing every leaf's name off the
	// right edge of the panel so only the bare icon showed. One column keeps the name
	// adjacent to its icon regardless of the longest parent's width.
	getDisplayStrings := func(_ int, _ int) [][]string {
		if self.loadErr != nil {
			return [][]string{{style.FgRed.Sprint(self.loadErr.Error())}}
		}
		if len(self.rows) == 0 {
			return [][]string{{style.FgYellow.Sprint(c.Tr.PrChecksEmpty)}}
		}
		return lo.Map(self.rows, func(row *PrCheckRow, _ int) []string {
			return []string{checkRowLine(row, self.collapsed[row.Workflow])}
		})
	}

	baseContext := NewBaseContext(NewBaseContextOpts{
		Kind:       types.SIDE_CONTEXT,
		View:       c.Views().PrChecks,
		WindowName: "prActivity",
		Key:        PR_CHECKS_CONTEXT_KEY,
		Focusable:  true,
	})

	self.ListContextTrait = &ListContextTrait{
		Context: NewSimpleContext(baseContext),
		ListRenderer: ListRenderer{
			list:              viewModel,
			getDisplayStrings: getDisplayStrings,
		},
		c: c,
	}

	return self
}

func (self *PrChecksContext) currentTarget() (string, string, int) {
	review := self.reviewContext()
	if review == nil {
		return "", "", 0
	}
	return review.Target()
}

func (self *PrChecksContext) reviewContext() *PrReviewContext {
	ctx, ok := self.c.ContextForKey(PR_REVIEW_CONTEXT_KEY).(*PrReviewContext)
	if !ok {
		return nil
	}
	return ctx
}

// Load fetches the PR's checks once (idempotent, driven from the controller's render
// hook like the other Activity tabs). The fixture seam loads synchronously; the gh
// path fetches on a worker.
func (self *PrChecksContext) Load() {
	if self.loadStarted {
		return
	}
	review := self.reviewContext()
	if review == nil {
		return
	}
	owner, repo, number := review.Target()
	if number == 0 {
		return
	}
	self.loadStarted = true

	if path := os.Getenv("LAZYGIT_PR_CHECKS_FIXTURE"); path != "" {
		raw, err := os.ReadFile(path)
		if err == nil {
			var checks []git_commands.PrCheck
			err = json.Unmarshal(raw, &checks)
			if err == nil {
				self.applyChecks(checks)
				return
			}
		}
		self.loadErr = err
		self.c.PostRefreshUpdate(self)
		return
	}

	// Warm render from the per-PR snapshot (D3.1: checks are a cached data type
	// too), then revalidate: checks are the most volatile review data — reruns
	// change them without the head moving — so the snapshot is always refreshed in
	// the background, freshest-wins.
	store := git_commands.NewReviewSnapshotStore(config.ConfigDir(), owner, repo, number)
	var cached []git_commands.PrCheck
	warm := false
	if _, _, ok := store.Read("checks", &cached); ok {
		self.applyChecks(cached)
		warm = true
	}
	if warm && os.Getenv("LAZYGIT_PR_REVIEW_OFFLINE") != "" {
		// Same test/airplane seam as the PR-data warm boot: with a warm snapshot,
		// skip the background refresh entirely.
		return
	}

	self.c.OnWorker(func(_ gocui.Task) error {
		checks, err := self.c.Git().GitHub.FetchPRChecks(owner, repo, number)
		if err == nil {
			if writeErr := store.Write("checks", "", checks); writeErr != nil {
				self.c.Log.Warnf("pr checks cache: %v", writeErr)
			}
		}
		self.c.OnUIThread(func() error {
			// A Retarget (which reloads this tab) may have superseded this fetch;
			// never overwrite the new PR's checks with the old PR's result.
			if curOwner, curRepo, curNumber := self.currentTarget(); curOwner != owner || curRepo != repo || curNumber != number {
				return nil
			}
			if err != nil {
				if warm {
					// Keep the cached tree; a failed refresh must not replace a
					// working view with an error screen.
					self.c.Log.Warnf("pr checks refresh failed (showing cached): %v", err)
					return nil
				}
				// Leave loadStarted true — checks failing to load renders the error;
				// a manual refresh (Reload) retries.
				self.loadErr = err
			} else {
				self.applyChecks(checks)
				return nil
			}
			self.c.PostRefreshUpdate(self)
			return nil
		})
		return nil
	})
}

// applyChecks installs the fetched checks: importance-sorted, grouped by workflow
// into parent rows, then rendered. Must run on the UI thread.
func (self *PrChecksContext) applyChecks(checks []git_commands.PrCheck) {
	self.loadErr = nil
	git_commands.SortChecksByImportance(checks)
	self.checks = checks
	self.rebuildRows()
	self.c.PostRefreshUpdate(self)
}

// rebuildRows flattens the workflow→checks tree into visible rows, honoring
// collapsed parents.
func (self *PrChecksContext) rebuildRows() {
	self.rows = groupChecksRows(self.checks, self.c.Tr.PrChecksUngrouped, self.collapsed)
}

// groupChecksRows builds the visible rows: each workflow's checks render
// CONTIGUOUSLY under their parent (a naive single pass scatters a workflow's checks
// under other parents whenever the importance sort interleaves workflows — workflow
// verification finding). Workflows are ordered by their most important check (first
// appearance in the sorted input), children keep the sorted order. Pure for
// unit-testability.
func groupChecksRows(checks []git_commands.PrCheck, ungroupedTitle string, collapsed map[string]bool) []*PrCheckRow {
	workflowOrder := []string{}
	byWorkflow := map[string][]*git_commands.PrCheck{}
	for i := range checks {
		check := &checks[i]
		workflow := check.Workflow
		if workflow == "" {
			workflow = ungroupedTitle
		}
		if _, ok := byWorkflow[workflow]; !ok {
			workflowOrder = append(workflowOrder, workflow)
		}
		byWorkflow[workflow] = append(byWorkflow[workflow], check)
	}

	rows := []*PrCheckRow{}
	for _, workflow := range workflowOrder {
		rows = append(rows, &PrCheckRow{Workflow: workflow})
		if collapsed[workflow] {
			continue
		}
		for _, check := range byWorkflow[workflow] {
			rows = append(rows, &PrCheckRow{Workflow: workflow, Check: check})
		}
	}
	return rows
}

// ToggleCollapsed flips a workflow parent's collapsed state and re-renders.
func (self *PrChecksContext) ToggleCollapsed(workflow string) {
	self.collapsed[workflow] = !self.collapsed[workflow]
	self.rebuildRows()
	self.c.PostRefreshUpdate(self)
}

// Reload clears the guard so the next render re-fetches (PR refresh).
func (self *PrChecksContext) Reload() {
	self.loadStarted = false
	self.loadErr = nil
	self.Load()
}

// SelectedRow returns the row under the cursor, or nil.
func (self *PrChecksContext) SelectedRow() *PrCheckRow {
	return self.GetSelected()
}

// checkRowLine renders one row as a single display cell: a collapsible workflow
// parent (arrow + workflow) or an indented leaf (icon + check name). A check with an
// empty name falls back to its workflow so a row is never blank. Pure, for testing.
func checkRowLine(row *PrCheckRow, collapsed bool) string {
	if row.Check == nil {
		arrow := "▼"
		if collapsed {
			arrow = "▶"
		}
		return style.FgCyan.SetBold().Sprint(arrow + " " + row.Workflow)
	}
	name := row.Check.Name
	if name == "" {
		name = row.Check.Workflow
	}
	return "  " + checkBucketIcon(row.Check.Bucket) + " " + style.FgDefault.Sprint(name)
}

// checkBucketIcon maps a gh bucket to a colored glyph, matching the importance
// ordering: failures scream, skips whisper.
func checkBucketIcon(bucket string) string {
	switch bucket {
	case "fail":
		return style.FgRed.Sprint("✗")
	case "cancel":
		return style.FgYellow.Sprint("■")
	case "pending":
		return style.FgYellow.Sprint("●")
	case "pass":
		return style.FgGreen.Sprint("✓")
	case "skipping":
		return style.FgDefault.Sprint("-")
	default:
		return style.FgDefault.Sprint("?")
	}
}
