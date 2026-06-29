package gui

import (
	"os"
	"os/exec"

	"github.com/jesseduffield/lazygit/pkg/integration/components"
)

// detectGhAvailable reports whether the GitHub CLI (`gh`) is on PATH. Call this
// once at startup and cache the result (see Gui.ghAvailable); do not call it on
// hot paths such as viewTabMap().
//
// GH_AVAILABLE_OVERRIDE, if set, forces the result ("true" → available,
// anything else → unavailable), bypassing PATH detection. Integration tests use
// it to drive the Pull Requests tab deterministically (the test runner defaults
// it to "false" so the branches window has a host-independent tab layout). In
// normal use the variable is unset, so this is just exec.LookPath.
func detectGhAvailable() bool {
	if override, ok := os.LookupEnv(components.GH_AVAILABLE_OVERRIDE_ENV_VAR); ok {
		return override == "true"
	}
	return ghIsAvailable(exec.LookPath)
}

// ghIsAvailable is the testable core of detectGhAvailable: it takes the lookPath
// function as a parameter so tests can stub the lookup instead of depending on
// the host's PATH.
func ghIsAvailable(lookPath func(string) (string, error)) bool {
	_, err := lookPath("gh")
	return err == nil
}
