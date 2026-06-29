# Refinement Questions — Session 1

| Field     | Value |
|-----------|-------|
| Session   | 1 |
| Date      | 2026-05-29 |
| Change ID | pull-requests-tab |

## Analysis Summary

Agents ran: all 6 (Spec Completeness, Problem & Context, Requirements Coverage, System & Infrastructure, Goal & Behaviour, Documentation).
Total findings: ~45 across the four expansion areas (live data, preview pane, review mode, future inline comments).
Breakdown: ~14 critical, ~20 important, ~11 minor.

**The blocking theme:** the current artifacts describe a *finished POC* (hardcoded data, one tab). Your 4 new asks are net-new and several **contradict** the POC spec or hit hard lazytit architecture limits. Nothing below is answerable by reading code — the agents already did that; these are genuine product/architecture decisions. Each question leads with the risk and my recommendation.

## Instructions

Write your answer on the blank line after each **Answer:** marker. Write the option letter and/or freeform text. `SKIP` carries the topic to the next session. When done, re-run `/opsx:refine pull-requests-tab`.

---

## Questions

### Q1 [CRITICAL] — One mega-change or split into a sequence?

The POC `pull-requests-tab` is **fully implemented and `[x]` through phase 6**. Piling live-data + preview + review-mode + future-comments onto it mixes done work with a large not-started block, and the spec would need a confusing mix of MODIFIED (hardcoded→live) and many new ADDED requirements. Items also have a natural dependency order (preview & review-mode both need *real* PR objects, so live-data is a prerequisite).

**Affected files**: whole change folder; `tasks.md` (already all `[x]`), `specs/pull-requests-tab/spec.md`.

**Options**:
- A) **Split into a sequence** (recommended): `pull-requests-live-data` → `pull-requests-preview` → `pull-requests-review-mode` → (future) `pr-inline-comments`. Each independently shippable/testable; the current POC stays archived as-is. Item 4 becomes its own future capability, not specced now.
- B) **One expanded change**: keep everything in `pull-requests-tab`; add a MODIFIED requirement + 3 new ADDED requirement groups + a fresh phase set in tasks.md. Simpler bookkeeping, but a large un-shippable-as-a-unit blob.

**Recommendation**: A. It's the only way item 1's "real PRs" cleanly supersedes the POC's "hardcoded" requirement without a tangled MODIFIED/ADDED mix, and it lets you ship live-data + preview without waiting on the much larger review-mode work.

**Answer:**


---

### Q2 [CRITICAL] — Data source, auth, and the visibility gate (they must agree)

Today there are **two inconsistent mechanisms**: the existing branch-PR-annotation feature fetches via **hand-rolled GraphQL over `net/http`** authenticated by a `go-gh` token (it never runs the `gh` binary), while the POC gates the tab on `exec.LookPath("gh")`. So a user with `GITHUB_TOKEN` but no `gh` binary gets annotations but no tab; a user with `gh` installed-but-unauthenticated gets the tab but live fetch fails. Also: `gh search prs` (the only **cross-repo, user-scoped** query — needed for author/review-requested/involved) carries a **thinner** JSON field set than `gh pr view` (no `reviewDecision`/`statusCheckRollup`/`headRefName`), so the rich preview data needs a *per-PR* `gh pr view`/`gh pr checks` follow-up regardless. Security note: the GraphQL path keeps the token in an HTTP header (never logged); any `gh` call routed through lazygit's command runner is logged verbatim unless built with `.DontLog()`.

**Affected files**: `pkg/commands/git_commands/github.go`, `pkg/gui/gh.go` (the gate), `pkg/commands/models/github.go`, design.md D4.

**Options**:
- A) **Shell out to `gh`** (matches design D4): `gh search prs` for the list, `gh pr view/checks/diff` for detail. Less code for the rich preview; but a hard runtime dep on `gh`, must enforce `.DontLog()` + no-token-in-argv, and re-handle GitHub Enterprise host (`GH_HOST`). Keep the gate on `gh` presence; add an **authenticated** check (`gh auth status`) for content vs an actionable "run `gh auth login`" empty-state.
- B) **Extend the existing GraphQL path**: add a user-scoped `search(query:"involves:@me", type:ISSUE)` query beside the branch-scoped one. Consistent auth, token never logged, Enterprise already handled by `graphQLEndpoint()`, no new binary dep; but more GraphQL to write and you map cross-repo `search` nodes yourself. The gate would change from `gh`-presence to **token-presence**.
- C) **Hybrid**: list+detail via `gh` (A) but keep the gate aligned to "gh present AND authenticated".

