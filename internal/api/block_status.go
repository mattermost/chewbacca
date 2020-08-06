package api

import (
	"fmt"
	"strings"

	"github.com/mattermost/chewbacca/internal/utils"

	log "github.com/sirupsen/logrus"
)

const (
	doNotMerge                  = "do-not-merge"
	doNotMergeAwaitingPR        = "do-not-merge/awaiting-PR"
	doNotMergeAwaitingSubmitter = "do-not-merge/awaiting-submitter-action"
	doNotMergeWIP               = "do-not-merge/work-in-progress"
	wip                         = "WIP"
	releaseNoteLabelNeeded      = "do-not-merge/release-note-label-needed"
)

// checkBlockStatus checks if need to block the PR to be merged
func checkBlockStatus(c *Context, org, repo string, number int) {
	c.Logger = c.Logger.WithFields(log.Fields{
		"number": number,
		"org":    org,
		"repo":   repo,
	})
	c.Logger.Debug("Checking if need to set a merge blocker")

	pr, err := c.GitHub.GetPullRequest(org, repo, number)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to get the PR#%d", number)
		return
	}

	if pr.GetState() == "closed" {
		return
	}

	labels, err := c.GitHub.GetIssueLabels(org, repo, number)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to list labels on PR #%d", number)
	}

	var mergeLabels []string
	if utils.HasLabel(doNotMerge, labels) {
		mergeLabels = append(mergeLabels, doNotMerge)
	}
	if utils.HasLabel(doNotMergeAwaitingPR, labels) {
		mergeLabels = append(mergeLabels, doNotMergeAwaitingPR)
	}
	if utils.HasLabel(doNotMergeAwaitingSubmitter, labels) {
		mergeLabels = append(mergeLabels, doNotMergeAwaitingPR)
	}
	if utils.HasLabel(doNotMergeWIP, labels) {
		mergeLabels = append(mergeLabels, doNotMergeWIP)
	}
	if utils.HasLabel(releaseNoteLabelNeeded, labels) {
		mergeLabels = append(mergeLabels, releaseNoteLabelNeeded)
	}
	if utils.HasLabel(releaseNoteActionRequired, labels) {
		mergeLabels = append(mergeLabels, releaseNoteActionRequired)
	}
	if utils.HasLabel(wip, labels) {
		mergeLabels = append(mergeLabels, wip)
	}

	var desc string
	state := "pending"
	if len(mergeLabels) == 1 {
		desc = fmt.Sprintf(" Should not have %s label.", mergeLabels[0])
	} else if len(mergeLabels) > 1 {
		desc = fmt.Sprintf(" Should not have %s labels.", strings.Join(mergeLabels, ", "))
	} else {
		desc = "Merged allowed."
		state = "success"
	}

	err = c.GitHub.SetStatus(org, repo, pr.GetHead().GetSHA(), state, desc)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to set the status PR#%d", number)
	}
}
