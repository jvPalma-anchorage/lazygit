package gui

import (
	"errors"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/stretchr/testify/assert"
)

func TestGlowIsAvailable(t *testing.T) {
	scenarios := []struct {
		name     string
		lookPath func(string) (string, error)
		expected bool
	}{
		{
			name: "glow found on PATH",
			lookPath: func(name string) (string, error) {
				assert.Equal(t, "glow", name)
				return "/usr/bin/glow", nil
			},
			expected: true,
		},
		{
			name: "glow not found on PATH",
			lookPath: func(name string) (string, error) {
				assert.Equal(t, "glow", name)
				return "", errors.New("exec: \"glow\": executable file not found in $PATH")
			},
			expected: false,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, glowIsAvailable(s.lookPath))
		})
	}
}

// TestDetectGlowAvailable locks the behavior of the GLOW_AVAILABLE_OVERRIDE seam
// that the integration tests rely on. The real-PATH fallthrough (when the
// override is unset) is intentionally not asserted here because it depends on
// the host.
func TestDetectGlowAvailable(t *testing.T) {
	t.Run("override of \"true\" forces available", func(t *testing.T) {
		t.Setenv(components.GLOW_AVAILABLE_OVERRIDE_ENV_VAR, "true")
		assert.True(t, detectGlowAvailable())
	})

	t.Run("override of anything other than \"true\" forces unavailable", func(t *testing.T) {
		t.Setenv(components.GLOW_AVAILABLE_OVERRIDE_ENV_VAR, "false")
		assert.False(t, detectGlowAvailable())
	})
}

func TestGlowRenderMarkdownFallsBackWhenUnavailable(t *testing.T) {
	gui := &Gui{glowAvailable: false}
	body := "# Heading\n\nsome **markdown**"
	// glowAvailable is false, so no command is run and body is returned verbatim.
	assert.Equal(t, body, gui.renderMarkdown(body, 80))
}

func TestGlowRenderMarkdownReturnsBlankBodyUnchanged(t *testing.T) {
	gui := &Gui{glowAvailable: true}
	// Even with glow available, a blank body short-circuits before spawning glow,
	// so gui.os may be nil here without panicking.
	assert.Equal(t, "   \n", gui.renderMarkdown("   \n", 80))
}

// TestGlowRenderMarkdownArgvCarriesNoBody is the security assertion: the
// (attacker-controlled) markdown body must travel only via stdin, never
// interpolated into the glow argv or env vars, so shell metacharacters in a
// comment body can't become exec/shell input.
func TestGlowRenderMarkdownArgvCarriesNoBody(t *testing.T) {
	const body = "# pwned `rm -rf /` $(touch /tmp/x); && | DANGEROUS_SENTINEL"

	runner := oscommands.NewFakeRunner(t)
	runner.ExpectFunc("glow renders markdown without leaking body into argv", func(cmdObj *oscommands.CmdObj) bool {
		args := cmdObj.Args()
		assert.Equal(t, []string{"glow", "-s", "dark", "-w", "80"}, args)
		for _, arg := range args {
			assert.NotContains(t, arg, "DANGEROUS_SENTINEL")
			assert.NotContains(t, arg, body)
		}
		// The body must not be smuggled in via env vars either.
		for _, envVar := range cmdObj.GetEnvVars() {
			assert.NotContains(t, envVar, "DANGEROUS_SENTINEL")
		}
		assert.Contains(t, cmdObj.GetEnvVars(), "CLICOLOR_FORCE=1")
		return true
	}, "<rendered>", nil)

	gui := &Gui{
		glowAvailable: true,
		os:            oscommands.NewDummyOSCommandWithRunner(runner),
	}

	out := gui.renderMarkdown(body, 80)
	assert.Equal(t, "<rendered>", out)
	runner.CheckForMissingCalls()
}

func TestGlowRenderMarkdownMemoizesPerBodyAndWidth(t *testing.T) {
	const body = "# memoized"

	runner := oscommands.NewFakeRunner(t)
	// Expect glow to be spawned exactly once even though renderMarkdown is called
	// twice for the same (body, width).
	runner.ExpectFunc("glow spawned once", func(cmdObj *oscommands.CmdObj) bool {
		return strings.HasPrefix(cmdObj.Args()[0], "glow")
	}, "<rendered>", nil)

	gui := &Gui{
		glowAvailable: true,
		os:            oscommands.NewDummyOSCommandWithRunner(runner),
	}

	first := gui.renderMarkdown(body, 80)
	second := gui.renderMarkdown(body, 80)
	assert.Equal(t, "<rendered>", first)
	assert.Equal(t, first, second)
	runner.CheckForMissingCalls()
}
