package git_commands

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PrListEntry is one row in the PR-review workspace's PR list, as returned by
// `gh search prs`. Owner/Repo are kept separate (rather than nameWithOwner)
// because opening a PR into review mode needs them as distinct arguments.
type PrListEntry struct {
	Owner  string
	Repo   string
	Number int
	Title  string
	State  string
	Author string
}

// ID uniquely identifies a PR across repos, satisfying the list view-model's HasID.
func (e *PrListEntry) ID() string {
	return fmt.Sprintf("%s/%s#%d", e.Owner, e.Repo, e.Number)
}

// SearchPrs runs one section's filters through `gh search prs` and returns the
// matching PRs. Like the other gh-backed reads, shelling out to gh means auth,
// host resolution, and rate limiting are all gh's problem, not ours. The
// filters string is split on whitespace into separate positional qualifiers,
// matching how gh-dash hands its section filters to the search API.
func (self *GitHubCommands) SearchPrs(filters string, defaultRepo string) ([]PrListEntry, error) {
	// gh's search endpoint is account-global; a section without a repo: or org:
	// qualifier (common in personal gh-dash configs) would search all of GitHub,
	// so we scope it to the repo lazygit is running in.
	if !strings.Contains(filters, "repo:") && !strings.Contains(filters, "org:") {
		filters += " repo:" + defaultRepo
	}

	cmdArgs := []string{
		"gh", "search", "prs",
		"--json", "number,title,state,author,repository",
		"--limit", "30",
	}
	cmdArgs = append(cmdArgs, strings.Fields(filters)...)

	stdout, stderr, err := self.cmd.New(cmdArgs).DontLog().RunWithOutputs()
	if err != nil {
		return nil, fmt.Errorf("gh search prs failed: %w: %s", err, stderr)
	}

	return parsePrSearchResults([]byte(stdout))
}

// prSearchResultItem mirrors the fields we request via --json (only the ones
// we consume).
type prSearchResultItem struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	Repository struct {
		Name          string `json:"name"`
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
}

// parsePrSearchResults maps the `gh search prs --json` array into PR list
// entries. Split out from SearchPrs so the mapping (notably deriving Owner
// from nameWithOwner) is unit-testable without a gh binary.
func parsePrSearchResults(respBytes []byte) ([]PrListEntry, error) {
	var items []prSearchResultItem
	if err := json.Unmarshal(respBytes, &items); err != nil {
		return nil, err
	}

	entries := make([]PrListEntry, 0, len(items))
	for _, item := range items {
		// nameWithOwner is "owner/name"; the owner is everything before the
		// first slash. Cut degrades gracefully if the slash is missing.
		owner, _, _ := strings.Cut(item.Repository.NameWithOwner, "/")
		entries = append(entries, PrListEntry{
			Owner:  owner,
			Repo:   item.Repository.Name,
			Number: item.Number,
			Title:  item.Title,
			State:  item.State,
			Author: item.Author.Login,
		})
	}
	return entries, nil
}
