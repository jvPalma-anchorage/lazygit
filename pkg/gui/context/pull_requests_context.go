package context

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type PullRequestsContext struct {
	*FilteredListViewModel[*models.GithubPullRequest]
	*ListContextTrait
}

var _ types.IListContext = (*PullRequestsContext)(nil)

func NewPullRequestsContext(c *ContextCommon) *PullRequestsContext {
	// POC: the list is sourced once from the hardcoded sample seam. The live
	// path will swap this getter to read from Model().PullRequests (or a `gh`
	// fetch). Capturing once gives a stable slice for selection tracking.
	prs := samplePullRequests()

	viewModel := NewFilteredListViewModel(
		func() []*models.GithubPullRequest { return prs },
		func(pr *models.GithubPullRequest) []string {
			return []string{pr.Title, pr.ID()}
		},
	)

	getDisplayStrings := func(_ int, _ int) [][]string {
		return presentation.GetPullRequestListDisplayStrings(viewModel.GetItems())
	}

	return &PullRequestsContext{
		FilteredListViewModel: viewModel,
		ListContextTrait: &ListContextTrait{
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
		},
	}
}

// HandleFocus renders the (static) PR list before focusing. List contexts
// normally have their content rendered by the refresh cycle (see
// refresh_helper.go); this POC's source is static and wired to no refresh, so
// we render on focus to guarantee the view is populated. The live path will
// render via the refresh cycle instead and this override can go away.
func (self *PullRequestsContext) HandleFocus(opts types.OnFocusOpts) {
	self.HandleRender()
	self.ListContextTrait.HandleFocus(opts)
}
