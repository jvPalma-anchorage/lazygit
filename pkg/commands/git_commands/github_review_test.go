package git_commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// sampleReviewDataJSON mirrors the captured cli/cli responses documented in
// openspec/changes/pr-review-mode/docs/github-api.md: a resolved single-line
// thread, an unresolved thread with a reply chain, an outdated thread (null
// line), reviewers from latestReviews, a null requested reviewer, a Team
// request, and a global comment.
const sampleReviewDataJSON = `{
  "data": {
    "repository": {
      "pullRequest": {
        "id": "PR_kwDOABC",
        "number": 9000,
        "title": "Add bundle flag",
        "state": "OPEN",
        "isDraft": false,
        "baseRefName": "trunk",
        "headRefName": "feature",
        "headRefOid": "deadbeefcafe",
        "body": "This is the PR description.",
        "author": { "login": "williammartin" },
        "comments": {
          "pageInfo": { "hasNextPage": false },
          "nodes": [
            { "author": { "login": "steiza" }, "body": "Overall looks good", "createdAt": "2024-04-25T19:55:42Z" }
          ]
        },
        "reviewRequests": {
          "pageInfo": { "hasNextPage": false },
          "nodes": [
            { "requestedReviewer": null },
            { "requestedReviewer": { "__typename": "User", "login": "octocat" } },
            { "requestedReviewer": { "__typename": "Team", "name": "security-team" } }
          ]
        },
        "latestReviews": {
          "nodes": [
            { "author": { "login": "steiza" }, "state": "APPROVED", "submittedAt": "2024-04-25T19:55:42Z" },
            { "author": { "login": "williammartin" }, "state": "COMMENTED", "submittedAt": "2024-04-29T16:07:45Z" }
          ]
        },
        "reviews": {
          "nodes": [
            { "author": { "login": "steiza" }, "state": "APPROVED", "submittedAt": "2024-04-25T19:55:42Z", "body": "workable alternative" }
          ]
        },
        "reviewThreads": {
          "pageInfo": { "hasNextPage": false },
          "nodes": [
            {
              "id": "PRRT_resolved",
              "isResolved": true,
              "isOutdated": false,
              "isCollapsed": false,
              "path": "pkg/cmd/attestation/verify/verify.go",
              "line": 130,
              "startLine": null,
              "originalLine": 130,
              "diffSide": "RIGHT",
              "comments": {
                "pageInfo": { "hasNextPage": false },
                "nodes": [
                  { "databaseId": 1, "author": { "login": "williammartin" }, "body": "Could I interest you in tests", "createdAt": "2024-04-29T14:06:54Z", "diffHunk": "@@ -127,6 +127,7 @@\n+\tcmdutil.X()", "originalLine": 130, "line": 130, "replyTo": null }
                ]
              }
            },
            {
              "id": "PRRT_replychain",
              "isResolved": false,
              "isOutdated": false,
              "isCollapsed": false,
              "path": "pkg/cmd/attestation/verify/verify.go",
              "line": 97,
              "startLine": 95,
              "originalLine": 97,
              "diffSide": "RIGHT",
              "comments": {
                "pageInfo": { "hasNextPage": false },
                "nodes": [
                  { "databaseId": 1577888327, "author": { "login": "steiza" }, "body": "I thought about authHelp()", "createdAt": "2024-04-29T14:00:00Z", "diffHunk": "@@ -95,3 +95,3 @@", "originalLine": 97, "line": 97, "replyTo": null },
                  { "databaseId": 1577925765, "author": { "login": "williammartin" }, "body": "Realistically if we go down this route", "createdAt": "2024-04-29T15:00:00Z", "diffHunk": "@@ -95,3 +95,3 @@", "originalLine": 97, "line": 97, "replyTo": { "databaseId": 1577888327 } }
                ]
              }
            },
            {
              "id": "PRRT_outdated",
              "isResolved": false,
              "isOutdated": true,
              "isCollapsed": false,
              "path": "pkg/cmd/old.go",
              "line": null,
              "startLine": null,
              "originalLine": 42,
              "diffSide": "RIGHT",
              "comments": {
                "pageInfo": { "hasNextPage": false },
                "nodes": [
                  { "databaseId": 2, "author": { "login": "steiza" }, "body": "this changed", "createdAt": "2024-04-20T10:00:00Z", "diffHunk": "@@ -40,3 +40,3 @@\n-\toldLine()", "originalLine": 42, "line": null, "replyTo": null }
                ]
              }
            },
            {
              "id": "PRRT_nullauthor",
              "isResolved": false,
              "isOutdated": false,
              "isCollapsed": false,
              "path": "pkg/cmd/x.go",
              "line": 5,
              "startLine": 5,
              "originalLine": 5,
              "diffSide": "LEFT",
              "comments": {
                "pageInfo": { "hasNextPage": false },
                "nodes": [
                  { "databaseId": 3, "author": null, "body": "from a deleted account", "createdAt": "2024-04-20T10:00:00Z", "diffHunk": "@@ -5 +5 @@", "originalLine": 5, "line": 5, "replyTo": null }
                ]
              }
            }
          ]
        }
      }
    }
  }
}`

