## ADDED Requirements

Scope marker: requirements tagged **[DEFERRED — Phase N]** are specified now but
ship after v1 (phases 1–5). All untagged requirements are v1. See `tasks.md` for
the phase mapping and `design.md` D1.6.

### Requirement: Dedicated review-workspace layout with its own windows and tabs

When booted in PR review mode, the system SHALL render an alternate set of
left-hand side windows — a PR list `[1]`, PR content `[2]`, and PR activity `[3]`
— in place of the normal `Status/Files/Branches/Commits/Stash` windows, each with
its own tabs and panel numbers. The normal Status window SHALL be suppressed in
review mode. Tab cycling (`[` / `]`) SHALL stay within the review windows' tabs and
MUST NOT surface the normal windows' tabs. Normal-mode launches MUST be unaffected.

#### Scenario: review windows replace the normal windows

- **WHEN** lazygit boots in PR review mode
- **THEN** the side column shows `[1]` PR list, `[2]` PR content, and `[3]` PR
  activity, and the normal `Files`/`Branches`/`Commits`/`Stash`/`Status` windows
  are not shown

#### Scenario: tab cycling stays within the review layout

- **WHEN** the user presses `]` (next tab) while focused on the PR content window
- **THEN** the tab switches between `Overview` and `Files Changed` only, and never
  shows `Files`/`Worktrees`/`Submodules`

#### Scenario: initial focus on boot

- **WHEN** lazygit boots in PR review mode with a target PR
- **THEN** focus starts on window `[2]` PR content, tab `Files Changed`, with the
  first file (or first directory node) selected

#### Scenario: normal mode is unchanged

- **WHEN** lazygit is launched normally (no review target)
- **THEN** the side windows, tabs, and panel numbers are exactly as before

### Requirement: Project-agnostic boot and repository-identity verification

PR review mode SHALL work in any git checkout without project-specific
configuration: the repository identity (`owner/repo`), the checkout path, and the
PR's base/head commit ids and repository URLs SHALL be resolved at runtime from the
launched checkout and the pull-request data, never from a hardcoded path or a
per-project constant. Before fetching any refs or posting any comment, the system
SHALL verify that the current checkout's remotes resolve to the launched
`OWNER/REPO`, and SHALL fail fast on a mismatch.

#### Scenario: boots in an arbitrary checkout

- **WHEN** lazygit is launched in PR review mode from any checkout of the target
  repository
- **THEN** it resolves the repository identity and PR refs at runtime and operates
  without any project-specific path or configuration

#### Scenario: checkout does not match the target repository

- **WHEN** the current checkout's remotes do not resolve to the launched
  `OWNER/REPO`
- **THEN** the system fails fast with a clear message and performs no ref fetch and
  no comment post

### Requirement: PR list window shows the launched pull request

In v1 the PR list window `[1]` SHALL show the pull request the session was launched
for, and selecting it SHALL load it into the content and activity windows.

#### Scenario: launched PR is shown and selected

- **WHEN** lazygit is launched as `llg-dev OWNER/REPO 123`
- **THEN** window `[1]` lists pull request 123 and it is the selected PR, with its
  content and activity loaded

### Requirement: PR list driven by gh-dash saved queries [DEFERRED — Phase 8]

The PR list window `[1]` SHALL present, as its tabs, the pull-request sections
defined in the user's gh-dash configuration (`prSections`), listing the PRs
matching the active section's filter. When the gh-dash config is absent or empty,
the system SHALL fall back to a default set of sections scoped to the launched
repository.

#### Scenario: gh-dash sections become tabs

- **WHEN** the user's gh-dash config defines sections such as `Needs My Review`
  and `Involved`
- **THEN** window `[1]` shows those section titles as tabs, each listing the PRs
  matching that section's filter

#### Scenario: gh-dash config absent

- **WHEN** no gh-dash config is present or it defines no sections
- **THEN** window `[1]` shows a default section set scoped to the launched
  repository, without error

### Requirement: PR overview

The PR content window's `Overview` tab SHALL present the pull request's metadata in
the main panel: title, base and head branch, author, created/updated dates, PR
number, labels with their colors, assignees, milestone, and linked development.
Missing optional fields (no milestone, no assignees) SHALL render without error.

#### Scenario: overview shows PR metadata

- **WHEN** the user selects the `Overview` tab of the content window
- **THEN** the main panel shows the PR's title, `base ← head`, author, dates,
  number, colored labels, assignees, and milestone

#### Scenario: overview with missing optional fields

- **WHEN** the pull request has no milestone and no assignees
- **THEN** the overview renders the remaining fields without error

### Requirement: Changed files as a tree with pager-rendered diffs

