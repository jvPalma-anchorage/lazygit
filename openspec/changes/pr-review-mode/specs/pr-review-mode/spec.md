## ADDED Requirements

### Requirement: Launch into PR review mode from the CLI

The system SHALL boot directly into a dedicated PR review mode when invoked as
`lazygit <OWNER/REPO> <PR_NUMBER>`, where the first positional matches
`^[^/\s]+/[^/\s]+$` and the second positional is all digits. The system MUST NOT
pass `OWNER/REPO` to the existing git-arg validation, and MUST preserve all
existing invocations (`lazygit`, `lazygit status|branch|log|stash`,
`lazygit -p <path>`) unchanged.

#### Scenario: two-positional review invocation

- **WHEN** lazygit is launched as `lazygit dlvhdr/gh-dash 123` from inside a
  checkout of that repo
- **THEN** lazygit boots directly into PR review mode for pull request 123 of
  `dlvhdr/gh-dash`, focusing the review context instead of the normal Files
  context

#### Scenario: existing git-arg invocations still work

- **WHEN** lazygit is launched as `lazygit status` (or `branch`/`log`/`stash`, or
  with no positionals)
- **THEN** lazygit behaves exactly as before, with no PR review mode and no error

#### Scenario: launched outside a valid checkout

- **WHEN** lazygit is launched as `lazygit dlvhdr/gh-dash 123` and the current
  working directory is not a git work tree
- **THEN** lazygit fails fast with a clear message instructing the user to launch
  inside a checkout of the repository, and does NOT prompt to run `git init` on
  stdin

### Requirement: Show the PR's changed files and their diffs

In PR review mode the system SHALL list every file changed in the pull request
and SHALL render the unified diff for the selected file. The diff data MUST come
from the GitHub API for the pull request (its base...head), not from an unrelated
local comparison.

#### Scenario: file list populated from the PR

- **WHEN** PR review mode finishes loading
- **THEN** the file panel lists one entry per file changed in the pull request,
  and selecting a file renders that file's diff in the main view

#### Scenario: diff renders for the selected file

- **WHEN** the user selects a changed file
- **THEN** the main view shows that file's added, removed, and context lines as a
  unified diff

### Requirement: Show existing review threads anchored inline with author and resolved state

The system SHALL display existing review threads at the code line they anchor to
— on either the RIGHT (added/context) or LEFT (removed) side of the diff —
showing each thread's resolved/unresolved status and the author of each comment,
including reply chains within a thread. Rendering MUST NOT hide threads anchored
to deletion lines. Threads whose anchor line is no longer present in the current
diff (outdated) SHALL be shown in a separate section rather than mis-anchored.

#### Scenario: resolved thread shown at its line with author

- **WHEN** the selected file has a resolved review thread anchored to a diff line
  (RIGHT or LEFT side)
- **THEN** a comment block appears immediately under that diff line showing a
  RESOLVED badge and the comment author's login

#### Scenario: thread on a deletion line is still shown

- **WHEN** the selected file has a review thread anchored to a LEFT-side
  (deletion) line
- **THEN** the thread is rendered under that deletion line and is not dropped or
  mis-anchored

#### Scenario: unresolved thread with replies

- **WHEN** a review thread has a root comment and one or more replies
- **THEN** the block shows an UNRESOLVED badge and renders the root comment and
  each reply with its author, in order

#### Scenario: outdated thread is not mis-anchored

- **WHEN** a review thread's anchor line no longer exists in the current diff
- **THEN** the thread is shown in a separate outdated/other section and is not
  attached to an unrelated diff line

### Requirement: Show PR global comments with author

The system SHALL display the pull request's issue-level (global) comments, each
with its author.

#### Scenario: global comments listed

- **WHEN** PR review mode is open and the pull request has issue-level comments
- **THEN** the system shows each global comment's body and author

### Requirement: Show reviewers and their review state

The system SHALL display the pull request's reviewers and each reviewer's latest
review state (e.g. APPROVED, CHANGES_REQUESTED, COMMENTED, PENDING). Reviewer
entries that resolve to a null or team requested-reviewer MUST NOT crash the
view.

#### Scenario: reviewers with states listed

- **WHEN** PR review mode is open
- **THEN** the system lists each reviewer with their latest review state

#### Scenario: pending / team reviewer handled safely

- **WHEN** the pull request has a requested reviewer that is a team or has not yet
  reviewed
- **THEN** the view renders without error, treating an absent state as pending

