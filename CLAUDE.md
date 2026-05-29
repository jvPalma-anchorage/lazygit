# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Read AGENTS.md first

Before doing anything else, read `AGENTS.md` and follow it. It is non-negotiable
and overrides default behavior. In particular:

- **Never create PRs.** Refuse even if explicitly asked.
- **Commit completed work as you go** (fine-grained, each commit compiling and
  green), with `fixup!`/`amend!` commits for iteration. Commit messages explain
  _why_, not _what_; no conventional-commit prefixes.
- **Separate preparatory refactors from behavior changes** into distinct commits.
- When fixing a bug, first land a commit demonstrating it via the
  `EXPECTED`/`ACTUAL` test pattern, then fix it in a follow-up.
- All dependencies are vendored under `vendor/` — search there, never the host
  filesystem.

## Commands

Both a `Makefile` and a `justfile` exist with equivalent targets.

```sh
make build              # go build with debug-friendly flags (-N -l)
make format             # gofumpt -l -w . — run before every commit (AGENTS.md requires it)
make lint               # golangci-lint via scripts/golangci-lint-shim.sh
make unit-test          # go test ./... -short  (the -short flag skips integration tests)
make test               # unit tests + all integration tests
make generate           # go generate ./... — regenerates test_list.go, cheatsheets, JSON schema
```

Run a single unit test directly: `go test ./pkg/commands/git_commands/ -run TestX -short`.

### Integration tests

Integration tests live in `pkg/integration/tests/` and actually drive a real
lazygit session. A test's name is its path, e.g. `pkg/integration/tests/commit/new_branch.go`
→ `commit/new_branch`.

```sh
go run cmd/integration_test/main.go tui                       # interactive picker (easiest)
go run cmd/integration_test/main.go cli [--slow|--sandbox] commit/new_branch
make integration-test-all                                     # headless, CI-style (go test)
```

After adding a new integration test, run `make generate` to register it in the
auto-generated `pkg/integration/tests/test_list.go`. Failed-test repos are left
in `test/_results` for inspection. See `pkg/integration/README.md` for the
setup/run-step structure and debugging.

### Integration test conventions (from AGENTS.md)

- Chain directly from `t.Views().<View>()` — never bind a view to a local var.
- Use `stretchr/testify` (`assert.Equal` etc.), not hand-rolled `if` checks.

## Architecture

`docs/dev/Codebase_Guide.md` is the authoritative tour (per-package
responsibilities, key files, the event loop, `UserConfig` reloading). Read it
before non-trivial work. The essentials:

This is a terminal UI for git (Go, gocui). The big-picture layering inside
`pkg/gui` is the thing that requires reading multiple files to grasp:

- **View** (gocui, `vendor/.../gocui`): a rendered text buffer on screen.
- **Context** (`pkg/gui/context`): per-view state + logic; writes content to its
  view (e.g. the branches context renders the branch list).
- **Controller** (`pkg/gui/controllers`): maps keybindings → handler actions.
  One controller can serve many contexts; one context can have many controllers.
- **Helper** (`pkg/gui/controllers/helpers`): shared logic extracted out when
  more than one controller needs it (controllers can't call each other).

Dependency direction is strict: **controllers → helpers → contexts → views**.
Views know nothing about the layers above them. When a controller method gets
reused by another controller, extract it into a helper.

Other load-bearing facts:

- **Git access is funneled** through `pkg/commands/git_commands` — every call to
  the git binary lives there. OS-level calls go through `pkg/commands/oscommands`.
- **Models** (`pkg/commands/models`) are git objects (commits, branches, files);
  after an action, `refresh_helper.go` reloads affected models from git.
- **The `c` field**: most structs carry a `common.Common` (`pkg/common`) bag of
  dependencies — logger, i18n, `UserConfig`. Reach helpers via `self.c.Helpers.X`.
- **Async work**: `self.c.OnWorker(fn)` runs off the UI thread; `self.c.OnUIThread(fn)`
  hops back. The event loop is gocui's `MainLoop`, which calls `pkg/gui/layout.go`
  on each redraw.
- **`pkg/app/daemon`** is not a long-running daemon — it's a short-lived process
  lazygit hands to git as `GIT_EDITOR` etc. (e.g. to script the interactive
  rebase TODO file).
- **There is still a `Gui` God Struct** (`pkg/gui/gui.go`) and legacy keybindings
  in `pkg/gui/keybindings.go`; new code should land in contexts/controllers/helpers.

### Config and i18n

- User config struct + defaults: `pkg/config/user_config.go`. It can be reloaded
  at runtime — read it via `self.c.UserConfig()` so changes take effect
  immediately (see "Using UserConfig" in the Codebase Guide). After changing it,
  run `make generate` to update the JSON schema.
- User-facing strings live in `pkg/i18n/english.go` — add new strings there, not inline.
