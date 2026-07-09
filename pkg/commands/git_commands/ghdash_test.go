package git_commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func TestLoadGhDashSectionsFromConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yml")
	assert.NoError(t, os.WriteFile(configPath, []byte(`prSections:
  - title: Team PRs
    filters: is:open org:acme
  - title: Needs my review
    filters: is:open review-requested:@me
`), 0o600))
	t.Setenv("GH_DASH_CONFIG", configPath)

	sections := LoadGhDashSections("own/rep")

	assert.Equal(t, []PrSectionSpec{
		{Title: "Team PRs", Filters: "is:open org:acme"},
		{Title: "Needs my review", Filters: "is:open review-requested:@me"},
	}, sections)
}

func TestLoadGhDashSectionsMissingFileReturnsDefaults(t *testing.T) {
	t.Setenv("GH_DASH_CONFIG", filepath.Join(t.TempDir(), "does-not-exist.yml"))

	sections := LoadGhDashSections("own/rep")

	assert.Len(t, sections, 3)
	assert.Equal(t, "Open PRs", sections[0].Title)
	assert.Equal(t, "Mine", sections[1].Title)
	assert.Equal(t, "Review requested", sections[2].Title)
	for _, section := range sections {
		assert.Contains(t, section.Filters, "repo:own/rep")
	}
}

func TestLoadGhDashSectionsCorruptYamlReturnsDefaults(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yml")
	assert.NoError(t, os.WriteFile(configPath, []byte("prSections: [not: {valid"), 0o600))
	t.Setenv("GH_DASH_CONFIG", configPath)

	sections := LoadGhDashSections("own/rep")

	assert.Len(t, sections, 3)
	assert.Equal(t, "Open PRs", sections[0].Title)
}

func TestParsePrSearchResults(t *testing.T) {
	sample := `[
	  {"number": 12, "title": "Fix the thing", "state": "open",
	   "author": {"login": "alice"},
	   "repository": {"name": "rep", "nameWithOwner": "own/rep"}},
	  {"number": 7, "title": "Add stuff", "state": "closed",
	   "author": {"login": "bob"},
	   "repository": {"name": "thing", "nameWithOwner": "other/thing"}}
	]`

	entries, err := parsePrSearchResults([]byte(sample))
	assert.NoError(t, err)

	assert.Equal(t, []PrListEntry{
		{Owner: "own", Repo: "rep", Number: 12, Title: "Fix the thing", State: "open", Author: "alice"},
		{Owner: "other", Repo: "thing", Number: 7, Title: "Add stuff", State: "closed", Author: "bob"},
	}, entries)
}

func TestParsePrSearchResultsMalformedJson(t *testing.T) {
	_, err := parsePrSearchResults([]byte(`{"not": "an array"`))
	assert.Error(t, err)
}

func TestSearchPrsAppendsRepoQualifierWhenMissing(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).
		ExpectArgs([]string{"gh", "search", "prs", "--json", "number,title,state,author,repository", "--limit", "30", "is:open", "repo:own/rep"}, `[]`, nil)
	instance := buildGitHubCommands(commonDeps{runner: runner})

	entries, err := instance.SearchPrs("is:open", "own/rep")
	assert.NoError(t, err)
	assert.Empty(t, entries)
	runner.CheckForMissingCalls()
}

func TestSearchPrsKeepsExistingRepoQualifier(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).
		ExpectArgs([]string{"gh", "search", "prs", "--json", "number,title,state,author,repository", "--limit", "30", "is:open", "repo:other/thing"}, `[]`, nil)
	instance := buildGitHubCommands(commonDeps{runner: runner})

	entries, err := instance.SearchPrs("is:open repo:other/thing", "own/rep")
	assert.NoError(t, err)
	assert.Empty(t, entries)
	runner.CheckForMissingCalls()
}
