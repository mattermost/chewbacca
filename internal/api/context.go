package api

import (
	"github.com/google/go-github/v31/github"
	"github.com/sirupsen/logrus"
)

// Actions describes the interface for actions.
type Actions interface {
	HandleReleaseNotesPR(c *Context, pr *github.PullRequestEvent)
}

// GitHub describes the interface required to persist changes made via API requests.
type GitHub interface {
	ValidateSignature(receivedHash []string, bodyBuffer []byte) error
	CreateComment(org, repo string, number int, comment string)
	AddLabels(org, repo string, number int, labels []string) error
	RemoveLabel(org, repo string, number int, label string) error
	GetIssueLabels(org, repo string, number int) ([]*github.Label, error)
	ListIssueComments(org, repo string, number int) ([]*github.IssueComment, error)
	GetComments(org, repo string, number int) ([]*github.IssueComment, error)
	IsMember(org, repo string) (bool, error)
	SetStatus(org, repo, sha, state, message string) error
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	ListRepoLabels(org, repo string) ([]*github.Label, error)
}

// Context provides the API with all necessary data and interfaces for responding to requests.
//
// It is cloned before each request, allowing per-request changes such as logger annotations.
type Context struct {
	GitHub    GitHub
	Actions   Actions
	RequestID string
	Logger    logrus.FieldLogger
}

// Clone creates a shallow copy of context, allowing clones to apply per-request changes.
func (c *Context) Clone() *Context {
	return &Context{
		GitHub:  c.GitHub,
		Actions: c.Actions,
		Logger:  c.Logger,
	}
}
