package git_commands

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
)

// ListOpenPullRequests returns the open pull requests of the repo lazygit is
// running in, newest first. gh resolves the repo from the remotes itself, and
// handles auth and the GitHub host.
func (self *GitHubCommands) ListOpenPullRequests() ([]*models.GithubPullRequest, error) {
	ghExe := ghExecutable()
	if ghExe == "" {
		return nil, errors.New("gh not found")
	}

	cmdArgs := []string{
		ghExe, "pr", "list",
		"--state", "open",
		"--limit", "30",
		"--json", "number,title,state,isDraft,headRefName,url,headRepositoryOwner",
	}
	stdout, stderr, err := self.cmd.New(cmdArgs).DontLog().RunWithOutputs()
	if err != nil {
		return nil, fmt.Errorf("gh pr list failed: %w: %s", err, stderr)
	}

	return parseOpenPullRequests([]byte(stdout))
}

type openPullRequestItem struct {
	models.GithubPullRequest
	IsDraft bool `json:"isDraft"`
}

// parseOpenPullRequests maps the `gh pr list --json` array to pull requests.
// gh reports a draft as state OPEN plus isDraft; we fold that into the DRAFT
// state so it gets the draft color.
func parseOpenPullRequests(respBytes []byte) ([]*models.GithubPullRequest, error) {
	var items []openPullRequestItem
	if err := json.Unmarshal(respBytes, &items); err != nil {
		return nil, err
	}

	prs := make([]*models.GithubPullRequest, 0, len(items))
	for _, item := range items {
		pr := item.GithubPullRequest
		if item.IsDraft {
			pr.State = "DRAFT"
		}
		prs = append(prs, &pr)
	}
	return prs, nil
}
