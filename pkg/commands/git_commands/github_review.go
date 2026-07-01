package git_commands

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
)

// PullRequestReviewData is the aggregate result of the combined review-data read
// query: the PR's identity (ID + live head SHA, both needed by the add-comment
// write path), its description, and the review threads / global comments /
// reviewers that the review UI renders. Truncated is set when any paginated
// connection reported hasNextPage (v1 caps each connection at first:50, see D3).
type PullRequestReviewData struct {
	ID     string
	Number int
	Title  string
	Body   string
	State  string
	// Author, BaseRefName and HeadRefName describe the PR for the Overview tab.
	Author      string
	BaseRefName string
	HeadRefName string
	HeadRefOid  string
	// BaseRefOid, BaseRepoURL and HeadRepoURL drive the local-ref read model
	// (DW2 / Phase 2): the exact OIDs to fetch and the per-side repo URLs (which
	// differ for a fork head) so the fetch is reproducible and fork-aware.
	BaseRefOid    string
	BaseRepoURL   string
	HeadRepoURL   string
	Threads       []models.ReviewThread
	IssueComments []models.IssueComment
	Reviewers     []models.Reviewer
	// Reviews are the submitted reviews with their summary bodies, for the
	// Conversation tab's per-reviewer detail (Phase 6). Reviewers (above) carries
	// only the latest state per author for the badge list.
	Reviews   []models.Review
	Truncated bool
}

// reviewDataQuery is the combined read query (design.md D3). It mirrors the
// blueprint in openspec/changes/pr-review-mode/docs/github-api.md: one round trip
// returns everything except the diff (which GraphQL cannot return; that comes
// from FetchPRChangedFiles). Each connection is capped at first:50 with a
// hasNextPage flag so the caller can surface a truncation note.
const reviewDataQuery = `query($owner:String!,$repo:String!,$number:Int!){
  repository(owner:$owner,name:$repo){
    pullRequest(number:$number){
      id number title state isDraft baseRefName headRefName headRefOid baseRefOid body
      baseRepository{ url }
      headRepository{ url }
      author{ login }
      comments(first:50){ pageInfo{ hasNextPage }
        nodes{ author{ login } body createdAt } }
      reviewRequests(first:50){ pageInfo{ hasNextPage }
        nodes{ requestedReviewer{ __typename ... on User{ login } ... on Team{ name } } } }
      latestReviews(first:50){ pageInfo{ hasNextPage }
        nodes{ author{ login } state submittedAt } }
      reviews(first:50){ pageInfo{ hasNextPage }
        nodes{ author{ login } state submittedAt body } }
      reviewThreads(first:50){ pageInfo{ hasNextPage }
        nodes{
          id isResolved isOutdated isCollapsed path line startLine originalLine diffSide
          comments(first:50){ pageInfo{ hasNextPage }
            nodes{ databaseId author{ login } body createdAt diffHunk
                   originalLine line replyTo{ databaseId } } } } } } } }`

// FetchPRReviewData runs the combined review-data read query for a single pull
// request by shelling out to the authenticated `gh` CLI (`gh api graphql`). This
// is the same backend gh-dash uses, so auth (incl. keyring / GitHub Enterprise
// hosts), host resolution, and request timeouts are all handled by gh rather than
// re-implemented here. The token argument is unused (gh resolves its own auth) and
// retained only for call-site compatibility.
func (self *GitHubCommands) FetchPRReviewData(owner string, repo string, number int, _ string) (*PullRequestReviewData, error) {
	cmdArgs := []string{
		"gh", "api", "graphql",
		"-f", "query=" + reviewDataQuery,
		"-f", "owner=" + owner,
		"-f", "repo=" + repo,
		"-F", "number=" + strconv.Itoa(number),
	}

	stdout, stderr, err := self.cmd.New(cmdArgs).DontLog().RunWithOutputs()
	if err != nil {
		return nil, fmt.Errorf("gh api graphql failed: %w: %s", err, stderr)
	}

	return parsePRReviewData([]byte(stdout))
}

