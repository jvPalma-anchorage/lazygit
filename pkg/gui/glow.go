package gui

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/integration/components"
)

// detectGlowAvailable reports whether the glow markdown renderer is on PATH.
// Call this once at startup and cache the result (see Gui.glowAvailable); do not
// call it on hot render paths, since spawning glow costs tens of milliseconds.
//
// GLOW_AVAILABLE_OVERRIDE, if set, forces the result ("true" → available,
// anything else → unavailable), bypassing PATH detection. Integration tests use
// it to drive PR-review markdown rendering deterministically. In normal use the
// variable is unset, so this is just exec.LookPath.
func detectGlowAvailable() bool {
	if override, ok := os.LookupEnv(components.GLOW_AVAILABLE_OVERRIDE_ENV_VAR); ok {
		return override == "true"
	}
	return glowIsAvailable(exec.LookPath)
}

// glowIsAvailable is the testable core of detectGlowAvailable: it takes the
// lookPath function as a parameter so tests can stub the lookup instead of
// depending on the host's PATH.
func glowIsAvailable(lookPath func(string) (string, error)) bool {
	_, err := lookPath("glow")
	return err == nil
}

// glowMinWidth is a floor for the glow wrap width so that very narrow panels
// don't produce degenerate wrapping.
const glowMinWidth = 20

// renderMarkdown renders a markdown body through glow, returning ANSI-styled
// text. It returns body unchanged when glow is unavailable or body is blank, and
// falls back to body on any error or empty output, so callers always get a
// renderable string.
//
// The body bytes travel only via stdin (SetStdin), never interpolated into the
// argv, so an attacker-controlled comment body can't inject shell/exec input.
// Glow is spawned via the argv builder (New, never NewShell).
//
// Output is memoized keyed by (body, width); glow is only spawned on the first
// request for a given pair (e.g. on selection or width change), keeping it off
// the gocui main loop's hot path.
func (gui *Gui) renderMarkdown(body string, width int) string {
	if !gui.glowAvailable || strings.TrimSpace(body) == "" {
		return body
	}

	if width < glowMinWidth {
		width = glowMinWidth
	}

	key := glowCacheKey{body: body, width: width}
	gui.glowCacheMutex.Lock()
	if gui.glowCache == nil {
		gui.glowCache = map[glowCacheKey]string{}
	}
	if cached, ok := gui.glowCache[key]; ok {
		gui.glowCacheMutex.Unlock()
		return cached
	}
	gui.glowCacheMutex.Unlock()

	rendered := body
	out, _, err := gui.os.Cmd.
		New([]string{"glow", "-s", "dark", "-w", strconv.Itoa(width)}).
		AddEnvVars("CLICOLOR_FORCE=1").
		SetStdin(body).
		DontLog().
		RunWithOutputs()
	if err == nil && strings.TrimSpace(out) != "" {
		rendered = out
	}

	gui.glowCacheMutex.Lock()
	gui.glowCache[key] = rendered
	gui.glowCacheMutex.Unlock()

	return rendered
}

// glowCacheKey memoizes rendered markdown so glow is only spawned once per
// (body, width) pair.
type glowCacheKey struct {
	body  string
	width int
}
