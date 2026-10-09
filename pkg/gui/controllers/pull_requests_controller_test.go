package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

func TestReviewCommandArgs(t *testing.T) {
	args, ok := reviewCommandArgs("/bin/lazygit", "/repo", &models.GithubPullRequest{
		Number: 7,
		Url:    "https://github.com/own/rep/pull/7",
	})
	assert.True(t, ok)
	assert.Equal(t, []string{"/bin/lazygit", "-p", "/repo", "own/rep", "7"}, args)

	_, ok = reviewCommandArgs("/bin/lazygit", "/repo", &models.GithubPullRequest{Number: 7})
	assert.False(t, ok)
}