func TestParsePRReviewData(t *testing.T) {
	result, err := parsePRReviewData([]byte(sampleReviewDataJSON))
	assert.NoError(t, err)

	// PR identity captured for the add-comment write path.
	assert.Equal(t, "PR_kwDOABC", result.ID)
	assert.Equal(t, "deadbeefcafe", result.HeadRefOid)
	assert.Equal(t, "Add bundle flag", result.Title)
	assert.Equal(t, "This is the PR description.", result.Body)
	assert.Equal(t, "OPEN", result.State)
	assert.False(t, result.Truncated)

	// Global comment.
	assert.Len(t, result.IssueComments, 1)
	assert.Equal(t, "steiza", result.IssueComments[0].Author)
	assert.Equal(t, "Overall looks good", result.IssueComments[0].Body)

	// Reviewers: two from latestReviews, the User request (PENDING), the Team
	// request by name (PENDING); the null requested reviewer is skipped.
	assert.Equal(t, []string{"steiza", "williammartin", "octocat", "security-team"},
		reviewerLogins(result))
	stateByLogin := map[string]string{}
	for _, r := range result.Reviewers {
		stateByLogin[r.Login] = r.State
	}
	assert.Equal(t, "APPROVED", stateByLogin["steiza"])
	assert.Equal(t, "COMMENTED", stateByLogin["williammartin"])
	assert.Equal(t, "PENDING", stateByLogin["octocat"])
	assert.Equal(t, "PENDING", stateByLogin["security-team"])

	assert.Len(t, result.Threads, 4)

	// Resolved single-line thread on the RIGHT side; null startLine collapses to
	// the line number.
	resolved := result.Threads[0]
	assert.Equal(t, "PRRT_resolved", resolved.ID)
	assert.True(t, resolved.IsResolved)
	assert.False(t, resolved.IsOutdated)
	assert.Equal(t, "RIGHT", resolved.Side)
	assert.Equal(t, 130, resolved.Line)
	assert.Equal(t, 130, resolved.StartLine)
	assert.Len(t, resolved.Comments, 1)
	assert.Equal(t, "williammartin", resolved.Comments[0].Author)
	assert.Equal(t, 0, resolved.Comments[0].ReplyToID)

	// Reply chain: root has ReplyToID 0, the reply links back to the root.
	chain := result.Threads[1]
	assert.False(t, chain.IsResolved)
	assert.Equal(t, 95, chain.StartLine)
	assert.Equal(t, 97, chain.Line)
	assert.Len(t, chain.Comments, 2)
	assert.Equal(t, 1577888327, chain.Comments[0].DatabaseID)
	assert.Equal(t, 0, chain.Comments[0].ReplyToID)
	assert.Equal(t, 1577925765, chain.Comments[1].DatabaseID)
	assert.Equal(t, 1577888327, chain.Comments[1].ReplyToID)

	// Outdated thread: null line maps to the outdated bucket, line/startLine 0,
	// diffHunk preserved from the root comment.
	outdated := result.Threads[2]
	assert.True(t, outdated.IsOutdated)
	assert.Equal(t, 0, outdated.Line)
	assert.Equal(t, 0, outdated.StartLine)
	assert.Equal(t, "@@ -40,3 +40,3 @@\n-\toldLine()", outdated.DiffHunk)

	// Null author does not panic and yields an empty login.
	nullAuthor := result.Threads[3]
	assert.Equal(t, "LEFT", nullAuthor.Side)
	assert.Equal(t, "", nullAuthor.Comments[0].Author)
}

