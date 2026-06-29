package context

import "github.com/jesseduffield/lazygit/pkg/commands/models"

// samplePullRequests is the hardcoded data source for the pull-requests tab
// proof-of-concept. It stands in for a live `gh` query (PRs where the user is
// author / review-requested / involved). Titles are deliberately prefixed
// "[SAMPLE]" so this stub is never mistaken for real pull-request data.
//
// This is the single seam the later live path will replace: swap the body to
// read from Model().PullRequests (or a `gh` fetch) and nothing else changes.
func samplePullRequests() []*models.GithubPullRequest {
	return []*models.GithubPullRequest{
		{
			Number:              101,
			Title:               "[SAMPLE] Add dark mode toggle",
			State:               "OPEN",
			Url:                 "https://github.com/sample-org/sample-repo/pull/101",
			HeadRefName:         "feature/dark-mode",
			HeadRepositoryOwner: models.GithubRepositoryOwner{Login: "sample-author"},
		},
		{
			Number:              102,
			Title:               "[SAMPLE] Fix flaky integration test",
			State:               "DRAFT",
			Url:                 "https://github.com/sample-org/sample-repo/pull/102",
			HeadRefName:         "fix/flaky-test",
			HeadRepositoryOwner: models.GithubRepositoryOwner{Login: "sample-reviewer"},
		},
		{
			Number:              103,
			Title:               "[SAMPLE] Bump dependencies",
			State:               "OPEN",
			Url:                 "https://github.com/sample-org/sample-repo/pull/103",
			HeadRefName:         "chore/bump-deps",
			HeadRepositoryOwner: models.GithubRepositoryOwner{Login: "sample-bot"},
		},
	}
}
