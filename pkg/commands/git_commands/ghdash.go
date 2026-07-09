package git_commands

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// PrSectionSpec is one tab in the PR-review workspace's PR list: a human-facing
// title plus the GitHub search qualifiers that populate it. The shape mirrors
// gh-dash's prSections config entries so an existing gh-dash setup carries over
// unchanged.
type PrSectionSpec struct {
	Title   string
	Filters string
}

// GhDashConfigPath resolves where the gh-dash config lives, using the same
// precedence gh-dash itself applies: an explicit GH_DASH_CONFIG override wins,
// then XDG_CONFIG_HOME, then the ~/.config fallback. Exported so callers can
// log which file was (or would have been) read when sections look wrong.
func GhDashConfigPath() string {
	if path := os.Getenv("GH_DASH_CONFIG"); path != "" {
		return path
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "gh-dash", "config.yml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// With no resolvable home there is no config to find; the relative path
		// simply fails to open and the caller degrades to defaults.
		return filepath.Join(".config", "gh-dash", "config.yml")
	}
	return filepath.Join(home, ".config", "gh-dash", "config.yml")
}

// ghDashConfig captures only the slice of gh-dash's config we consume; unknown
// keys are ignored so any real gh-dash config parses cleanly.
type ghDashConfig struct {
	PrSections []struct {
		Title   string `yaml:"title"`
		Filters string `yaml:"filters"`
	} `yaml:"prSections"`
}

// LoadGhDashSections reads the user's gh-dash prSections so the PR list shows
// the same tabs they already use in gh-dash. It never returns an error: a
// missing file, unparsable YAML, or an empty section list all degrade to a
// sensible default set scoped to defaultRepo, because a broken dotfile should
// not block opening the review workspace.
func LoadGhDashSections(defaultRepo string) []PrSectionSpec {
	defaults := []PrSectionSpec{
		{Title: "Open PRs", Filters: "is:open repo:" + defaultRepo},
		{Title: "Mine", Filters: "is:open author:@me repo:" + defaultRepo},
		{Title: "Review requested", Filters: "is:open review-requested:@me repo:" + defaultRepo},
	}

	content, err := os.ReadFile(GhDashConfigPath())
	if err != nil {
		return defaults
	}

	var config ghDashConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return defaults
	}

	if len(config.PrSections) == 0 {
		return defaults
	}

	sections := make([]PrSectionSpec, 0, len(config.PrSections))
	for _, section := range config.PrSections {
		sections = append(sections, PrSectionSpec{Title: section.Title, Filters: section.Filters})
	}
	return sections
}
