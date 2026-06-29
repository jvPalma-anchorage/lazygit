# Inline comment rendering

Self-contained reference for the custom diff presenter that interleaves review
threads at their anchored lines. **Feasibility verdict: feasible for v1, but only
by writing a NEW presenter that owns its own view-line↔model map.** The existing
diff-to-main path is a dead end for interleaving, and `patch_exploring.State` is
not reusable.

## Why the existing diff path cannot carry inline comments

`newPtyTask` (pty.go:46-104) starts the diff command under a pseudo-terminal and
streams its bytes straight into the view; with no custom pager it falls back to
`newCmdTask` which still streams raw `git diff` text. Either way the view receives
opaque pre-rendered text with NO per-line model — nowhere to splice a comment
block at line N.

Conclusion: the PR-review diff view MUST NOT use `RunPtyTask`/`RunCommandTask`. It
builds its own ANSI string and uses `NewRenderStringWithoutScrollTask`
(main_panels.go:13-14 → tasks_adapter newStringTaskWithoutScroll). Precedent:
`staging_helper.go:56-114` fetches a parseable diff string then
`SetState`/`RenderToMainViews`.

## What IS reusable: the patch parser

`pkg/commands/patch` public API gives the anchor mapping with no modification:
- `patch.Parse(diff)` (parse.go:12) → `*Patch`.
- `Patch.Lines()` (patch.go:41-52) → `[]*PatchLine` (`.Kind`, `.Content`).
- `Patch.LineNumberOfLine(idx)` (patch.go:88-115) → the NEW-file line number for
  that patch-line index (counts ADDITION+CONTEXT).
- `PatchLine.Kind` ∈ `{PATCH_HEADER, HUNK_HEADER, ADDITION, DELETION, CONTEXT}`.

## What is NOT reusable: patch_exploring.State

`patch_exploring.State` (state.go) hard-assumes `viewLineIdx == patchLineIdx`
(viewLineIndices/patchLineIndices, state.go:27-30). Interleaving COMMENT rows
breaks that invariant and silently corrupts every selection/hunk/range op.
**Building a parallel selection map is mandatory, not optional** — this is the
main schedule risk; do not underestimate it.

## Anchor map (RIGHT-side, v1)

For v1, anchor ONLY RIGHT-side threads (added/context lines) — the common review
case, and it needs NO new patch-package methods. LEFT-side (deletion) anchoring
needs a new old-file-line function and is a non-goal.

```go
p := patch.Parse(diff)
for idx, line := range p.Lines() {
    if line.Kind == patch.ADDITION || line.Kind == patch.CONTEXT {
        fileLine := p.LineNumberOfLine(idx) // new-file line on RIGHT
        if th := threads.atRight(fileLine); th != nil {
            emitDiffLine(idx, line)
            emitCommentBlock(th)
            continue
        }
    }
    emitDiffLine(idx, line)
}
```

## The presenter + selection map

Mirror the colorizing loop in `pkg/commands/patch/format.go:61-109` (header bold,
hunk header cyan, additions FgGreen, deletions FgRed). While emitting, build THREE
parallel slices — this triple IS the selection map that replaces State:

```go
type rowKind int
const ( rowDiff rowKind = iota; rowComment; rowHeader )

type renderedDiff struct {
    content     string  // final ANSI buffer
    rowKind     []rowKind
    rowPatchIdx []int    // for rowDiff rows: index into patch.Lines()
}
```

Comment rows are tagged `rowComment` (non-selectable). Diff rows carry their
`rowPatchIdx` so selection can map back to file lines.

## Comment block layout (author + resolved state)

Each thread renders immediately under its anchor line:

```
┌─ @alice · RESOLVED ───────────
│ <glow-rendered body, wrapped to InnerWidth>
│
│ @bob (reply)
│ <body>
└───────────────────────────────
```

- Header line: first comment's author + badge (green `RESOLVED` / yellow
  `UNRESOLVED` from `thread.IsResolved`).
- Each comment: author label line, then body via `renderMarkdown(body, width)`
  (see glow-rendering.md). When glow is absent, raw markdown text.
- Wrap bodies to `view.InnerWidth()` (reuse `utils.WrapViewLinesToWidth`).
- All block rows tagged `rowComment` so selection skips them.

## Outdated / LEFT-side threads

Threads with no live RIGHT-side anchor (`thread.line == null`, or LEFT side in v1)
render in a per-file "Outdated / other" section at the end of the file, showing
`comments[0].diffHunk` verbatim + the comments. Without this they silently vanish.

## Width re-render and caching

- Comment prose must re-wrap on resize. The combined buffer + `rowKind` +
  `rowPatchIdx` ALL rebuild on width change (`NeedsRerenderOnWidthChange`,
  patch_explorer_context.go:148-154).
- Render once into a cached string on file-select; re-run ONLY on width change or
  after a comment is added. Never re-run glow per keystroke/scroll (glow spawn
  ~tens of ms would jank the gocui main loop).

## Diff source command

Feed the presenter a real unified diff string (the REST `patch` field per file,
see github-api.md). If sourcing from local git instead, use no-color/no-pager so
`patch.Parse` works: `git -c color.diff=never diff --no-ext-diff <base>...<head>
-- path`.

## Cheaper fallback (de-scope option, NOT chosen for v1)

If the inline build slips: a gutter marker on commented lines + a secondary
split-main panel (`splitMainPanel`, main_panels.go:135) showing the thread for the
line under the cursor. Needs no custom main renderer. Recorded as the escape
hatch; v1 commits to the inline presenter.
