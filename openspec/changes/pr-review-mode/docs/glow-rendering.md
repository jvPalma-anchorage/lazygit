# Glow markdown rendering

Self-contained reference for rendering PR/comment markdown bodies through `glow`
with a plain-text fallback. **Verdict: feasible and clean** — every primitive
exists and is verified. `glow` 2.1.1 is installed at
`/home/linuxbrew/.linuxbrew/bin/glow`.

## Flow

1. Detect `glow` once at startup (cached bool, mirroring `pkg/gui/gh.go`).
2. Build `glow` via the argv command builder (NEVER `NewShell`).
3. Pipe the markdown via `SetStdin`; capture stdout.
4. Feed the raw ANSI string into the existing `NewRenderStringTask` path — the
   same path git's colored diff already uses (gocui runs `OutputTrue`,
   gui.go:833, and its `escapeInterpreter` parses 256/truecolor ANSI).
5. Fallback when glow is absent/errored/empty: render the raw markdown unchanged.

## Detection gate (new file pkg/gui/glow.go)

Copy `pkg/gui/gh.go:19-32`. Add a `glowAvailable bool` Gui field set once at
startup (same place `ghAvailable` is initialized — do NOT `LookPath` on hot render
paths). Split into `detectGlowAvailable()` (honors a `GLOW_AVAILABLE_OVERRIDE` env
var for deterministic integration tests, like `GH_AVAILABLE_OVERRIDE_ENV_VAR`) and
a unit-testable core:

```go
func glowIsAvailable(lookPath func(string) (string, error)) bool {
    _, err := lookPath("glow")
    return err == nil
}
```

## Invocation (the security-critical contract)

```go
func (gui *Gui) renderMarkdown(body string, width int) string {
    if !gui.glowAvailable || strings.TrimSpace(body) == "" {
        return body
    }
    out, _, err := gui.os.Cmd.
        New([]string{"glow", "-s", "dark", "-w", strconv.Itoa(width)}).
        AddEnvVars("CLICOLOR_FORCE=1").
        SetStdin(body).
        DontLog().
        RunWithOutputs()
    if err != nil || strings.TrimSpace(out) == "" {
        return body
    }
    return out
}
```

Callers ALWAYS call `renderMarkdown()` and ALWAYS get a renderable string — zero
code-path divergence at the call sites; the feature degrades to plain markdown.

## Three grounded corrections (verified on glow 2.1.1)

### 1. Use `-s dark`/`-s light`, NOT `-s auto`

`-s auto` needs a TTY to probe the terminal background; into a pipe (always the
case when we capture output) it emits ZERO ANSI — indistinguishable from the
plaintext fallback.

```sh
printf '# Hello\n\n**bold** and `code`\n' | CLICOLOR_FORCE=1 glow -s dark -w 80 | od -c
# => 033 [ 9 3 ; 1 0 4 ; 1 m ...   (ANSI present)
printf '# H\n\n**b**\n' | CLICOLOR_FORCE=1 glow -s auto -w 80 | od -c
# => '#   H' only, NO 033 escapes   (auto is broken when piped)
```

`CLICOLOR_FORCE=1` is still required to guarantee emission across glow versions
when stdout is the captured pipe.

### 2. Use `RunWithOutputs`, NOT `RunWithOutput`

`RunWithOutput` calls `CombinedOutput()` (cmd_obj_runner.go:108), which would
splice any glow stderr warning INTO the rendered markdown shown to the user.
`RunWithOutputs` keeps stdout/stderr separate — take stdout only.

### 3. Glow spawn cost (~tens of ms)

Do NOT run glow on every redraw/resize on the gocui main loop. Either render off
the main thread (`OnWorker` → `OnUIThread`) or memoize keyed by `(body, width)`
and only re-render on selection/width change.

## Security

The safe path (`New([]string{...})`, exec.Command, no shell, cmd_obj_builder.go:38)
and the dangerous path (`NewShell` → `sh -c`, cmd_obj_builder.go:47) are different
methods on the same builder. The markdown MUST go via `SetStdin`, never
interpolated into a shell string — otherwise a backtick/`$()`/`;` in an
attacker-controlled comment body would execute. Add a test asserting the glow
argv contains no interpolated body text.

## Embedding the output

No special handling: gocui parses ANSI on write (`SetContent` → `writeString` →
`parseInput` → `escapeInterpreter.parseOne`), handling SGR 256-color and truecolor
(`38;2;r;g;b`). Use the standard `NewRenderStringTask(renderedOrPlain)` into the
PR-review view — identical to how the colored git diff is already shown. Glow pads
lines to `-w` width with trailing spaces; harmless in a view.

## Width

Recompute `width := view.InnerWidth()` per (re)render; re-run `renderMarkdown` on
resize, because glow hard-wraps at `-w` at generation time (the wrap is baked into
the captured string, unlike a live pager). Apply a minimum `-w` floor (e.g. 20) so
very narrow panels don't produce degenerate wrapping.

## Open knobs (deferred)

- Style: `-s dark` hardcoded for v1 (lazygit exposes no "terminal is light"
  signal to glow). A `git.pullRequests.glowStyle` config knob is a later nicety.
- A config flag to disable glow even when present (accessibility / raw-markdown
  preference).
