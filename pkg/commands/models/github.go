package models

import "strconv"

type GithubPullRequest struct {
	HeadRefName         string                `json:"headRefName"`
	Number              int                   `json:"number"`
	Title               string                `json:"title"`
	State               string                `json:"state"` // "MERGED", "OPEN", "CLOSED", "DRAFT"
	Url                 string                `json:"url"`
	HeadRepositoryOwner GithubRepositoryOwner `json:"headRepositoryOwner"`
}

// ID satisfies the HasID constraint used by list view-models. A PR number is
// unique within its repository, so it identifies the row.
func (pr *GithubPullRequest) ID() string {
	return strconv.Itoa(pr.Number)
}

// URN satisfies types.HasUrn (used for selection tracking in list contexts).
func (pr *GithubPullRequest) URN() string {
	return "pull_request-" + pr.ID()
}

func (pr *GithubPullRequest) UserName() string {
	// e.g. 'jesseduffield'
	return pr.HeadRepositoryOwner.Login
}

func (pr *GithubPullRequest) BranchName() string {
	// e.g. 'feature/my-feature'
	return pr.HeadRefName
}

type GithubRepositoryOwner struct {
	Login string `json:"login"`
}

// ReviewComment is a single comment within a review thread. A thread's comments
// form a reply chain ordered root-first; the root comment has ReplyToID == 0.
type ReviewComment struct {
	DatabaseID int
	Author     string
	Body       string
	CreatedAt  string
	// ReplyToID is the DatabaseID of the comment this one replies to, or 0 for
	// the root comment of the thread.
	ReplyToID int
}

// ReviewThread is a line-anchored review conversation on a pull request. Side is
// "RIGHT" (added/context lines, anchored by new-file line) or "LEFT" (deletion
// lines, anchored by old-file line). When the anchored line no longer exists in
// the current diff the thread is outdated: Line/StartLine are 0 and DiffHunk
// holds the captured context (its last line is the commented line).
type ReviewThread struct {
	ID        string
	Path      string
	Line      int
	StartLine int
	// OriginalLine is the line number in the diff at the time the thread was
	// created. It is the fallback anchor for threads whose current Line is null
	// (the code moved), letting a LEFT/RIGHT thread still be located in the
	// trailing/outdated section rather than being silently dropped.
	OriginalLine int
	Side         string
	IsResolved   bool
	IsOutdated   bool
	DiffHunk     string
	Comments     []ReviewComment
}

// IssueComment is a pull-request-level (global) comment, not anchored to a line.
type IssueComment struct {
	Author    string
	Body      string
	CreatedAt string
}

// Reviewer is a pull request reviewer and their latest review state (e.g.
// "APPROVED", "CHANGES_REQUESTED", "COMMENTED", "PENDING"). A requested reviewer
// that has not yet reviewed (including a team) is represented with State
// "PENDING".
type Reviewer struct {
	Login string
	State string
}

// ID implements the list-view-model HasID interface (login uniquely identifies a
// reviewer within a PR).
func (r *Reviewer) ID() string {
	return r.Login
}

// Review is a single submitted pull-request review: its author, its state, and its
// optional summary body (the text entered when submitting the review). Used to show a
// reviewer's review body in the Conversation tab.
type Review struct {
	Author      string
	State       string
	Body        string
	SubmittedAt string
}

// GithubPullRequestFile is a file changed by a pull request, with its unified
// diff Patch as returned by the REST pulls/{n}/files endpoint.
type GithubPullRequestFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Patch     string `json:"patch"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}
