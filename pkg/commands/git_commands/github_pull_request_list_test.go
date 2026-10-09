package git_commands

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func TestListOpenPullRequests(t *testing.T) {
	t.Setenv("GH_PATH", "gh")
	runner := oscommands.NewFakeRunner(t).
		ExpectArgs([]string{
			"gh", "pr", "list", "--state", "open", "--limit", "30",
			"--json", "number,title,state,isDraft,headRefName,url,headRepositoryOwner",
		}, `[
			{"number": 7, "title": "Add feature", "state": "OPEN", "isDraft": false, "headRefName": "feature", "url": "https://github.com/own/rep/pull/7", "headRepositoryOwner": {"login": "alice"}},
			{"number": 8, "title": "WIP", "state": "OPEN", "isDraft": true, "headRefName": "wip", "url": "https://github.com/own/rep/pull/8", "headRepositoryOwner": {"login": "bob"}}
		]`, nil)
	instance := buildGitHubCommands(commonDeps{runner: runner})

	prs, err := instance.ListOpenPullRequests()
	assert.NoError(t, err)
	assert.Equal(t, []*models.GithubPullRequest{
		{Number: 7, Title: "Add feature", State: "OPEN", HeadRefName: "feature", Url: "https://github.com/own/rep/pull/7", HeadRepositoryOwner: models.GithubRepositoryOwner{Login: "alice"}},
		{Number: 8, Title: "WIP", State: "DRAFT", HeadRefName: "wip", Url: "https://github.com/own/rep/pull/8", HeadRepositoryOwner: models.GithubRepositoryOwner{Login: "bob"}},
	}, prs)
	runner.CheckForMissingCalls()
}

func TestParseOpenPullRequestsMalformedJson(t *testing.T) {
	_, err := parseOpenPullRequests([]byte(`not json`))
	assert.Error(t, err)
}
