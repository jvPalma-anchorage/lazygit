package presentation

import (
	"testing"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
	"github.com/xo/terminfo"
)

func TestGetPullRequestListDisplayStrings(t *testing.T) {
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelNone)
	defer color.ForceSetColorLevel(oldColorLevel)

	scenarios := []struct {
		name     string
		prs      []*models.GithubPullRequest
		expected [][]string
	}{
		{
			name:     "empty input renders no rows",
			prs:      []*models.GithubPullRequest{},
			expected: [][]string{},
		},
		{
			name: "renders number and title per row",
			prs: []*models.GithubPullRequest{
				{Number: 101, Title: "Add dark mode", State: "OPEN"},
				{Number: 7, Title: "Fix crash", State: "DRAFT"},
			},
			expected: [][]string{
				{"#101", "Add dark mode"},
				{"#7", "Fix crash"},
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, GetPullRequestListDisplayStrings(s.prs))
		})
	}
}
