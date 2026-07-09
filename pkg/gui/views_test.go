package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestReviewTabViewZOrder locks the z-order invariant behind the review-mode default
// tabs: views sharing a window are drawn in declaration order and mouse clicks hit the
// topmost, so each review window's DEFAULT tab view must be declared LAST in its stack
// (the same convention as Tags → Remotes → PullRequests → Branches). A regression here
// makes a non-default tab (e.g. Commits) render on top of and swallow clicks meant for
// the default (Conversation) — invisible to focus-based assertions.
func TestReviewTabViewZOrder(t *testing.T) {
	gui := &Gui{}

	idx := map[string]int{}
	for i, mapping := range gui.orderedViewNameMappings() {
		idx[mapping.name] = i
	}

	for _, name := range []string{"prList", "prOverview", "prReview", "prConversation", "prChecks", "prCommits"} {
		assert.Contains(t, idx, name)
	}

	// prContent stack: Files Changed (prReview) is the boot default → last.
	assert.Greater(t, idx["prReview"], idx["prOverview"],
		"prReview (Files Changed, the default prContent tab) must be declared after prOverview")

	// prActivity stack: Conversation is the default → last.
	assert.Greater(t, idx["prConversation"], idx["prCommits"],
		"prConversation (the default prActivity tab) must be declared after prCommits")
	assert.Greater(t, idx["prConversation"], idx["prChecks"],
		"prConversation (the default prActivity tab) must be declared after prChecks")
}
