package models

import "strconv"

type GithubPullRequest struct {
	HeadRefName         string                `json:"headRefName"`
	Number              int                   `json:"number"`
	Title               string                `json:"title"`
	State               string                `json:"state"` // "MERGED", "OPEN", "CLOSED", "DRAFT"
	Url                 string                `json:"url"`
	HeadRepositoryOwner GithubRepositoryOwner `json:"headRepositoryOwner"`
}

// ID satisfies the HasID constraint used by list view-models. A PR number is
// unique within its repository, so it identifies the row.
func (pr *GithubPullRequest) ID() string {
	return strconv.Itoa(pr.Number)
}

// URN satisfies types.HasUrn (used for selection tracking in list contexts).
func (pr *GithubPullRequest) URN() string {
	return "pull_request-" + pr.ID()
}

func (pr *GithubPullRequest) UserName() string {
	// e.g. 'jesseduffield'
	return pr.HeadRepositoryOwner.Login
}

func (pr *GithubPullRequest) BranchName() string {
	// e.g. 'feature/my-feature'
	return pr.HeadRefName
}

type GithubRepositoryOwner struct {
	Login string `json:"login"`
}
