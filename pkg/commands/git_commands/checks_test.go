package git_commands

import (
	"errors"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

const samplePrChecksJSON = `[
	{
		"workflow": "CI",
		"name": "build",
		"bucket": "pass",
		"state": "SUCCESS",
		"link": "https://github.com/owner/repo/actions/runs/111/job/222",
		"startedAt": "2026-06-01T10:00:00Z",
		"completedAt": "2026-06-01T10:05:00Z"
	},
	{
		"workflow": "CI",
		"name": "test",
		"bucket": "fail",
		"state": "FAILURE",
		"link": "https://github.com/owner/repo/actions/runs/111/job/333",
		"startedAt": "2026-06-01T10:00:00Z",
		"completedAt": "2026-06-01T10:10:00Z"
	},
	{
		"workflow": "Lint",
		"name": "golangci-lint",
		"bucket": "pending",
		"state": "IN_PROGRESS",
		"link": "https://github.com/owner/repo/actions/runs/444",
		"startedAt": "2026-06-01T10:01:00Z",
		"completedAt": ""
	}
]`

func TestParsePrChecks(t *testing.T) {
	checks, err := parsePrChecks([]byte(samplePrChecksJSON))
	assert.NoError(t, err)
	assert.Len(t, checks, 3)

	assert.Equal(t, PrCheck{
		Workflow:    "CI",
		Name:        "build",
		Bucket:      "pass",
		State:       "SUCCESS",
		Link:        "https://github.com/owner/repo/actions/runs/111/job/222",
		StartedAt:   "2026-06-01T10:00:00Z",
		CompletedAt: "2026-06-01T10:05:00Z",
	}, checks[0])
	assert.Equal(t, PrCheck{
		Workflow:    "CI",
		Name:        "test",
		Bucket:      "fail",
		State:       "FAILURE",
		Link:        "https://github.com/owner/repo/actions/runs/111/job/333",
		StartedAt:   "2026-06-01T10:00:00Z",
		CompletedAt: "2026-06-01T10:10:00Z",
	}, checks[1])
	assert.Equal(t, PrCheck{
		Workflow:    "Lint",
		Name:        "golangci-lint",
		Bucket:      "pending",
		State:       "IN_PROGRESS",
		Link:        "https://github.com/owner/repo/actions/runs/444",
		StartedAt:   "2026-06-01T10:01:00Z",
		CompletedAt: "",
	}, checks[2])
}

func TestParsePrChecksMalformedJSON(t *testing.T) {
	_, err := parsePrChecks([]byte("gh: not logged in"))
	assert.Error(t, err)
}

func TestSortChecksByImportance(t *testing.T) {
	checks := []PrCheck{
		{Name: "skipped", Bucket: "skipping", CompletedAt: "2026-06-01T12:00:00Z"},
		{Name: "fail-older", Bucket: "fail", CompletedAt: "2026-06-01T10:00:00Z"},
		{Name: "pass", Bucket: "pass", CompletedAt: "2026-06-01T11:00:00Z"},
		{Name: "pending", Bucket: "pending", StartedAt: "2026-06-01T10:30:00Z"},
		{Name: "fail-newest", Bucket: "fail", CompletedAt: "2026-06-01T10:45:00Z"},
		{Name: "cancel", Bucket: "cancel", CompletedAt: "2026-06-01T10:15:00Z"},
	}

	SortChecksByImportance(checks)

	names := make([]string, 0, len(checks))
	for _, check := range checks {
		names = append(names, check.Name)
	}
	assert.Equal(t, []string{
		"fail-newest",
		"fail-older",
		"cancel",
		"pending",
		"pass",
		"skipped",
	}, names)
}

func TestCheckRunJobIDs(t *testing.T) {
	runID, jobID, ok := CheckRunJobIDs("https://github.com/owner/repo/actions/runs/123456/job/7891011")
	assert.True(t, ok)
	assert.Equal(t, "123456", runID)
	assert.Equal(t, "7891011", jobID)

	runID, jobID, ok = CheckRunJobIDs("https://github.com/owner/repo/actions/runs/123456")
	assert.True(t, ok)
	assert.Equal(t, "123456", runID)
	assert.Equal(t, "", jobID)

	_, _, ok = CheckRunJobIDs("https://ci.example.com/builds/42")
	assert.False(t, ok)
}

func TestFetchPRChecks(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).
		ExpectArgs([]string{"gh", "pr", "checks", "42", "--repo", "own/rep", "--json", "workflow,name,bucket,state,link,startedAt,completedAt"}, samplePrChecksJSON, nil)
	instance := buildGitHubCommands(commonDeps{runner: runner})

	checks, err := instance.FetchPRChecks("own", "rep", 42)
	assert.NoError(t, err)
	assert.Len(t, checks, 3)
	assert.Equal(t, "build", checks[0].Name)
	runner.CheckForMissingCalls()
}

func TestFetchPRChecksIgnoresExitErrorWhenOutputParses(t *testing.T) {
	// gh pr checks exits non-zero whenever any check is failing, even though it
	// printed a valid JSON payload; that must not be treated as a failure.
	runner := oscommands.NewFakeRunner(t).
		ExpectArgs([]string{"gh", "pr", "checks", "42", "--repo", "own/rep", "--json", "workflow,name,bucket,state,link,startedAt,completedAt"}, samplePrChecksJSON, errors.New("exit status 1"))
	instance := buildGitHubCommands(commonDeps{runner: runner})

	checks, err := instance.FetchPRChecks("own", "rep", 42)
	assert.NoError(t, err)
	assert.Len(t, checks, 3)
	runner.CheckForMissingCalls()
}

func TestFetchPRChecksReportsErrorWhenOutputUnparseable(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).
		ExpectArgs([]string{"gh", "pr", "checks", "42", "--repo", "own/rep", "--json", "workflow,name,bucket,state,link,startedAt,completedAt"}, "no pull requests found", errors.New("exit status 1"))
	instance := buildGitHubCommands(commonDeps{runner: runner})

	_, err := instance.FetchPRChecks("own", "rep", 42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gh pr checks failed")
	runner.CheckForMissingCalls()
}

func TestChecksLogCmdObj(t *testing.T) {
	instance := buildGitHubCommands(commonDeps{})

	withJob := instance.ChecksLogCmdObj("own", "rep", "123456", "7891011")
	assert.Equal(t, []string{"gh", "run", "view", "123456", "--repo", "own/rep", "--job", "7891011", "--log"}, withJob.Args())
	assert.Contains(t, withJob.GetEnvVars(), "FORCE_COLOR=1")

	withoutJob := instance.ChecksLogCmdObj("own", "rep", "123456", "")
	assert.Equal(t, []string{"gh", "run", "view", "123456", "--repo", "own/rep", "--log"}, withoutJob.Args())
	assert.Contains(t, withoutJob.GetEnvVars(), "FORCE_COLOR=1")
}
