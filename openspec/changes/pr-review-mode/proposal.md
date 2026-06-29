## Why

gh-dash (dlvhdr/gh-dash) gives the user a fast keyboard-driven list of pull
requests, but reviewing one still means leaving the terminal for a browser. The
user already launches this lazygit fork from gh-dash for other actions (the
existing `L`/`E`/`D`/`B` keybindings shell out to it). The missing capability is
a single keystroke in gh-dash that drops the user straight into a dedicated
**PR review mode** for a specific `OWNER/REPO` + `PR_NUMBER`, where they can read
the diff, read existing review threads anchored to their lines, see global
comments / reviewers / states, and add new range-anchored review comments —
without opening GitHub in a browser.

This fork already has GitHub plumbing: `pkg/commands/git_commands/github.go`
fetches PRs via hand-rolled GraphQL over `net/http` using a go-gh auth token
(it does **not** shell out to `gh`), and there is a `gh`-gated proof-of-concept
"Pull Requests" tab in the branches window (the separate `pull-requests-tab`
change). PR review mode is a **separate entry path**: it reuses the auth token,
the `GithubPullRequest` model, and the commit-files diff machinery where they
fit, but it is not the POC tab and does not depend on it.

## What Changes

- **New CLI entry shape.** `lazygit <OWNER/REPO> <PR_NUMBER>` (two bare
  positionals) boots directly into PR review mode. Today this is a hard error
  (`parseGitArg` only accepts status/branch/log/stash and flaggy rejects a
  second unregistered positional), so the signature is free to claim and the
  change is non-breaking.
- **New StartArgs path.** A `ReviewTarget{Owner, Repo, PRNumber}` is threaded
  through `StartArgs` into the GUI, which branches into a new PR review context
  instead of the normal Files context on boot.
- **New GitHub read path.** One combined GraphQL query (run via the existing
  `net/http` + go-gh token transport, mirroring `fetchRecentPRsAux`) fetches:
  review threads (line-anchored, with resolved/outdated state + author + reply
  chains), issue-level comments, reviewers + review states, and PR metadata
  (`id`, `headRefOid`). The unified diff per file comes from a second REST call
  (`pulls/{n}/files` `patch` field) or `gh pr diff`.
- **New review UI.** A **file tree** (reusing lazygit's commit-files tree UX, not
  a flat list) with a per-file **"mark reviewed"** indicator, plus a custom inline
  diff presenter that interleaves comment threads at their anchored lines (both
  sides), rendering resolved/unresolved badges and authors. `[t]` toggles
  unified ↔ side-by-side; `[d]` opens the PR description (glow-rendered).
- **Add-comment-by-range.** Selecting a start+end line range in the diff and
  submitting posts a review comment via `POST /pulls/{n}/comments` (line + side,
  not the legacy diff-position offset), authored with the same go-gh token.
- **Glow rendering.** Comment/markdown bodies render through `glow` when it is on
  `PATH` (cached LookPath gate mirroring `pkg/gui/gh.go`), falling back to plain
  text otherwise.

**Reference blueprint:** `~/projects/gh-review` (Rust) already implements this
exact feature (same CLI signature, same gh-dash keybinding). It is a proven
template — `docs/gh-review-reference.md` maps its modules to lazygit equivalents.
The two things it lacks (a file *tree* and *mark-reviewed*) are exactly what
lazygit adds here.

**Honest v1 scope (see design.md for the committed decisions and non-goals):**
the inline renderer with selectable, collapsible comment blocks is hard. v1
*displays* threads on **both** diff sides (so no comments are hidden) with
non-selectable blocks; what's deferred is **LEFT-side comment *creation*** (adding
a new comment stays RIGHT-side only), batched/pending reviews, resolve/unresolve
and reply mutations, persisted reviewed state, and clone-less boot. Side-by-side
(`[t]`) is in scope but is the first thing to defer to a fast-follow if schedule
slips — unified alone satisfies the core review flow.

## Capabilities

### New Capabilities
- `pr-review-mode`: a dedicated review session launched as
  `lazygit OWNER/REPO PR_NUMBER`, showing the PR's changed files and diff with
  existing review threads interleaved inline (author + resolved state), global
  comments, reviewers and their states, the ability to add a review comment over
  a selected line range, and markdown rendering through glow with a plain-text
  fallback.

### Modified Capabilities
<!-- None. The existing branch-level PR annotation and the pull-requests-tab POC
     are untouched. The CLI grammar change is purely additive (the two-positional
     form is a hard error today). -->

## Impact

- **New files**:
  - `pkg/gui/context/pr_review_context.go` — the review context (not an
    `IListContext`; owns its own view-line↔model map).
  - `pkg/gui/presentation/pr_review_diff.go` — the inline diff+comment presenter.
  - `pkg/gui/glow.go` + `pkg/gui/glow_test.go` — cached glow-availability gate and
    `renderMarkdown` helper (mirrors `pkg/gui/gh.go`/`gh_test.go`).
  - `pkg/commands/git_commands/github_review.go` — the combined review-data
    GraphQL query, the per-file diff fetch, and the add-comment POST.
  - New models in `pkg/commands/models/github.go` (or a sibling file):
    `ReviewThread`, `ReviewComment`, `IssueComment`, `Reviewer`.
- **Touched files**:
  - `pkg/app/entry_point.go` — register a 2nd optional positional, detect review
    mode after `flaggy.Parse` and before `parseGitArg`, add fields to `cliArgs`.
  - `pkg/app/types/types.go` — `ReviewTarget` + `StartArgs` field + `NewStartArgs`.
  - `pkg/app/app.go` — relax the not-a-repo boot guard for review mode (v1: still
    require a local checkout via `cd`/`-p`, so this is a guard tweak, not a
    repo-less rewrite).
  - `pkg/gui/gui.go` — `resetState`/boot wiring: when `ReviewTarget` is set, make
    the PR review context the initial context (the branch happens one level above
    `initialContext`, which only returns `IListContext`).
  - `pkg/gui/views.go`, `pkg/gui/context/setup.go` — declare/register the review
    view + context.
  - `pkg/i18n/english.go` — new user-facing strings.
- **gh-dash side**: a documented keybinding (no gh-dash code change) that runs
  `cd {{.RepoPath}} && llg-dev {{.RepoName}} {{.PrNumber}}`.
- **Dependencies**: no new Go modules. `glow` and `gh` are external binaries
  detected via `exec.LookPath`; go-gh is already vendored.
- **No CI/pipeline changes.**