The PR content window's `Files Changed` tab SHALL present the pull request's
changed files as a collapsible tree (not a flat list). Moving the cursor onto a
file SHALL render that file's diff in the main panel **through the user's
configured diff pager**. Moving the cursor onto a directory node SHALL render the
aggregate diff of all files under that directory. When a directory contains both
viewed and unviewed files, the aggregate SHALL split the main panel — unviewed
files' diffs on top, viewed files' diffs below — mirroring lazygit's unstaged/staged
split, with the "viewed" mark playing the role of "staged".

#### Scenario: file diff rendered through the configured pager

- **WHEN** the user's git pager is configured (e.g. git-delta) and the user selects
  a changed file
- **THEN** the main panel shows that file's diff rendered by the configured pager,
  matching how diffs look elsewhere in lazygit

#### Scenario: directory node shows aggregate diff

- **WHEN** the user moves the cursor onto a directory node in the file tree
- **THEN** the main panel shows the combined diff of every changed file under that
  directory, not a "no changed files" message

#### Scenario: directory with mixed viewed state splits the diff

- **WHEN** a directory contains both viewed and unviewed files and the cursor is on it
- **THEN** the main panel splits, showing the unviewed files' aggregate diff on top and
  the viewed files' aggregate diff on the bottom

#### Scenario: empty pull request

- **WHEN** the pull request has no changed files
- **THEN** the Files Changed tab shows an empty-state message and does not error

### Requirement: Bounded rendering for large pull requests

The system SHALL show all of a pull request's changed files in the tree, matching
lazygit's own uncapped Files/CommitFiles panels, and SHALL cap only pathologically
huge pull requests (hundreds of files, where resolving per-file identity would stall),
surfacing the hidden count when the cap applies. The diff preview SHALL be rendered
through the user's pager via lazygit's incremental (lazy) streaming, which bounds each
render and never stalls, so the diff keeps full pager (delta) rendering rather than
being hard-truncated at a fixed line count (decision: keep delta; lazy streaming is the
line guard; the file list is not artificially capped for normal PRs).

#### Scenario: normal PR shows every changed file

- **WHEN** a pull request changes a normal number of files (e.g. tens)
- **THEN** the tree lists every changed file, not an artificially truncated subset

#### Scenario: render cap on a pathologically huge PR

- **WHEN** a pull request changes more files than the safety cap
- **THEN** the tree shows the capped subset and surfaces a "first N; M more hidden"
  subtitle, and the UI does not hang

#### Scenario: large single-file or aggregate diff

- **WHEN** a selected file's diff (or a directory's aggregate diff) is very large
- **THEN** the main panel streams it incrementally through the pager without stalling,
  and the directory aggregate is bounded by the 10-file tree cap

### Requirement: Toggle and persist a content-invalidated viewed state

The system SHALL let the user toggle a "viewed" mark on a file in the tree
(`space`), SHALL show a distinct indicator for viewed files, and SHALL persist that
mark keyed by repository, PR, path, and the file's content identity (its blob OID),
so it survives leaving and re-opening the pull request. A viewed mark SHALL be
invalidated automatically when the file's content changes, and SHALL be handled
correctly for added, deleted, renamed, and binary files. Viewed-state entries for
merged or closed pull requests SHALL be pruned on boot.

#### Scenario: viewed mark persists across re-entry

- **WHEN** the user presses `space` to mark a file viewed, leaves the pull request,
  and reopens it
- **THEN** the file is still shown as viewed

#### Scenario: viewed mark clears when the file changes

- **WHEN** a file the user marked viewed is changed by a newer commit (its blob OID
  differs from the stored one)
- **THEN** the file is shown as not-viewed again

#### Scenario: renamed and deleted files

- **WHEN** a viewed file is renamed (new path) or deleted in a later commit
- **THEN** the rename clears the mark (new path is unviewed) and the deleted file is
  not shown as viewable, without error

### Requirement: Select a line range and post a review comment

The system SHALL provide a focusable diff surface, entered by pressing Enter on a
file in the tree, where the user can move a cursor over diff lines, select a
start-and-end range, and post a standalone review comment anchored to that range.
The selection MUST map to GitHub's line and side parameters (and start_line /
start_side for multi-line) using new-file line numbers on the RIGHT side, and MUST
be posted with the pull request's current head commit id. A single-line selection
MUST omit start_line. Pressing Enter on a directory node SHALL collapse or expand
it rather than open a diff.

#### Scenario: multi-line range comment posted

- **WHEN** the user selects a range of RIGHT-side diff lines, enters a comment body
  (`c`), and submits
- **THEN** the system posts a review comment with `start_line`/`line` set to the
  selected range's new-file line numbers on the RIGHT side and the PR's head commit
  id

#### Scenario: single-line comment omits start_line

- **WHEN** the user selects a single RIGHT-side diff line and submits a comment
- **THEN** the comment is posted with `line` and `side` only

#### Scenario: enter on a directory collapses it

- **WHEN** the user presses `enter` on a directory node in the file tree
- **THEN** the directory collapses or expands and no diff surface is opened

