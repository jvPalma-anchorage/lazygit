package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnoughJumpKeys(t *testing.T) {
	cases := []struct {
		name           string
		jumpKeyCount   int
		sideWindowsLen int
		expected       bool
	}{
		{
			name:           "exactly enough keys (default 5 keys, 5 windows)",
			jumpKeyCount:   5,
			sideWindowsLen: 5,
			expected:       true,
		},
		{
			name:           "surplus keys when a panel is hidden (5 keys, 4 windows)",
			jumpKeyCount:   5,
			sideWindowsLen: 4,
			expected:       true,
		},
		{
			name:           "surplus keys when all optional panels are hidden (5 keys, 2 windows)",
			jumpKeyCount:   5,
			sideWindowsLen: 2,
			expected:       true,
		},
		{
			name:           "too few keys for the visible windows (4 keys, 5 windows)",
			jumpKeyCount:   4,
			sideWindowsLen: 5,
			expected:       false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, enoughJumpKeys(c.jumpKeyCount, c.sideWindowsLen))
		})
	}
}
