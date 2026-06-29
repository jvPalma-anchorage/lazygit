package presentation

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/samber/lo"
)

// GetPullRequestListDisplayStrings renders the rows for the pull-requests tab,
// one row per PR. Each row is [number, title]; the number is colored by PR
// state via WithPrColor (the same coloring used for the per-branch PR badge).
// Mirrors GetRemoteListDisplayStrings.
func GetPullRequestListDisplayStrings(prs []*models.GithubPullRequest) [][]string {
	return lo.Map(prs, func(pr *models.GithubPullRequest, _ int) []string {
		return getPullRequestDisplayStrings(pr)
	})
}

func getPullRequestDisplayStrings(pr *models.GithubPullRequest) []string {
	number := WithPrColor(pr.State, fmt.Sprintf("#%d", pr.Number), false)
	return []string{number, pr.Title}
}
