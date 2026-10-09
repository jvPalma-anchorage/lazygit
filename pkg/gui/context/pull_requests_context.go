package context

import (
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

type PullRequestsContext struct {
	*FilteredListViewModel[*models.GithubPullRequest]
	*ListContextTrait

	prs []*models.GithubPullRequest
	// Set while the tab is unfocused, so that the next focus reloads the list.
	// A refresh re-focuses the tab without it losing focus first, so this keeps
	// the reload's own refresh from triggering another reload.
	stale bool

	overviewCache *prOverviewCache
}

var _ types.IListContext = (*PullRequestsContext)(nil)

func NewPullRequestsContext(c *ContextCommon) *PullRequestsContext {
	self := &PullRequestsContext{stale: true, overviewCache: newPrOverviewCache(c)}

	viewModel := NewFilteredListViewModel(
		func() []*models.GithubPullRequest { return self.prs },
		func(pr *models.GithubPullRequest) []string {
			return []string{pr.Title, pr.ID()}
		},
	)

	getDisplayStrings := func(_ int, _ int) [][]string {
		return presentation.GetPullRequestListDisplayStrings(viewModel.GetItems())
	}

	self.FilteredListViewModel = viewModel
	self.ListContextTrait = &ListContextTrait{
		Context: NewSimpleContext(NewBaseContext(NewBaseContextOpts{
			View:       c.Views().PullRequests,
			WindowName: "branches",
			Key:        PULL_REQUESTS_CONTEXT_KEY,
			Kind:       types.SIDE_CONTEXT,
			Focusable:  true,
		})),
		ListRenderer: ListRenderer{
			list:              viewModel,
			getDisplayStrings: getDisplayStrings,
		},
		c: c,
	}

	return self
}

// HandleFocus reloads the repo's open pull requests whenever the tab is
// entered, so the list is current without wiring it into the refresh cycle.
// The previous list stays on screen until the new one arrives; on the first
// entry that is the list cached on disk by the last run.
func (self *PullRequestsContext) HandleFocus(opts types.OnFocusOpts) {
	if self.stale {
		self.stale = false
		if self.prs == nil {
			self.prs = self.loadFromCache()
		}
		self.load()
	}
	self.HandleRender()
	self.ListContextTrait.HandleFocus(opts)
}

func (self *PullRequestsContext) HandleFocusLost(opts types.OnFocusLostOpts) {
	self.stale = true
	self.ListContextTrait.HandleFocusLost(opts)
}

func (self *PullRequestsContext) loadFromCache() []*models.GithubPullRequest {
	cached, err := self.c.GetConfig().GetCachedOpenPullRequests(self.c.Git().RepoPaths.RepoPath())
	if err != nil {
		self.c.Log.Warnf("error loading open pull request cache: %v", err)
	}

	return lo.Map(cached, func(cached config.CachedPullRequest, _ int) *models.GithubPullRequest {
		return &models.GithubPullRequest{
			HeadRefName:         cached.HeadRefName,
			Number:              cached.Number,
			Title:               cached.Title,
			State:               cached.State,
			Url:                 cached.Url,
			HeadRepositoryOwner: models.GithubRepositoryOwner{Login: cached.HeadRepositoryOwner},
		}
	})
}

func (self *PullRequestsContext) saveToCache(repoPath string, prs []*models.GithubPullRequest) {
	cached := lo.Map(prs, func(pr *models.GithubPullRequest, _ int) config.CachedPullRequest {
		return config.CachedPullRequest{
			HeadRefName:         pr.HeadRefName,
			Number:              pr.Number,
			Title:               pr.Title,
			State:               pr.State,
			Url:                 pr.Url,
			HeadRepositoryOwner: pr.HeadRepositoryOwner.Login,
		}
	})

	if err := self.c.GetConfig().SaveCachedOpenPullRequests(repoPath, cached); err != nil {
		self.c.Log.Warnf("error saving open pull request cache: %v", err)
	}
}

func (self *PullRequestsContext) load() {
	// Capture the repo path here on the UI thread, so that a repo switch while
	// the fetch is in flight can't file this repo's PRs under the new repo.
	repoPath := self.c.Git().RepoPaths.RepoPath()
	self.c.OnWorker(func(_ gocui.Task) error {
		prs, err := self.c.Git().GitHub.ListOpenPullRequests()
		if err != nil {
			self.c.Log.Warnf("pull requests tab: %v", err)
			self.c.OnUIThread(func() error {
				self.c.ErrorToast(self.c.Tr.FailedToLoadPullRequests)
				return nil
			})
			return nil
		}

		self.saveToCache(repoPath, prs)
		self.c.OnUIThread(func() error {
			self.prs = prs
			self.c.PostRefreshUpdate(self)
			return nil
		})
		return nil
	})
}

// OverviewDataForSelected returns the selected PR's full review data for its
// overview preview, or nil while it is still being fetched (the list re-renders
// once it arrives).
func (self *PullRequestsContext) OverviewDataForSelected() *git_commands.PullRequestReviewData {
	pr := self.GetSelected()
	if pr == nil {
		return nil
	}
	owner, repo, ok := pr.BaseRepo()
	if !ok {
		return nil
	}

	return self.overviewCache.get(owner, repo, pr.Number, func() {
		self.c.PostRefreshUpdate(self)
	})
}
