package context

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSamplePullRequests(t *testing.T) {
	prs := samplePullRequests()

	assert.NotEmpty(t, prs, "stub source should return sample PRs")

	for _, pr := range prs {
		assert.True(t, strings.HasPrefix(pr.Title, "[SAMPLE]"),
			"sample PR titles must be clearly marked as stub data, got %q", pr.Title)
		assert.NotZero(t, pr.Number, "sample PR should have a number")
		assert.NotEmpty(t, pr.State, "sample PR should have a state")
	}
}
