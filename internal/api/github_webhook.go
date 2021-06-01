package api

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mattermost/chewbacca/model"

	"github.com/google/go-github/v31/github"
	"github.com/gorilla/mux"
)

// initGitHubWebhook registers webhook endpoints on the given router.
func initGitHubWebhook(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	webhooksRouter := apiRouter.PathPrefix("/github_event").Subrouter()
	webhooksRouter.Handle("", addContext(handleReceiveWebhook)).Methods("POST")
}

// handleReceiveWebhook responds to POST /api/github_event, when receive a event from GitHub.
func handleReceiveWebhook(c *Context, w http.ResponseWriter, r *http.Request) {
	buf, _ := ioutil.ReadAll(r.Body)

	receivedHash := strings.SplitN(r.Header.Get("X-Hub-Signature"), "=", 2)
	if receivedHash[0] != "sha1" {
		c.Logger.Error("invalid webhook hash signature: SHA1")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := c.GitHub.ValidateSignature(receivedHash, buf)
	if err != nil {
		c.Logger.WithError(err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var org, repo string
	var number int
	eventType := r.Header.Get("X-GitHub-Event")
	switch eventType {
	case "ping":
		pingEvent := model.PingEventFromJSON(ioutil.NopCloser(bytes.NewBuffer(buf)))
		if pingEvent == nil {
			c.Logger.WithField("hookID", pingEvent.GetHookID()).Info("ping event")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		return
	case "pull_request":
		event := model.PullRequestEventFromJSON(ioutil.NopCloser(bytes.NewBuffer(buf)))
		c.Logger = c.Logger.WithField("pr", event.GetNumber())
		c.Logger.WithField("action", event.GetAction()).Info("pull request event")
		org = event.GetRepo().GetOwner().GetLogin()
		repo = event.GetRepo().GetName()
		number = event.GetNumber()
		handlePullRequestEvent(c, event)
	case "issue_comment":
		event := model.IssueCommentEventFromJSON(ioutil.NopCloser(bytes.NewBuffer(buf)))
		c.Logger = c.Logger.WithField("issue", event.GetIssue().GetNumber())
		c.Logger.Info("issue comment event")
		handleIssueCommentEvent(c, event)
		org = event.GetRepo().GetOwner().GetLogin()
		repo = event.GetRepo().GetName()
		number = event.GetIssue().GetNumber()
		if !event.GetIssue().IsPullRequest() {
			// if not a pull request dont need to set the status
			w.WriteHeader(http.StatusAccepted)
			return
		}
	default:
		c.Logger.Info("other events not implemented")
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	go checkBlockStatus(c, org, repo, number)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}

func handlePullRequestEvent(c *Context, pr *github.PullRequestEvent) {
	handleReleaseNotesPR(c, pr)
}

func handleIssueCommentEvent(c *Context, issueComment *github.IssueCommentEvent) {
	handleReleaseNotesComment(c, issueComment)
	handleCommentLabel(c, issueComment)
}