**Recommendation**: A or C. You already chose `gh` as the gate, and `gh pr view --json` / `gh pr checks` give checks/reviewers/comments in one call each — far less code than re-deriving them in GraphQL. Accept the `.DontLog()` + stdin-only discipline as hard spec requirements. (If you'd rather avoid the `gh` runtime dep entirely, pick B.) **Consequence either way:** a two-model split — a thin `PullRequestListItem` (from search, carries `repository.nameWithOwner` for cross-repo) and a richer `PullRequestDetail` fetched lazily; the POC's 6-field `GithubPullRequest` is too thin and has no repo field.

**Answer:**


---

### Q3 [CRITICAL] — Preview pane layout (lazytit's main area is a fixed 2 slots)

You described "details on top + a bottom split into checks/CI | reviewers | comments, or maybe tabs." Hard constraint: lazytit's main area is a **fixed two-slot model** (`Main` + `Secondary`) — there is **no 3-way split** and no main-area tab system (the `viewTabMap` tabs are side-window only). A true 3-box layout means surgery on the layout core (`MainContextPair`/`RefreshMainOpts`/`mainSectionChildren`), which is high-blast-radius.

**Affected files**: `pkg/gui/types/rendering.go`, `pkg/gui/main_panels.go`, `pkg/gui/controllers/helpers/window_arrangement_helper.go`; new `docs/preview-layout.md`.

**Options**:
- A) **Single `main` view, one composed string, in-view "tabs"** (recommended v1): render details + the active section (checks/reviewers/comments) as one scrollable buffer; a key (`]`/`[` or a section key) toggles which section shows by re-rendering. Zero layout changes. Tabs are a view-state enum, not real gocui tabs.
- B) **`Main` (top = details) + `Secondary` (bottom = one switchable panel)**: uses the existing 2-slot split; a key cycles the bottom panel content (checks ↔ reviewers ↔ comments). Genuine top/bottom, no new types.
- C) **True 3-box main**: extend the layout core for a third slot. Matches your literal description but touches code every view depends on. Not recommended for v1.

**Recommendation**: A for the first cut (lowest risk, satisfies the spirit), or B if you specifically want a persistent top/bottom split. Avoid C.

**Answer:**


---

### Q4 [CRITICAL] — When is preview detail fetched? (network-per-keystroke risk)

lazytit renders the preview on **every cursor move** (the standard side-list → main behaviour). If selecting a PR triggers a network fetch for checks/reviewers/comments, holding the arrow key through 30 PRs = 30+ API round-trips, tripping rate limits and racing stale responses into the view. There is **no cancellation primitive** (`OnWorker` can't be killed) and **no debounce** in the codebase today.

**Affected files**: `pkg/gui/controllers/helpers/refresh_helper.go`, the PR context's render-to-main; new debounce util.

**Options**:
- A) **Fetch-on-selection, debounced + cached + stale-discard** (recommended): render instantly from already-loaded list fields; after ~250ms of stable selection, fetch detail in a worker; on completion, **discard if the selected PR changed**; cache by PR number so re-selection is instant. Smooth UX, bounded calls.
- B) **Fetch-only-on-Enter**: the preview shows only list-row fields (number/title/state/author/labels) until you press Enter to commit to a PR, which then loads full detail. Zero per-scroll network cost; less rich while browsing.

**Recommendation**: A — it's the expected lazytit feel — but it requires building a small debounce + the "selection still matches" guard (no primitive exists). If you want minimal effort/network, B is a legitimate simpler contract.

**Answer:**


---

### Q5 [CRITICAL] — How is the PR diff obtained for [Enter] → review mode?

Review mode needs the PR's changed files + diffs. lazytit's reusable precedent is the **commitFiles** context (a transient file-list that borrows the branches window and renders a `base...head` diff to main). The diff *source* is an unmade, cascading decision — it also determines whether future inline comments (item 4) are even feasible.

