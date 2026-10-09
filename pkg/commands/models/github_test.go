package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGithubPullRequestBaseRepo(t *testing.T) {
	scenarios := []struct {
		url           string
		expectedOwner string
		expectedRepo  string
		expectedOk    bool
	}{
		{"https://github.com/own/rep/pull/7", "own", "rep", true},
		{"https://github.example.com/own/rep/pull/7", "own", "rep", true},
		{"https://github.com/own/rep", "", "", false},
		{"", "", "", false},
	}

	for _, s := range scenarios {
		t.Run(s.url, func(t *testing.T) {
			owner, repo, ok := (&GithubPullRequest{Url: s.url}).BaseRepo()
			assert.Equal(t, s.expectedOwner, owner)
			assert.Equal(t, s.expectedRepo, repo)
			assert.Equal(t, s.expectedOk, ok)
		})
	}
}
