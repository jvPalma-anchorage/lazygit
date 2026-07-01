package context

import (
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// PrCommitsContext is the PR review workspace's Commits tab (window prActivity). It
// lists ONLY the commits between the PR's base and head (base..head), in a private
// slice — deliberately not Model().Commits — so it never clobbers the normal commits
// panel (design: a scoped context, not LocalCommitsContext). The base/head endpoints
// come from the live PR-review (Files Changed) context, so the list refreshes whenever
// the PR reloads.
type PrCommitsContext struct {
	*ListViewModel[*models.Commit]
	*ListContextTrait

	c *ContextCommon

	commits     []*models.Commit
	loadStarted bool
}

var _ types.IListContext = (*PrCommitsContext)(nil)

func NewPrCommitsContext(c *ContextCommon) *PrCommitsContext {
	self := &PrCommitsContext{c: c}

	viewModel := NewListViewModel(func() []*models.Commit { return self.commits })
	self.ListViewModel = viewModel

	getDisplayStrings := func(_ int, _ int) [][]string {
		if len(self.commits) == 0 {
			return [][]string{{style.FgYellow.Sprint(c.Tr.PrReviewNoCommits)}}
		}
		return lo.Map(self.commits, func(commit *models.Commit, _ int) []string {
			hash := commit.Hash()
			if len(hash) > 8 {
				hash = hash[:8]
			}
			return []string{style.FgYellow.Sprint(hash), commit.Name}
		})
	}

	baseContext := NewBaseContext(NewBaseContextOpts{
		Kind:       types.SIDE_CONTEXT,
		View:       c.Views().PrCommits,
		WindowName: "prActivity",
		Key:        PR_COMMITS_CONTEXT_KEY,
		Focusable:  true,
	})
	simpleContext := NewSimpleContext(baseContext)

	self.ListContextTrait = &ListContextTrait{
		Context: simpleContext,
		ListRenderer: ListRenderer{
			list:              viewModel,
			getDisplayStrings: getDisplayStrings,
		},
		c: c,
	}

	return self
}

// reviewContext returns the live PR-review (Files Changed) context, which owns the
// resolved base/head endpoints.
func (self *PrCommitsContext) reviewContext() *PrReviewContext {
	ctx, ok := self.c.ContextForKey(PR_REVIEW_CONTEXT_KEY).(*PrReviewContext)
	if !ok {
		return nil
	}
	return ctx
}

// Load fetches the base..head commits via the commit loader the first time the tab is
// rendered with resolved endpoints. It is driven from the controller's render hook
// (GetOnRenderToMain), which fires reliably on focus — unlike the context's on-focus
// callback. Idempotent; Reload re-fetches after a PR refresh.
func (self *PrCommitsContext) Load() {
	if self.loadStarted {
		return
	}
	review := self.reviewContext()
	if review == nil {
		return
	}
	// Use the base and head REFS (not the diff's merge-base): `git log base..head`
	// lists exactly the commits on head not on the base branch (GitHub's Commits-tab
	// semantics), which differs from the merge-base when the PR merged base into itself.
	base, head, ok := review.LocalCommitRange()
	if !ok {
		// The PR hasn't finished loading its refs yet; try again on the next render.
		return
	}

	commits, err := self.c.Git().Loaders.CommitLoader.GetCommits(git_commands.GetCommitsOptions{
		RefName:      base + ".." + head,
		Limit:        true,
		MainBranches: self.c.Model().MainBranches,
		HashPool:     self.c.Model().HashPool,
	})
	if err != nil {
		// Leave loadStarted false so a transient git error retries on the next render.
		self.c.Log.Errorf("loading PR commits for %s..%s: %v", base, head, err)
		return
	}
	self.loadStarted = true
	self.commits = commits
	self.c.PostRefreshUpdate(self)
}

// Reload clears the load guard so the next focus re-fetches (e.g. after the PR's refs
// move on a refresh).
func (self *PrCommitsContext) Reload() {
	self.loadStarted = false
	self.Load()
}

// SelectedCommit returns the commit the cursor is on, or nil.
func (self *PrCommitsContext) SelectedCommit() *models.Commit {
	return self.GetSelected()
}
