package helpers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestSideWindowNames(t *testing.T) {
	type Test struct {
		name       string
		sidePanels []config.SidePanel
		reviewMode bool
		expected   []string
	}

	tests := []Test{
		{
			name:     "default",
			expected: []string{"status", "files", "branches", "commits", "stash"},
		},
		{
			name:       "status, commits, and stash hidden",
			sidePanels: []config.SidePanel{{"files"}, {"branches"}},
			expected:   []string{"files", "branches"},
		},
		{
			name:       "review mode ignores the side panel config",
			sidePanels: []config.SidePanel{{"files"}, {"branches"}},
			reviewMode: true,
			expected:   []string{"prList", "prContent", "prActivity"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conf := config.GetDefaultConfig()
			if test.sidePanels != nil {
				conf.Gui.SidePanels = test.sidePanels
			}
			assert.Equal(t, test.expected, SideWindowNames(conf, test.reviewMode))
		})
	}
}
