package gui

import (
	"errors"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/stretchr/testify/assert"
)

func TestGhIsAvailable(t *testing.T) {
	scenarios := []struct {
		name     string
		lookPath func(string) (string, error)
		expected bool
	}{
		{
			name: "gh found on PATH",
			lookPath: func(name string) (string, error) {
				assert.Equal(t, "gh", name)
				return "/usr/bin/gh", nil
			},
			expected: true,
		},
		{
			name: "gh not found on PATH",
			lookPath: func(name string) (string, error) {
				assert.Equal(t, "gh", name)
				return "", errors.New("exec: \"gh\": executable file not found in $PATH")
			},
			expected: false,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, ghIsAvailable(s.lookPath))
		})
	}
}

// TestDetectGhAvailable locks the behavior of the GH_AVAILABLE_OVERRIDE seam
// that the integration tests rely on. The real-PATH fallthrough (when the
// override is unset) is intentionally not asserted here because it depends on
// the host.
func TestDetectGhAvailable(t *testing.T) {
	t.Run("override of \"true\" forces available", func(t *testing.T) {
		t.Setenv(components.GH_AVAILABLE_OVERRIDE_ENV_VAR, "true")
		assert.True(t, detectGhAvailable())
	})

	t.Run("override of anything other than \"true\" forces unavailable", func(t *testing.T) {
		t.Setenv(components.GH_AVAILABLE_OVERRIDE_ENV_VAR, "false")
		assert.False(t, detectGhAvailable())
	})
}
