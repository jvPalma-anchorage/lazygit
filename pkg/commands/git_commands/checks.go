package git_commands

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// PrCheck is one CI check on a pull request as reported by `gh pr checks`.
// Bucket is gh's normalized rollup of the check's status/conclusion (fail,
// cancel, pending, pass, skipping) and is what the Checks tab keys its
// ordering and styling off; State carries the raw GitHub state for display.
// StartedAt/CompletedAt are RFC3339 strings straight from the API (empty when
// the check hasn't started/finished), kept as strings because RFC3339 compares
// correctly lexicographically and we only ever sort/display them.
type PrCheck struct {
	Workflow    string
	Name        string
	Bucket      string
	State       string
	Link        string
	StartedAt   string
	CompletedAt string
}

// FetchPRChecks lists the CI checks for a pull request via `gh pr checks`.
// Note that gh deliberately exits non-zero when any check is failing or still
// pending, even though it printed a perfectly valid JSON payload — so we parse
// stdout first and only surface the error when there is nothing parseable
// (auth failure, no such PR, etc.).
func (self *GitHubCommands) FetchPRChecks(owner string, repo string, number int) ([]PrCheck, error) {
	cmdArgs := []string{
		"gh", "pr", "checks", strconv.Itoa(number),
		// Explicit --repo: the workspace can be retargeted at a PR in a different
		// repository than the local checkout, and gh would otherwise resolve the
		// number against the checkout's repo (codex review finding).
		"--repo", owner + "/" + repo,
		"--json", "workflow,name,bucket,state,link,startedAt,completedAt",
	}

	stdout, stderr, err := self.cmd.New(cmdArgs).DontLog().RunWithOutputs()

	checks, parseErr := parsePrChecks([]byte(stdout))
	if parseErr == nil {
		return checks, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gh pr checks failed: %w: %s", err, stderr)
	}
	return nil, parseErr
}

func parsePrChecks(respBytes []byte) ([]PrCheck, error) {
	var raw []struct {
		Workflow    string `json:"workflow"`
		Name        string `json:"name"`
		Bucket      string `json:"bucket"`
		State       string `json:"state"`
		Link        string `json:"link"`
		StartedAt   string `json:"startedAt"`
		CompletedAt string `json:"completedAt"`
	}
	if err := json.Unmarshal(respBytes, &raw); err != nil {
		return nil, err
	}

	checks := make([]PrCheck, 0, len(raw))
	for _, c := range raw {
		checks = append(checks, PrCheck{
			Workflow:    c.Workflow,
			Name:        c.Name,
			Bucket:      c.Bucket,
			State:       c.State,
			Link:        c.Link,
			StartedAt:   c.StartedAt,
			CompletedAt: c.CompletedAt,
		})
	}
	return checks, nil
}

// bucketRank orders buckets by how urgently the user needs to see them:
// failures first, then cancellations, then still-running checks, then
// successes, with skipped checks always last. An unrecognized bucket (gh may
// grow new ones) slots in just above "pass" so it stays visible without
// displacing actionable failures.
func bucketRank(bucket string) int {
	switch bucket {
	case "fail":
		return 0
	case "cancel":
		return 1
	case "pending":
		return 2
	case "pass":
		return 4
	case "skipping":
		return 5
	default:
		return 3
	}
}

// checkSortTime is the timestamp used for intra-bucket ordering: when a check
// finished, falling back to when it started (a pending check has no
// CompletedAt yet).
func checkSortTime(check PrCheck) string {
	if check.CompletedAt != "" {
		return check.CompletedAt
	}
	return check.StartedAt
}

// SortChecksByImportance sorts the checks in place so the Checks tab surfaces
// what matters: failing checks first, skipped ones last (see bucketRank), and
// within a bucket the most recently completed check first. The sort is stable
// so checks that tie on bucket and timestamp keep gh's original order.
func SortChecksByImportance(checks []PrCheck) {
	sort.SliceStable(checks, func(i, j int) bool {
		rankI, rankJ := bucketRank(checks[i].Bucket), bucketRank(checks[j].Bucket)
		if rankI != rankJ {
			return rankI < rankJ
		}
		// RFC3339 strings compare correctly lexicographically; descending puts
		// the most recent activity first.
		return checkSortTime(checks[i]) > checkSortTime(checks[j])
	})
}

var actionsRunLinkPattern = regexp.MustCompile(`^https://[^/]+/[^/]+/[^/]+/actions/runs/(\d+)(?:/job/(\d+))?`)

// CheckRunJobIDs extracts the workflow run ID (and job ID, when present) from
// a check's details link, e.g.
// https://github.com/OWNER/REPO/actions/runs/123456/job/7891011.
// ok is false for links that don't point at a GitHub Actions run — external CI
// systems put arbitrary URLs here, and those checks have no gh-viewable log.
func CheckRunJobIDs(link string) (string, string, bool) {
	match := actionsRunLinkPattern.FindStringSubmatch(link)
	if match == nil {
		return "", "", false
	}
	return match[1], match[2], true
}

// ChecksLogCmdObj builds the `gh run view --log` command for a check's
// workflow run (scoped to a single job when jobID is given). The caller
// renders it through a pty task, so FORCE_COLOR=1 asks gh to keep its ANSI
// coloring despite stdout not being a terminal on gh's side of the pipe.
func (self *GitHubCommands) ChecksLogCmdObj(owner string, repo string, runID string, jobID string) *oscommands.CmdObj {
	cmdArgs := []string{"gh", "run", "view", runID, "--repo", owner + "/" + repo}
	if jobID != "" {
		cmdArgs = append(cmdArgs, "--job", jobID)
	}
	cmdArgs = append(cmdArgs, "--log")

	return self.cmd.New(cmdArgs).AddEnvVars("FORCE_COLOR=1").DontLog()
}