**Affected files**: new review-mode context (modeled on `pkg/gui/context/commit_files_context.go`); new `docs/review-mode.md`.

**Options**:
- A) **`gh pr diff <n>`**: simplest, no local refs touched; but a flat textual diff — you lose lazytit's interactive file-tree/per-file navigation unless you parse it into files yourself.
- B) **`git fetch origin pull/<n>/head` + diff against base** (recommended): no worktree mutation, works for fork PRs, gives real local refs so the commitFiles machinery (file tree, per-file diff, future line-anchored comments) works. Cost: writes a ref under `refs/`, needs cleanup on exit.
- C) **Checkout the PR branch**: mutates the worktree, fails on a dirty tree, awkward for forks. Not recommended.

**Recommendation**: B. It reuses the commitFiles file-tree experience and is the only option that leaves room for item 4's line-anchored comments later. Specify ref cleanup on exit.

**Answer:**


---

### Q6 [IMPORTANT] — "Mark file reviewed": state lifetime + the `space` keybinding

There is **no "reviewed" concept** anywhere in lazytit, and models are reloaded from git after every action (so a naive flag gets wiped on refresh). Also `space` already means **stage/unstage** in file contexts — reusing it for "reviewed" is a muscle-memory collision (though it'd be context-scoped to review mode). And if the PR head **force-pushes** mid-review, file SHAs change and "reviewed" marks become silently stale (false sense of done).

**Affected files**: review-mode context; possibly `AppState` (persistence); `pkg/i18n/english.go` (keybinding strings).

**Options** (you can mix — e.g. "B + ephemeral"):
- A) **Ephemeral, in-memory, session-scoped** (recommended v1): marks live on the review context; lost on quit/refresh. Matches lazytit's "git is the source of truth" philosophy; zero new persistence subsystem.
- B) **Persisted on disk** (per repo + PR number + head-SHA): survives sessions; needs a new client-state store and invalidation when the head SHA changes. Bigger build.
- For the key: **(i)** accept `space` (context-scoped, mirrors staging muscle memory) — recommended; or **(ii)** use a different key to avoid the overload.
- For force-push: clear/flag stale marks when the head SHA changes (don't silently carry them).

**Recommendation**: A + (i), with head-SHA staleness clearing. Persisted state (B) is a real subsystem better deferred unless you want cross-session resume.

**Answer:**


---

### Q7 [IMPORTANT] — glow markdown rendering: in scope for v1, and the security constraint

glow (2.1.1) is installed here; its library (`glamour`) is **not** vendored, so it must be shelled out. Critical security point: PR **titles/bodies/comments are attacker-controlled text** (anyone who opens a PR controls them). If that text is ever interpolated into a shell string (`echo "<body>" | glow`), a body like `$(...)`/`; rm -rf ~` executes arbitrary code.

**Affected files**: new glow gate (mirror the `gh` gate); render-to-main path; new `docs/glow-rendering.md`.

**Options**:
- A) **In scope for v1, with hard constraints** (recommended): detect `glow` via `exec.LookPath` (cached, like `gh`); pipe markdown to `glow -s auto -w <paneWidth>` **via stdin using the argv builder — never `NewShell`/string interpolation, never a command arg**; never `-p` (no pager inside the pane); **plaintext fallback** when glow is absent (preview degrades, never disappears or errors). Config-gated, default on-if-present.
- B) **Defer glow**: render plain markdown text for v1; add glow later. Lower risk surface now.

**Recommendation**: A — but the stdin-only / no-shell-interpolation rule is a non-negotiable spec requirement, not a nicety. If you'd rather not carry that security burden in the first cut, B is fine and the body still renders (just unstyled).

**Answer:**


---

## Processing Log (filled automatically on re-run)

- [ ] All answers parsed
- [ ] Proposal/scope updated (Q1)
- [ ] Specs updated (MODIFIED hardcoded→live; new requirements)
- [ ] Design decisions recorded (D1.x)
- [ ] Tasks updated
- [ ] docs/*.md authored (gh-commands, data-shapes, preview-layout, review-mode, glow-rendering)
- [ ] Refinement log appended
