package context

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// prOverviewCache holds pull requests' full review data for the overview
// preview shown while hovering a PR in a list. It is backed by the PR review
// snapshot store on disk (the same meta snapshot that review mode reads and
// writes), so a PR seen in an earlier run shows instantly; each PR is then
// refetched once per run, off the UI thread, to bring it up to date.
type prOverviewCache struct {
	c *ContextCommon
	// data holds the PRs loaded so far; requested records which PRs have had a
	// fetch started this run.
	data      map[string]*git_commands.PullRequestReviewData
	requested map[string]bool
}

func newPrOverviewCache(c *ContextCommon) *prOverviewCache {
	return &prOverviewCache{
		c:         c,
		data:      map[string]*git_commands.PullRequestReviewData{},
		requested: map[string]bool{},
	}
}

// get returns the PR's review data, from memory or disk, or nil if there is
// none yet. The first call for a PR also starts fetching it; onLoaded runs on
// the UI thread once the fresh data arrives.
func (self *prOverviewCache) get(owner string, repo string, number int, onLoaded func()) *git_commands.PullRequestReviewData {
	id := fmt.Sprintf("%s/%s#%d", owner, repo, number)
	store := git_commands.NewReviewSnapshotStore(config.ConfigDir(), owner, repo, number)

	data, ok := self.data[id]
	if !ok {
		var cached git_commands.PullRequestReviewData
		if _, _, hit := store.Read("meta", &cached); hit {
			data = &cached
			self.data[id] = data
		}
	}

	if !self.requested[id] {
		self.requested[id] = true
		self.fetch(id, store, owner, repo, number, onLoaded)
	}

	return data
}

func (self *prOverviewCache) fetch(id string, store *git_commands.ReviewSnapshotStore, owner string, repo string, number int, onLoaded func()) {
	self.c.OnWorker(func(_ gocui.Task) error {
		token := self.c.Git().GitHub.GetAuthToken("github.com")
		data, err := self.c.Git().GitHub.FetchPRReviewData(owner, repo, number, token)
		if err == nil {
			if writeErr := store.Write("meta", data.HeadRefOid, data); writeErr != nil {
				self.c.Log.Warnf("pr overview %s: caching: %v", id, writeErr)
			}
		}

		self.c.OnUIThread(func() error {
			if err != nil {
				// Let the next hover try again.
				self.requested[id] = false
				self.c.Log.Warnf("pr overview %s: %v", id, err)
				return nil
			}
			self.data[id] = data
			onLoaded()
			return nil
		})
		return nil
	})
}