func TestParsePRReviewDataTruncation(t *testing.T) {
	json := `{"data":{"repository":{"pullRequest":{
		"id":"x","number":1,"headRefOid":"sha",
		"reviewThreads":{"pageInfo":{"hasNextPage":true},"nodes":[]}}}}}`
	result, err := parsePRReviewData([]byte(json))
	assert.NoError(t, err)
	assert.True(t, result.Truncated)
}

func TestParsePRReviewDataMissingPullRequest(t *testing.T) {
	json := `{"data":{"repository":{"pullRequest":null}}}`
	_, err := parsePRReviewData([]byte(json))
	assert.Error(t, err)
}

func TestParsePRReviewDataGraphQLErrors(t *testing.T) {
	json := `{"errors":[{"message":"Could not resolve to a Repository"}]}`
	_, err := parsePRReviewData([]byte(json))
	assert.Error(t, err)
}

func TestParsePRChangedFiles(t *testing.T) {
	json := `[
	  {"filename":"pkg/cmdutil/auth_check.go","status":"modified","additions":27,"deletions":2,"patch":"@@ -1,16 +1,29 @@ x"},
	  {"filename":"pkg/new.go","status":"added","additions":10,"deletions":0,"patch":"@@ -0,0 +1,10 @@ y"}
	]`
	files, err := parsePRChangedFiles([]byte(json))
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, "pkg/cmdutil/auth_check.go", files[0].Filename)
	assert.Equal(t, "modified", files[0].Status)
	assert.Equal(t, 27, files[0].Additions)
	assert.Equal(t, "@@ -1,16 +1,29 @@ x", files[0].Patch)
	assert.Equal(t, "added", files[1].Status)
}

// TestParsePRReviewDataLocalRefFields covers the Phase 2 query extension (2.2): the
// base OID and the per-side repository URLs (which differ for a fork head) that the
// local-ref fetch needs. A fork PR has distinct base/head repository URLs.
func TestParsePRReviewDataLocalRefFields(t *testing.T) {
	json := `{
	  "data": { "repository": { "pullRequest": {
	    "id": "PR_1", "number": 5, "title": "t", "state": "OPEN", "isDraft": false,
	    "baseRefName": "main", "headRefName": "feature",
	    "headRefOid": "headoid111", "baseRefOid": "baseoid222",
	    "baseRepository": { "url": "https://github.com/owner/repo" },
	    "headRepository": { "url": "https://github.com/forkuser/repo" },
	    "author": { "login": "forkuser" },
	    "comments": { "pageInfo": { "hasNextPage": false }, "nodes": [] },
	    "reviewRequests": { "pageInfo": { "hasNextPage": false }, "nodes": [] },
	    "latestReviews": { "pageInfo": { "hasNextPage": false }, "nodes": [] },
	    "reviews": { "pageInfo": { "hasNextPage": false }, "nodes": [] },
	    "reviewThreads": { "pageInfo": { "hasNextPage": false }, "nodes": [] }
	  } } }
	}`
	result, err := parsePRReviewData([]byte(json))
	assert.NoError(t, err)
	assert.Equal(t, "headoid111", result.HeadRefOid)
	assert.Equal(t, "baseoid222", result.BaseRefOid)
	assert.Equal(t, "https://github.com/owner/repo", result.BaseRepoURL)
	assert.Equal(t, "https://github.com/forkuser/repo", result.HeadRepoURL)
}

func reviewerLogins(result *PullRequestReviewData) []string {
	logins := make([]string, 0, len(result.Reviewers))
	for _, r := range result.Reviewers {
		logins = append(logins, r.Login)
	}
	return logins
}