// FetchPRChangedFiles fetches the per-file unified diffs for a pull request via
// `gh api` against the REST pulls/{n}/files endpoint (the diff source chosen in
// D3: the `patch` field feeds the per-file inline renderer directly). --paginate
// merges all pages into a single JSON array, so PRs with more than 100 changed
// files are no longer silently truncated.
func (self *GitHubCommands) FetchPRChangedFiles(owner string, repo string, number int, _ string) ([]*models.GithubPullRequestFile, error) {
	cmdArgs := []string{
		"gh", "api",
		fmt.Sprintf("repos/%s/%s/pulls/%d/files", owner, repo, number),
		"--paginate",
	}

	stdout, stderr, err := self.cmd.New(cmdArgs).DontLog().RunWithOutputs()
	if err != nil {
		return nil, fmt.Errorf("gh api pulls/files failed: %w: %s", err, stderr)
	}

	return parsePRChangedFiles([]byte(stdout))
}

// AddReviewComment posts a standalone review comment over a new-file line range
// (design D5). It maps directly to GitHub's line+side API (not the legacy diff
// `position` offset): {body, commit_id, path, side:RIGHT, line}. A multi-line
// range additionally sends {start_line, start_side:RIGHT}; a single-line comment
// must OMIT start_line entirely (sending start_line == line is a 422). commit_id
// must be the live headRefOid from the read query (a stale SHA silently marks the
// comment "outdated"). On a non-201 response the body is surfaced in the error so
// the caller can toast a 422 validation message.
func (self *GitHubCommands) AddReviewComment(owner string, repo string, number int, headOid string, path string, body string, startLine int, line int, _ string) error {
	cmdArgs := []string{
		"gh", "api", "--method", "POST",
		fmt.Sprintf("repos/%s/%s/pulls/%d/comments", owner, repo, number),
		"-f", "body=" + body,
		"-f", "commit_id=" + headOid,
		"-f", "path=" + path,
		"-f", "side=RIGHT",
		"-F", "line=" + strconv.Itoa(line),
	}
	// A single-line comment must OMIT start_line (sending start_line == line is a
	// 422); a multi-line range additionally sends start_line + start_side.
	if startLine != line {
		cmdArgs = append(cmdArgs,
			"-F", "start_line="+strconv.Itoa(startLine),
			"-f", "start_side=RIGHT",
		)
	}

	_, stderr, err := self.cmd.New(cmdArgs).DontLog().RunWithOutputs()
	if err != nil {
		return fmt.Errorf("gh api add-comment failed: %w: %s", err, stderr)
	}

	return nil
}

func parsePRChangedFiles(respBytes []byte) ([]*models.GithubPullRequestFile, error) {
	var files []*models.GithubPullRequestFile
	if err := json.Unmarshal(respBytes, &files); err != nil {
		return nil, err
	}
	return files, nil
}

// --- GraphQL response shapes (only the fields we consume) ---

