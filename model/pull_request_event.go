package model

import (
	"encoding/json"
	"io"

	"github.com/google/go-github/v31/github"
)

// PullRequestEventFromJSON decodes the incomming message to a github.PullRequestEvent
func PullRequestEventFromJSON(data io.Reader) *github.PullRequestEvent {
	decoder := json.NewDecoder(data)
	var event github.PullRequestEvent
	if err := decoder.Decode(&event); err != nil {
		return nil
	}

	return &event
}