func TestParsePRReviewDataCapturesReviewBodies(t *testing.T) {
	json := `{
	  "data": { "repository": { "pullRequest": {
	    "id": "PR_1", "number": 5, "title": "t", "state": "OPEN", "isDraft": false,
	    "baseRefName": "main", "headRefName": "feature", "headRefOid": "h", "baseRefOid": "b",
	    "author": { "login": "author" },
	    "comments": { "pageInfo": { "hasNextPage": false }, "nodes": [] },
	    "reviewRequests": { "pageInfo": { "hasNextPage": false }, "nodes": [] },
	    "latestReviews": { "pageInfo": { "hasNextPage": false }, "nodes": [
	      { "author": { "login": "alice" }, "state": "CHANGES_REQUESTED", "submittedAt": "2026-01-01" }
	    ] },
	    "reviews": { "pageInfo": { "hasNextPage": false }, "nodes": [
	      { "author": { "login": "alice" }, "state": "CHANGES_REQUESTED", "submittedAt": "2026-01-01", "body": "please fix the naming" },
	      { "author": { "login": "bob" }, "state": "APPROVED", "submittedAt": "2026-01-02", "body": "" }
	    ] },
	    "reviewThreads": { "pageInfo": { "hasNextPage": false }, "nodes": [] }
	  } } }
	}`
	result, err := parsePRReviewData([]byte(json))
	assert.NoError(t, err)
	// The non-empty review body is captured; the empty-body approval is skipped.
	assert.Len(t, result.Reviews, 1)
	assert.Equal(t, "alice", result.Reviews[0].Author)
	assert.Equal(t, "CHANGES_REQUESTED", result.Reviews[0].State)
	assert.Equal(t, "please fix the naming", result.Reviews[0].Body)
}

func TestParsePRReviewDataUpgradesPendingCommenters(t *testing.T) {
	// jvpalma is a requested reviewer (PENDING) who left an inline comment → COMMENTED.
	// tiago is requested (PENDING) with no comment → stays PENDING. jenny APPROVED and
	// also commented → stays APPROVED (stronger state wins).
	json := `{
	  "data": { "repository": { "pullRequest": {
	    "id": "PR_1", "number": 5, "title": "t", "state": "OPEN", "isDraft": false,
	    "baseRefName": "main", "headRefName": "feature", "headRefOid": "h", "baseRefOid": "b",
	    "author": { "login": "author" },
	    "comments": { "pageInfo": { "hasNextPage": false }, "nodes": [] },
	    "reviewRequests": { "pageInfo": { "hasNextPage": false }, "nodes": [
	      { "requestedReviewer": { "__typename": "User", "login": "jvpalma" } },
	      { "requestedReviewer": { "__typename": "User", "login": "tiago" } }
	    ] },
	    "latestReviews": { "pageInfo": { "hasNextPage": false }, "nodes": [
	      { "author": { "login": "jenny" }, "state": "APPROVED", "submittedAt": "2026-01-01" }
	    ] },
	    "reviews": { "pageInfo": { "hasNextPage": false }, "nodes": [] },
	    "reviewThreads": { "pageInfo": { "hasNextPage": false }, "nodes": [
	      { "id": "T1", "isResolved": false, "isOutdated": false, "isCollapsed": false,
	        "path": "a.go", "line": 5, "startLine": 5, "originalLine": 5, "diffSide": "RIGHT",
	        "comments": { "pageInfo": { "hasNextPage": false }, "nodes": [
	          { "databaseId": 1, "author": { "login": "jvpalma" }, "body": "nit", "createdAt": "x", "diffHunk": "@@", "originalLine": 5, "line": 5, "replyTo": null },
	          { "databaseId": 2, "author": { "login": "jenny" }, "body": "ok", "createdAt": "x", "diffHunk": "@@", "originalLine": 5, "line": 5, "replyTo": null }
	        ] } }
	    ] }
	  } } }
	}`
	result, err := parsePRReviewData([]byte(json))
	assert.NoError(t, err)
	state := map[string]string{}
	for _, r := range result.Reviewers {
		state[r.Login] = r.State
	}
	assert.Equal(t, "COMMENTED", state["jvpalma"], "a PENDING reviewer who commented becomes COMMENTED")
	assert.Equal(t, "PENDING", state["tiago"], "a PENDING reviewer with no comment stays PENDING")
	assert.Equal(t, "APPROVED", state["jenny"], "an APPROVED reviewer who commented stays APPROVED")
}