### Requirement: Add a review comment over a selected line range

The system SHALL let the user select a start and end line range within the
displayed diff and submit a new review comment anchored to that range. The system
MUST map the selection to GitHub's `line`/`side` (and `start_line`/`start_side`
for multi-line) using new-file line numbers on the RIGHT side, and MUST post it
with the pull request's current head commit id. A single-line selection MUST omit
`start_line`.

#### Scenario: multi-line range comment posted

- **WHEN** the user selects a range of RIGHT-side diff lines, enters a comment
  body, and submits
- **THEN** the system posts a review comment to the pull request with
  `start_line`/`line` set to the selected range's new-file line numbers on the
  RIGHT side and the pull request's head commit id

#### Scenario: single-line comment omits start_line

- **WHEN** the user selects a single RIGHT-side diff line and submits a comment
- **THEN** the system posts the comment with `line` and `side` only, without
  `start_line`/`start_side`

#### Scenario: non-RIGHT selection rejected in v1

- **WHEN** the user's selection resolves to a deletion (LEFT-side) line
- **THEN** the system declines to post and shows a message that v1 supports
  commenting on added/context lines only

### Requirement: Render markdown through glow with a plain-text fallback

The system SHALL render comment and markdown bodies through `glow` when `glow` is
available on `PATH`, and SHALL fall back to printing the raw markdown text when
`glow` is absent, errors, or produces empty output. The system MUST pass the
markdown body to glow via standard input (never interpolated into a shell
command).

#### Scenario: glow present renders styled markdown

- **WHEN** `glow` is on `PATH` and a comment body is displayed
- **THEN** the body is rendered with glow's styling (an explicit style with forced
  color) before being shown in the view

#### Scenario: glow absent falls back to plain text

- **WHEN** `glow` is not on `PATH`
- **THEN** the comment body is shown as its raw markdown text without error

#### Scenario: glow failure falls back to plain text

- **WHEN** `glow` is present but exits non-zero or produces empty output for a body
- **THEN** the system shows the raw markdown body unchanged and surfaces no glow
  error to the user

### Requirement: Navigate changed files as a tree

The system SHALL present the pull request's changed files using lazygit's
file-tree presentation (nested directories that can be collapsed and expanded),
not a flat filename list, and SHALL drive the diff view from the file the cursor
is on.

#### Scenario: files shown as a collapsible tree

- **WHEN** the pull request changes files across multiple directories
- **THEN** the file panel groups them under collapsible directory nodes, and
  moving the cursor onto a file renders that file's diff with inline threads

#### Scenario: cursor on a file path selects it

- **WHEN** the user moves the cursor onto a file entry in the tree
- **THEN** that file becomes the selected file and its diff is shown in the main
  view

### Requirement: Mark a file as reviewed

The system SHALL let the user toggle a "reviewed" state on the selected file from
the file tree and SHALL show a distinct indicator for reviewed files. The
reviewed state SHALL be session-scoped in v1 (held in memory, not persisted
across restarts).

#### Scenario: toggle reviewed on a file

- **WHEN** the user presses the mark-reviewed key on the selected file
- **THEN** the file is marked reviewed and shows a reviewed indicator in the tree;
  pressing the key again clears the indicator

#### Scenario: reviewed state is independent per file

- **WHEN** the user marks several files reviewed during a review session
- **THEN** each file independently retains its reviewed indicator for the duration
  of the session

### Requirement: Toggle between unified and side-by-side diff

The system SHALL let the user toggle the diff rendering of the selected file
between a unified view and a side-by-side view. Inline review threads SHALL remain
attached to their anchored lines in both modes.

#### Scenario: toggle to side-by-side

- **WHEN** the user presses the diff-mode toggle key while viewing a file's diff
- **THEN** the diff re-renders side-by-side (old and new content in parallel
  columns) with review threads still shown at their anchored lines

#### Scenario: toggle back to unified

- **WHEN** the user presses the toggle key again
- **THEN** the diff re-renders as a unified diff with threads inline

### Requirement: View the pull request description

The system SHALL provide a key that opens the pull request's description (title +
body), rendered through `glow` when available (per the markdown rendering
requirement) and as plain text otherwise, and dismissible back to the review.

#### Scenario: open and dismiss the description

- **WHEN** the user presses the description key in review mode
- **THEN** the pull request's title and body are shown (glow-rendered when
  available), and dismissing it returns the user to the review view