#### Scenario: comment post fails

- **WHEN** the comment post returns a 4xx/5xx (e.g. 422 invalid line, 401 auth)
- **THEN** the system shows an error toast and does not silently drop the comment

### Requirement: Submit a grouped review [DEFERRED — Phase 7]

The system SHALL let the user accumulate review comments across multiple files
during a session and submit them as a single pull-request review with an overall
body and an event (comment / approve / request-changes).

#### Scenario: grouped review submitted

- **WHEN** the user has added comments on more than one file and submits the review
  with a body
- **THEN** the system posts one pull-request review containing all the pending
  comments and the body, rather than separate standalone comments

### Requirement: Conversation — reviewers, states, threads, and PR-body comments [DEFERRED — Phase 6]

The PR activity window's `Conversation` tab SHALL list the pull request's reviewers,
each with a state indicator (pending / commented / approved / changes-requested).
Selecting a reviewer SHALL show, in the main panel, that reviewer's review body and
their thread comments, each thread annotated with its state (resolved / unresolved /
outdated). The conversation SHALL also include pull-request-body (issue-level)
comments.

#### Scenario: reviewers listed with states

- **WHEN** the user opens the `Conversation` tab
- **THEN** each reviewer is listed with an icon reflecting their latest state, and a
  requested-but-not-yet-reviewed reviewer shows as pending without error

#### Scenario: reviewer's comments with thread state

- **WHEN** the user selects a reviewer who left code comments
- **THEN** the main panel shows that reviewer's comments grouped by file/thread,
  each thread showing whether it is resolved, unresolved, or outdated

#### Scenario: PR-body comments shown

- **WHEN** the pull request has issue-level (non-code) comments
- **THEN** those comments appear in the conversation alongside the code threads

### Requirement: Checks tree with importance ordering and logs [DEFERRED — Phase 8]

The PR activity window's `Checks` tab SHALL list the pull request's checks/actions
as a collapsible tree (an action with sub-actions renders as an expandable parent).
The list SHALL be sorted first by importance — failed, then cancelled, then
interrupted, then pending, then succeeded, then skipped (skipped last) — and within
the same importance by most-recent update. Entering a check SHALL render its logs
(including errors) in the main panel with color output forced on.

#### Scenario: checks sorted by importance then recency

- **WHEN** the user opens the `Checks` tab on a PR with mixed check states
- **THEN** failed checks appear first and skipped checks appear last, with
  same-importance checks ordered by most recent update

#### Scenario: nested actions expand and collapse

- **WHEN** a check has child actions
- **THEN** it renders as a collapsible parent node whose children can be expanded,
  collapsed, and entered

#### Scenario: check logs shown with color

- **WHEN** the user enters a check
- **THEN** the main panel shows that check's logs and errors with color output
  preserved

### Requirement: PR commits only

The PR activity window's `Commits` tab SHALL list only the commits between the pull
request's head and its base (the PR's own commits), behaving like lazygit's normal
commits panel: the cursor commit's message is shown, and entering a commit drills
into that commit's changed files with diffs in the main panel.

#### Scenario: only PR commits listed

- **WHEN** the user opens the `Commits` tab
- **THEN** the list contains exactly the commits between the PR base and head, and
  no unrelated repository commits

#### Scenario: drilling into a PR commit

- **WHEN** the user enters a commit in the `Commits` tab
- **THEN** the main panel shows that commit's changed files and their diffs

### Requirement: All GitHub data via the gh CLI, with fail-fast boot and recoverable panel errors

The system SHALL read and write all GitHub data (PR lists, metadata, changed files,
review threads, comments, reviewers, checks, and logs) through the `gh`
command-line tool, reusing the user's existing `gh` authentication, passing any
user-provided filter strings as arguments (never interpolated into a shell). If
`gh` is missing or unauthenticated, or the initial load (identity check, ref fetch,
or first query) fails, the system SHALL fail fast at boot with an actionable
message and SHALL NOT hang on a "Loading…" state. After the workspace has loaded,
a per-panel `gh` failure SHALL surface as a toast with a retry, not a frozen panel.

#### Scenario: gh used for GitHub access

- **WHEN** the workspace loads any GitHub-sourced panel
- **THEN** the data is obtained by invoking `gh`, not a separately-authenticated
  HTTP path

#### Scenario: gh missing or unauthenticated at boot

- **WHEN** `gh` is not installed or not authenticated when review mode boots
- **THEN** the system exits fast with a clear, actionable message and does not hang

#### Scenario: per-panel gh failure after load

- **WHEN** a panel's `gh` call fails after the workspace has loaded
- **THEN** that panel shows an error toast with a retry option and the rest of the
  workspace remains usable

#### Scenario: filter strings passed safely

- **WHEN** a gh-dash filter string is used to query PRs
- **THEN** it is passed as a command argument (argv), never interpolated into a
  shell command
