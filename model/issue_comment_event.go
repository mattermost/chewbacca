package model

import (
	"encoding/json"
	"io"

	"github.com/google/go-github/v31/github"
)

// IssueCommentEventFromJSON decodes the incomming message to a github.IssueCommentEvent
func IssueCommentEventFromJSON(data io.Reader) *github.IssueCommentEvent {
	decoder := json.NewDecoder(data)
	var event github.IssueCommentEvent
	if err := decoder.Decode(&event); err != nil {
		return nil
	}

	return &event
}