type reviewDataResponse struct {
	Data struct {
		Repository struct {
			PullRequest *reviewPullRequestNode `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type graphQLPageInfo struct {
	HasNextPage bool `json:"hasNextPage"`
}

type graphQLLogin struct {
	Login string `json:"login"`
}

type reviewPullRequestNode struct {
	ID             string `json:"id"`
	Number         int    `json:"number"`
	Title          string `json:"title"`
	State          string `json:"state"`
	IsDraft        bool   `json:"isDraft"`
	BaseRefName    string `json:"baseRefName"`
	HeadRefName    string `json:"headRefName"`
	HeadRefOid     string `json:"headRefOid"`
	BaseRefOid     string `json:"baseRefOid"`
	Body           string `json:"body"`
	BaseRepository *struct {
		URL string `json:"url"`
	} `json:"baseRepository"`
	HeadRepository *struct {
		URL string `json:"url"`
	} `json:"headRepository"`
	Author   *graphQLLogin `json:"author"`
	Comments struct {
		PageInfo graphQLPageInfo `json:"pageInfo"`
		Nodes    []struct {
			Author    *graphQLLogin `json:"author"`
			Body      string        `json:"body"`
			CreatedAt string        `json:"createdAt"`
		} `json:"nodes"`
	} `json:"comments"`
	ReviewRequests struct {
		PageInfo graphQLPageInfo `json:"pageInfo"`
		Nodes    []struct {
			RequestedReviewer *struct {
				Typename string `json:"__typename"`
				Login    string `json:"login"`
				Name     string `json:"name"`
			} `json:"requestedReviewer"`
		} `json:"nodes"`
	} `json:"reviewRequests"`
	LatestReviews struct {
		PageInfo graphQLPageInfo `json:"pageInfo"`
		Nodes    []reviewNode    `json:"nodes"`
	} `json:"latestReviews"`
	Reviews struct {
		PageInfo graphQLPageInfo `json:"pageInfo"`
		Nodes    []reviewNode    `json:"nodes"`
	} `json:"reviews"`
	ReviewThreads struct {
		PageInfo graphQLPageInfo    `json:"pageInfo"`
		Nodes    []reviewThreadNode `json:"nodes"`
	} `json:"reviewThreads"`
}

type reviewNode struct {
	Author      *graphQLLogin `json:"author"`
	State       string        `json:"state"`
	SubmittedAt string        `json:"submittedAt"`
	Body        string        `json:"body"`
}

type reviewThreadNode struct {
	ID           string `json:"id"`
	IsResolved   bool   `json:"isResolved"`
	IsOutdated   bool   `json:"isOutdated"`
	IsCollapsed  bool   `json:"isCollapsed"`
	Path         string `json:"path"`
	Line         *int   `json:"line"`
	StartLine    *int   `json:"startLine"`
	OriginalLine *int   `json:"originalLine"`
	DiffSide     string `json:"diffSide"`
	Comments     struct {
		PageInfo graphQLPageInfo     `json:"pageInfo"`
		Nodes    []reviewCommentNode `json:"nodes"`
	} `json:"comments"`
}

type reviewCommentNode struct {
	DatabaseID int           `json:"databaseId"`
	Author     *graphQLLogin `json:"author"`
	Body       string        `json:"body"`
	CreatedAt  string        `json:"createdAt"`
	DiffHunk   string        `json:"diffHunk"`
	ReplyTo    *struct {
		DatabaseID int `json:"databaseId"`
	} `json:"replyTo"`
}

// parsePRReviewData maps the GraphQL response into the internal review models.
// It is the unit-tested core (3.5): all null-safety (3.4) lives here — null
// authors, null requested reviewers, Team reviewers, and null thread lines
// (outdated threads) are all handled without panicking.
func parsePRReviewData(respBytes []byte) (*PullRequestReviewData, error) {
	var resp reviewDataResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, err
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL query returned errors: %s", resp.Errors[0].Message)
	}

	node := resp.Data.Repository.PullRequest
	if node == nil {
		return nil, fmt.Errorf("pull request not found")
	}

	result := &PullRequestReviewData{
		ID:          node.ID,
		Number:      node.Number,
		Title:       node.Title,
		Body:        node.Body,
		State:       node.State,
		Author:      loginOf(node.Author),
		BaseRefName: node.BaseRefName,
		HeadRefName: node.HeadRefName,
		HeadRefOid:  node.HeadRefOid,
		BaseRefOid:  node.BaseRefOid,
	}
	if node.BaseRepository != nil {
		result.BaseRepoURL = node.BaseRepository.URL
	}
	if node.HeadRepository != nil {
		result.HeadRepoURL = node.HeadRepository.URL
	}

	truncated := node.Comments.PageInfo.HasNextPage ||
		node.ReviewRequests.PageInfo.HasNextPage ||
		node.LatestReviews.PageInfo.HasNextPage ||
		node.Reviews.PageInfo.HasNextPage ||
		node.ReviewThreads.PageInfo.HasNextPage

	// Global (issue-level) comments.
	for _, c := range node.Comments.Nodes {
		result.IssueComments = append(result.IssueComments, models.IssueComment{
			Author:    loginOf(c.Author),
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
		})
	}

	// Reviewers: latestReviews gives one state-per-author (the badge source);
	// reviewRequests adds requested-but-not-yet-reviewed reviewers as PENDING.
	seenReviewers := map[string]bool{}
	for _, r := range node.LatestReviews.Nodes {
		login := loginOf(r.Author)
		if login == "" || seenReviewers[login] {
			continue
		}
		seenReviewers[login] = true
		result.Reviewers = append(result.Reviewers, models.Reviewer{Login: login, State: r.State})
	}
	for _, rr := range node.ReviewRequests.Nodes {
		// requestedReviewer is null when the reviewer has already reviewed (so
		// they're already covered by latestReviews above).
		if rr.RequestedReviewer == nil {
			continue
		}
		login := rr.RequestedReviewer.Login
		if login == "" {
			// A Team request carries `name` rather than `login`.
			login = rr.RequestedReviewer.Name
		}
		if login == "" || seenReviewers[login] {
			continue
		}
		seenReviewers[login] = true
		result.Reviewers = append(result.Reviewers, models.Reviewer{Login: login, State: "PENDING"})
	}

	// Submitted reviews that carry a summary body, for the Conversation tab's
	// per-reviewer detail. A body-less review (a bare approve/comment) has nothing to
	// show here — the reviewer's state already appears in the Reviewers list.
	for _, r := range node.Reviews.Nodes {
		if strings.TrimSpace(r.Body) == "" {
			continue
		}
		result.Reviews = append(result.Reviews, models.Review{
			Author:      loginOf(r.Author),
			State:       r.State,
			Body:        r.Body,
			SubmittedAt: r.SubmittedAt,
		})
	}

	// Review threads.
	for _, t := range node.ReviewThreads.Nodes {
		if t.Comments.PageInfo.HasNextPage {
			truncated = true
		}

		thread := models.ReviewThread{
			ID:         t.ID,
			Path:       t.Path,
			Side:       t.DiffSide,
			IsResolved: t.IsResolved,
		}

		if t.OriginalLine != nil {
			thread.OriginalLine = *t.OriginalLine
		}

		// A null `line` means the anchored line no longer exists in the current
		// diff: the thread is outdated and must render in the trailing
		// outdated/other section using its diffHunk rather than mis-anchoring.
		if t.Line == nil {
			thread.IsOutdated = true
		} else {
			thread.Line = *t.Line
			thread.IsOutdated = t.IsOutdated
			if t.StartLine != nil {
				thread.StartLine = *t.StartLine
			} else {
				// A null startLine denotes a single-line thread.
				thread.StartLine = *t.Line
			}
		}

		for _, c := range t.Comments.Nodes {
			comment := models.ReviewComment{
				DatabaseID: c.DatabaseID,
				Author:     loginOf(c.Author),
				Body:       c.Body,
				CreatedAt:  c.CreatedAt,
			}
			if c.ReplyTo != nil {
				comment.ReplyToID = c.ReplyTo.DatabaseID
			}
			thread.Comments = append(thread.Comments, comment)
		}

		// The thread's diff context is carried on its root comment's diffHunk;
		// surface it on the thread so the outdated renderer needs only the thread.
		if len(t.Comments.Nodes) > 0 {
			thread.DiffHunk = t.Comments.Nodes[0].DiffHunk
		}

		result.Threads = append(result.Threads, thread)
	}

	// Reflect commenting activity in reviewer states. GitHub keeps a requested reviewer
	// as PENDING even after they leave inline / PR comments (unless they submit a formal
	// review), which misrepresents them as not having engaged. Anyone who authored an
	// inline thread comment or a PR-body comment has effectively commented, so a PENDING
	// reviewer with any such comment is upgraded to COMMENTED. Reviewers with a stronger
	// state (APPROVED / CHANGES_REQUESTED / COMMENTED) are left untouched.
	upgradePendingCommenters(result)

	result.Truncated = truncated

	return result, nil
}

// upgradePendingCommenters flips a reviewer's state from PENDING to COMMENTED when they
// have authored any inline thread comment or PR-body comment.
func upgradePendingCommenters(result *PullRequestReviewData) {
	commenters := map[string]bool{}
	for _, t := range result.Threads {
		for _, c := range t.Comments {
			if c.Author != "" {
				commenters[c.Author] = true
			}
		}
	}
	for _, c := range result.IssueComments {
		if c.Author != "" {
			commenters[c.Author] = true
		}
	}
	for i := range result.Reviewers {
		if result.Reviewers[i].State == "PENDING" && commenters[result.Reviewers[i].Login] {
			result.Reviewers[i].State = "COMMENTED"
		}
	}
}

// loginOf null-safely extracts a login (author can be null for a deleted
// account).
func loginOf(l *graphQLLogin) string {
	if l == nil {
		return ""
	}
	return l.Login
}
