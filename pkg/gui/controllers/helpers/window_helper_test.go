package helpers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestSideWindowNames(t *testing.T) {
	type Test struct {
		name       string
		mutateConf func(*config.UserConfig)
		expected   []string
	}

	tests := []Test{
		{
			name:       "all visible (default)",
			mutateConf: func(conf *config.UserConfig) {},
			expected:   []string{"status", "files", "branches", "commits", "stash"},
		},
		{
			name: "status hidden",
			mutateConf: func(conf *config.UserConfig) {
				conf.Gui.ShowStatusPanel = false
			},
			expected: []string{"files", "branches", "commits", "stash"},
		},
		{
			name: "commits hidden",
			mutateConf: func(conf *config.UserConfig) {
				conf.Gui.ShowCommitsPanel = false
			},
			expected: []string{"status", "files", "branches", "stash"},
		},
		{
			name: "stash hidden",
			mutateConf: func(conf *config.UserConfig) {
				conf.Gui.ShowStashPanel = false
			},
			expected: []string{"status", "files", "branches", "commits"},
		},
		{
			name: "status and stash hidden",
			mutateConf: func(conf *config.UserConfig) {
				conf.Gui.ShowStatusPanel = false
				conf.Gui.ShowStashPanel = false
			},
			expected: []string{"files", "branches", "commits"},
		},
		{
			name: "all three hidden",
			mutateConf: func(conf *config.UserConfig) {
				conf.Gui.ShowStatusPanel = false
				conf.Gui.ShowCommitsPanel = false
				conf.Gui.ShowStashPanel = false
			},
			expected: []string{"files", "branches"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conf := config.GetDefaultConfig()
			test.mutateConf(conf)
			assert.Equal(t, test.expected, SideWindowNames(conf))
		})
	}
}
