package context

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// PrListContext is the PR review workspace's window [1]: a gh-dash-style PR browser.
// The gh-dash sections are the window's TABS (cycled with [ / ]); the list shows only
// the active section's pull requests (Phase 13). The launched PR is pinned as a
// "Current" first section so the panel always has content. Hovering a PR previews its
// Overview in the main panel; SPACE loads it for review.
type PrListContext struct {
	*ListViewModel[*git_commands.PrListEntry]
	*ListContextTrait

	c *ContextCommon

	sections      []prListSection
	activeSection int
	loadStarted   bool

	// overviewData caches each hovered PR's full review data (keyed by entry ID) so
	// the overview preview is instant on re-hover; overviewLoading guards against
	// firing a second fetch for a PR already in flight.
	overviewData    map[string]*git_commands.PullRequestReviewData
	overviewLoading map[string]bool
}

// prListSection is one gh-dash section: a tab title and its matching PRs.
type prListSection struct {
	title string
	prs   []*git_commands.PrListEntry
}

var _ types.IListContext = (*PrListContext)(nil)

func NewPrListContext(c *ContextCommon) *PrListContext {
	self := &PrListContext{
		c:               c,
		overviewData:    map[string]*git_commands.PullRequestReviewData{},
		overviewLoading: map[string]bool{},
	}

	viewModel := NewListViewModel(func() []*git_commands.PrListEntry { return self.activePRs() })
	self.ListViewModel = viewModel

	// Three uniform columns (the aligner panics on ragged rows): #number, title, state.
	getDisplayStrings := func(_ int, _ int) [][]string {
		prs := self.activePRs()
		if len(prs) == 0 {
			msg := c.Tr.PrReviewLoading
			if self.loadStarted && len(self.sections) > 0 {
				msg = c.Tr.PrListEmptySection
			}
			return [][]string{{style.FgYellow.Sprint(msg), "", ""}}
		}
		curOwner, curRepo, curNumber := self.currentTarget()
		return lo.Map(prs, func(e *git_commands.PrListEntry, _ int) []string {
			marker := "  "
			if e.Owner == curOwner && e.Repo == curRepo && e.Number == curNumber {
				marker = style.FgGreen.Sprint("▶ ")
			}
			return []string{
				marker + style.FgYellow.Sprint(fmt.Sprintf("#%d", e.Number)),
				style.FgDefault.Sprint(e.Title),
				colorizeReviewerState(e.State),
			}
		})
	}

	baseContext := NewBaseContext(NewBaseContextOpts{
		Kind:       types.SIDE_CONTEXT,
		View:       c.Views().PrList,
		WindowName: "prList",
		Key:        PR_LIST_CONTEXT_KEY,
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

func (self *PrListContext) reviewContext() *PrReviewContext {
	ctx, ok := self.c.ContextForKey(PR_REVIEW_CONTEXT_KEY).(*PrReviewContext)
	if !ok {
		return nil
	}
	return ctx
}

func (self *PrListContext) currentTarget() (string, string, int) {
	review := self.reviewContext()
	if review == nil {
		return "", "", 0
	}
	return review.Target()
}

// activePRs is the active section's PR slice, or nil.
func (self *PrListContext) activePRs() []*git_commands.PrListEntry {
	if self.activeSection < 0 || self.activeSection >= len(self.sections) {
		return nil
	}
	return self.sections[self.activeSection].prs
}

// Load populates the browser once: the launched PR immediately as the "Current"
// section (no network), then the gh-dash sections as they arrive from the worker.
// Idempotent; driven from the controller's render hook. The LAZYGIT_PR_LIST_FIXTURE
// seam replaces the gh-dash + search path for tests.
func (self *PrListContext) Load() {
	if self.loadStarted {
		return
	}
	owner, repo, number := self.currentTarget()
	if owner == "" {
		return
	}
	self.loadStarted = true

	self.sections = []prListSection{{
		title: self.c.Tr.PrListCurrentSection,
		prs: []*git_commands.PrListEntry{
			{Owner: owner, Repo: repo, Number: number, Title: self.launchedPrTitle(), State: "OPEN"},
		},
	}}
	self.applyTabs()
	self.c.PostRefreshUpdate(self)

	if path := os.Getenv("LAZYGIT_PR_LIST_FIXTURE"); path != "" {
		self.loadFixtureSections(path)
		return
	}

	defaultRepo := fmt.Sprintf("%s/%s", owner, repo)
	self.c.OnWorker(func(_ gocui.Task) error {
		specs := git_commands.LoadGhDashSections(defaultRepo)
		loaded := make([]prListSection, 0, len(specs))
		for _, spec := range specs {
			prs, err := self.c.Git().GitHub.SearchPrs(spec.Filters, defaultRepo)
			if err != nil {
				// A failing section (bad filter, offline) renders as empty rather
				// than killing the whole browser.
				self.c.Log.Warnf("pr list: section %q: %v", spec.Title, err)
				continue
			}
			loaded = append(loaded, prListSection{title: spec.Title, prs: toEntryPtrs(prs)})
		}

		self.c.OnUIThread(func() error {
			self.sections = append(self.sections, loaded...)
			self.applyTabs()
			self.c.PostRefreshUpdate(self)
			return nil
		})
		return nil
	})
}

func toEntryPtrs(prs []git_commands.PrListEntry) []*git_commands.PrListEntry {
	out := make([]*git_commands.PrListEntry, len(prs))
	for i := range prs {
		entry := prs[i]
		out[i] = &entry
	}
	return out
}

// applyTabs pushes the section titles into the view's tab bar so window [1] renders
// them as tabs with the active one highlighted.
func (self *PrListContext) applyTabs() {
	view := self.c.Views().PrList
	if view == nil {
		return
	}
	view.Tabs = lo.Map(self.sections, func(s prListSection, _ int) string { return s.title })
	view.TabIndex = self.activeSection
}

// NextSection / PrevSection cycle the active section tab, reset the cursor to the top
// of the new section, and re-render.
func (self *PrListContext) NextSection() {
	self.switchSection(self.activeSection + 1)
}

func (self *PrListContext) PrevSection() {
	self.switchSection(self.activeSection - 1)
}

func (self *PrListContext) switchSection(index int) {
	if len(self.sections) == 0 {
		return
	}
	self.activeSection = (index%len(self.sections) + len(self.sections)) % len(self.sections)
	self.SetSelection(0)
	self.applyTabs()
	self.c.PostRefreshUpdate(self)
}

// launchedPrTitle returns the loaded PR's title once the review data has arrived,
// else a plain placeholder.
func (self *PrListContext) launchedPrTitle() string {
	if review := self.reviewContext(); review != nil && review.Data() != nil {
		return review.Data().Title
	}
	return self.c.Tr.PrListLaunchedPlaceholder
}

// loadFixtureSections reads sections from LAZYGIT_PR_LIST_FIXTURE synchronously.
func (self *PrListContext) loadFixtureSections(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		self.c.Log.Warnf("pr list fixture: %v", err)
		return
	}
	var sections []struct {
		Title string
		Prs   []git_commands.PrListEntry
	}
	if err := json.Unmarshal(raw, &sections); err != nil {
		self.c.Log.Warnf("pr list fixture: %v", err)
		return
	}
	for _, section := range sections {
		self.sections = append(self.sections, prListSection{title: section.Title, prs: toEntryPtrs(section.Prs)})
	}
	self.applyTabs()
	self.c.PostRefreshUpdate(self)
}

// RefreshLaunchedTitle updates the pinned launched-PR row's title once the review
// data arrives (the row is created before the fetch completes).
func (self *PrListContext) RefreshLaunchedTitle() {
	if len(self.sections) > 0 && len(self.sections[0].prs) > 0 {
		self.sections[0].prs[0].Title = self.launchedPrTitle()
		self.c.PostRefreshUpdate(self)
	}
}

// SelectedEntry returns the PR under the cursor, or nil.
func (self *PrListContext) SelectedEntry() *git_commands.PrListEntry {
	return self.GetSelected()
}

// OverviewDataForSelected returns the full review data to render the selected PR's
// overview preview. For the currently-loaded PR it reuses the live review data; for
// any other PR it serves the on-hover fetch cache, kicking off a background fetch on a
// cache miss and returning nil (the controller then shows a lightweight header).
func (self *PrListContext) OverviewDataForSelected() *git_commands.PullRequestReviewData {
	entry := self.GetSelected()
	if entry == nil {
		return nil
	}

	if review := self.reviewContext(); review != nil && review.Data() != nil {
		curOwner, curRepo, curNumber := review.Target()
		if entry.Owner == curOwner && entry.Repo == curRepo && entry.Number == curNumber {
			return review.Data()
		}
	}

	if data, ok := self.overviewData[entry.ID()]; ok {
		return data
	}
	self.fetchOverview(entry)
	return nil
}

// fetchOverview lazily fetches a non-current PR's review data for its overview
// preview, once per PR (guarded), off the UI thread. Fixture mode skips the network.
func (self *PrListContext) fetchOverview(entry *git_commands.PrListEntry) {
	if os.Getenv("LAZYGIT_PR_LIST_FIXTURE") != "" {
		return
	}
	id := entry.ID()
	if self.overviewLoading[id] {
		return
	}
	self.overviewLoading[id] = true
	target := *entry

	self.c.OnWorker(func(_ gocui.Task) error {
		token := self.c.Git().GitHub.GetAuthToken("github.com")
		data, err := self.c.Git().GitHub.FetchPRReviewData(target.Owner, target.Repo, target.Number, token)
		self.c.OnUIThread(func() error {
			self.overviewLoading[id] = false
			if err != nil {
				self.c.Log.Warnf("pr list overview %s: %v", id, err)
				return nil
			}
			self.overviewData[id] = data
			self.c.PostRefreshUpdate(self)
			return nil
		})
		return nil
	})
}
