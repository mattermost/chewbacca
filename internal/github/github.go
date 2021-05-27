package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"

	"github.com/google/go-github/v31/github"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// GHClient set the configuration needed.
type GHClient struct {
	GitHubClient *github.Client
	GitHubSecret string
	logger       log.FieldLogger
}

// NewGithubClient creates a new GitHub client.
func NewGithubClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	return github.NewClient(tc)
}

// NewGitHubConfig creates a new KopsProvisioner.
func NewGitHubConfig(gitHubToken, gitHubSecret string, logger log.FieldLogger) *GHClient {
	return &GHClient{
		GitHubClient: NewGithubClient(gitHubToken),
		GitHubSecret: gitHubSecret,
		logger:       logger,
	}
}

// ValidateSignature validate the incoming github event.
func (g *GHClient) ValidateSignature(receivedHash []string, bodyBuffer []byte) error {
	hash := hmac.New(sha1.New, []byte(g.GitHubSecret))
	if _, err := hash.Write(bodyBuffer); err != nil {
		msg := fmt.Sprintf("Cannot compute the HMAC for request: %s\n", err)
		return errors.New(msg)
	}

	expectedHash := hex.EncodeToString(hash.Sum(nil))
	if receivedHash[1] != expectedHash {
		msg := fmt.Sprintf("Expected Hash does not match the received hash: %s\n", expectedHash)
		return errors.New(msg)
	}

	return nil
}

// CreateComment sends a GitHub Comment to a specific issue/pull request.
func (g *GHClient) CreateComment(org, repo string, number int, comment string) error {
	g.logger.WithField("comment", comment).Debug("Sending GitHub comment")
	_, _, err := g.GitHubClient.Issues.CreateComment(context.Background(), org, repo, number, &github.IssueComment{Body: &comment})
	if err != nil {
		return errors.Wrap(err, "Failed to send GitHub comment")
	}

	return nil
}

// CreateLabel creates a GitHub label to a specific repository if it doesn't exist.
func (g *GHClient) CreateLabel(org, repo string, label github.Label) error {
	g.logger.WithField("labels", label).Debug("Creating GitHub label")
	_, _, err := g.GitHubClient.Issues.CreateLabel(context.Background(), org, repo, &label)
	if err != nil {
		return errors.Wrap(err, "Failed to create GitHub label")
	}

	return nil
}

// AddLabels adds a GitHub label to a specific issue/pull request.
func (g *GHClient) AddLabels(org, repo string, number int, labels []string) error {
	g.logger.WithField("labels", labels).Debug("Setting GitHub label")
	_, _, err := g.GitHubClient.Issues.AddLabelsToIssue(context.Background(), org, repo, number, labels)
	if err != nil {
		return errors.Wrap(err, "Failed to set GitHub labels")
	}

	return nil
}

// RemoveLabel remove a GitHub label from a specific issue/pull request.
func (g *GHClient) RemoveLabel(org, repo string, number int, label string) error {
	g.logger.WithField("label", label).Debug("Removing GitHub label")
	_, err := g.GitHubClient.Issues.RemoveLabelForIssue(context.Background(), org, repo, number, label)
	if err != nil {
		return errors.Wrap(err,"Failed to set GitHub labels")
	}

	return nil
}

// GetComments get comments a specific issue/pull request.
func (g *GHClient) GetComments(org, repo string, number int) ([]*github.IssueComment, error) {
	g.logger.WithFields(log.Fields{
		"number":    number,
		"org":       org,
		"repo_name": repo,
	}).Debug("Getting GitHub comment")
	comments, _, err := g.GitHubClient.Issues.ListComments(context.Background(), org, repo, number, nil)
	if err != nil {
		errors.Wrap(err,"Failed to set GitHub labels")
	}
	return comments, nil
}

// GetIssueLabels get all the labels for a specific issue/pull request.
func (g *GHClient) GetIssueLabels(org, repo string, number int) ([]*github.Label, error) {
	g.logger.WithFields(log.Fields{
		"number":    number,
		"org":       org,
		"repo_name": repo,
	}).Debug("Getting GitHub issue label")

	labels, _, err := g.GitHubClient.Issues.ListLabelsByIssue(context.Background(), org, repo, number, nil)
	if err != nil {
		errors.Wrap(err,"Failed to set GitHub labels")
	}
	return labels, nil
}

// ListIssueComments get all the comment for a specific issue/pull request.
func (g *GHClient) ListIssueComments(org, repo string, number int) ([]*github.IssueComment, error) {
	g.logger.WithFields(log.Fields{
		"number":    number,
		"org":       org,
		"repo_name": repo,
	}).Debug("Getting GitHub issue label")

	comments, _, err := g.GitHubClient.Issues.ListComments(context.Background(), org, repo, number, nil)
	if err != nil {
		errors.Wrap(err,"Failed to set GitHub labels")
	}
	return comments, nil
}

// IsMember check if a user is member of the org
func (g *GHClient) IsMember(org, user string) (bool, error) {
	g.logger.WithFields(log.Fields{
		"org":  org,
		"user": user,
	}).Debug("Checking user membership")

	if org == user {
		// Make it possible to run a couple of plugins on personal repos.
		return true, nil
	}

	member, resp, err := g.GitHubClient.Organizations.GetOrgMembership(context.Background(), user, org)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == 200 && member.GetState() == "active" {
		return true, nil
	} else if resp.StatusCode == 204 && member.GetState() == "active" {
		return true, nil
	} else if resp.StatusCode == 404 {
		return false, nil
	} else if resp.StatusCode == 302 {
		return false, fmt.Errorf("requester is not %s org member", org)
	}

	//should not reach here
	return false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
}

// SetStatus set the PR status
func (g *GHClient) SetStatus(org, repo, sha, state, message string) error {
	g.logger.WithFields(log.Fields{
		"org":     org,
		"repo":    repo,
		"sha":     sha,
		"state":   state,
		"message": message,
	}).Debug("Setting status")

	mergeStatus := &github.RepoStatus{
		Context:     github.String("blocker"),
		State:       github.String(state),
		Description: github.String(message),
		TargetURL:   github.String(""),
	}

	_, _, err := g.GitHubClient.Repositories.CreateStatus(context.Background(), org, repo, sha, mergeStatus)
	if err != nil {
		return	errors.Wrap(err,"Unable to create the github status for for PR")
	}

	return nil
}

// GetPullRequest get a Pull Request
func (g *GHClient) GetPullRequest(org, repo string, number int) (*github.PullRequest, error) {
	g.logger.WithFields(log.Fields{
		"org":    org,
		"repo":   repo,
		"number": number,
	}).Debug("Getting Pull Request")

	pr, _, err := g.GitHubClient.PullRequests.Get(context.Background(), org, repo, number)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get the pull request")
	}

	return pr, nil
}

// ListRepoLabels list all labels for a repo
func (g *GHClient) ListRepoLabels(org, repo string) ([]*github.Label, error) {
	g.logger.WithFields(log.Fields{
		"org":  org,
		"repo": repo,
	}).Debug("Getting Repo labels")

	var allLabels []*github.Label

	opt := &github.ListOptions{
		PerPage: 50,
	}

	for {
		labels, resp, err := g.GitHubClient.Issues.ListLabels(context.Background(), org, repo, opt)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to get the pull request")
		}

		allLabels = append(allLabels, labels...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allLabels, nil

}
