package api

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/chewbacca/internal/utils"
	"github.com/mattermost/chewbacca/model"

	"github.com/google/go-github/v31/github"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// ReleaseNoteLabelNeeded defines the label used when a missing release-note label is blocking the
	// merge.
	ReleaseNoteLabelNeeded    = "do-not-merge/release-note-label-needed"
	releaseNote               = "release-note"
	releaseNoteNone           = "release-note-none"
	releaseNoteActionRequired = "release-note-action-required"
	deprecationLabel          = "kind/deprecation"

	releaseNoteFormat            = `Adding the "%s" label because no release-note block was detected, please follow our [release note process](https://github.com/mattermost/chewbacca#release-notes-process) to remove it.`
	releaseNoteDeprecationFormat = `Adding the "%s" label and removing any existing "%s" label because there is a "%s" label on the PR.`

	actionRequiredNote = "action required"
)

var (
	releaseNoteBody            = fmt.Sprintf(releaseNoteFormat, ReleaseNoteLabelNeeded)
	releaseNoteDeprecationBody = fmt.Sprintf(releaseNoteDeprecationFormat, ReleaseNoteLabelNeeded, releaseNoteNone, deprecationLabel)

	noteMatcherRE = regexp.MustCompile(`(?s)(?:Release note\*\*:\s*(?:<!--[^<>]*-->\s*)?` + "```(?:release-note)?|```release-note)(.+?)```")
	noneRe        = regexp.MustCompile(`(?i)^\W*NONE\W*$`)

	allRNLabels = []string{
		releaseNoteNone,
		releaseNoteActionRequired,
		ReleaseNoteLabelNeeded,
		releaseNote,
	}

	releaseNoteNoneRe = regexp.MustCompile(`(?mi)^/release-note-none\s*$`)
)

func handleReleaseNotesPR(c *Context, pr *github.PullRequestEvent) {
	// Only consider events that edit the PR body or add a label
	if pr.GetAction() != model.PullRequestActionOpened &&
		pr.GetAction() != model.PullRequestActionEdited &&
		pr.GetAction() != model.PullRequestActionLabeled {
		return
	}

	org := pr.GetRepo().GetOwner().GetLogin()
	repo := pr.GetRepo().GetName()
	number := pr.GetNumber()
	user := pr.GetPullRequest().GetUser().GetLogin()

	prInitLabels, err := c.GitHub.GetIssueLabels(org, repo, number)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to list labels on PR #%d", number)
	}

	prLabels := utils.LabelsSet(prInitLabels)

	var comments []*github.IssueComment
	labelToAdd := determineReleaseNoteLabel(pr.GetPullRequest().GetBody(), prLabels)

	if labelToAdd == ReleaseNoteLabelNeeded {
		if prLabels.Has(deprecationLabel) {
			if !prLabels.Has(ReleaseNoteLabelNeeded) {
				comment := utils.FormatSimpleResponse(user, releaseNoteDeprecationBody)
				c.GitHub.CreateComment(org, repo, number, comment)
			}
		} else {
			comments, err = c.GitHub.ListIssueComments(org, repo, number)
			if err != nil {
				c.Logger.WithError(err).Errorf("failed to list comments on %s/%s#%d.", org, repo, number)
				return
			}
			if containsNoneCommand(comments) {
				labelToAdd = releaseNoteNone
			} else if !prLabels.Has(ReleaseNoteLabelNeeded) {
				comment := utils.FormatSimpleResponse(user, releaseNoteBody)
				c.GitHub.CreateComment(org, repo, number, comment)
			}
		}
	}

	// Add the label if needed
	if !prLabels.Has(labelToAdd) {
		c.GitHub.AddLabels(org, repo, number, []string{labelToAdd})
		prLabels.Insert(labelToAdd)
	}

	err = removeOtherLabels(
		func(l string) error {
			return c.GitHub.RemoveLabel(org, repo, number, l)
		},
		labelToAdd,
		allRNLabels,
		prLabels,
	)
	if err != nil {
		c.Logger.WithError(err)
	}

}

func handleReleaseNotesComment(c *Context, ic *github.IssueCommentEvent) error {
	// Only consider PRs and new comments.
	if !ic.GetIssue().IsPullRequest() || ic.GetAction() != model.IssueCommentActionCreated {
		return nil
	}

	org := ic.GetRepo().GetOwner().GetLogin()
	repo := ic.GetRepo().GetName()
	number := ic.GetIssue().GetNumber()

	// Which label does the comment want us to add?
	switch {
	case releaseNoteNoneRe.MatchString(ic.GetComment().GetBody()):
		c.Logger.Info("release note none command match")
	default:
		return nil
	}

	// Only allow authors and org members to add labels.
	isMember, err := c.GitHub.IsMember(org, ic.GetComment().GetUser().GetLogin())
	if err != nil {
		return err
	}

	isAuthor := utils.IsAuthor(ic.GetIssue().GetUser().GetLogin(), ic.GetComment().GetUser().GetLogin())

	if !isMember && !isAuthor {
		format := "you can only set the release note label to %s if you are the PR author or an org member."
		resp := fmt.Sprintf(format, releaseNoteNone)
		c.GitHub.CreateComment(org, repo, number, utils.FormatICResponse(ic.GetComment(), resp))
		return nil
	}

	// Don't allow the /release-note-none command if the release-note block contains a valid release note.
	blockNL := determineReleaseNoteLabel(ic.GetIssue().GetBody(), utils.LabelsSet(ic.GetIssue().Labels))
	if blockNL == releaseNote || blockNL == releaseNoteActionRequired {
		format := "you can only set the release note label to %s if the release-note block in the PR body text is empty or \"none\"."
		resp := fmt.Sprintf(format, releaseNoteNone)
		c.GitHub.CreateComment(org, repo, number, utils.FormatICResponse(ic.GetComment(), resp))
		return nil
	}

	if !utils.HasLabel(releaseNoteNone, ic.GetIssue().Labels) {
		if err := c.GitHub.AddLabels(org, repo, number, []string{releaseNoteNone}); err != nil {
			return err
		}
	}

	labels := sets.String{}
	for _, label := range ic.Issue.Labels {
		labels.Insert(label.GetName())
	}
	// Remove all other release-note-* labels if necessary.
	return removeOtherLabels(
		func(l string) error {
			return c.GitHub.RemoveLabel(org, repo, number, l)
		},
		releaseNoteNone,
		allRNLabels,
		labels,
	)

}

func removeOtherLabels(remover func(string) error, label string, labelSet []string, currentLabels sets.String) error {
	var errs []error
	for _, elem := range labelSet {
		if elem != label && currentLabels.Has(elem) {
			if err := remover(elem); err != nil {
				errs = append(errs, err)
			}
			currentLabels.Delete(elem)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("encountered %d errors setting labels: %v", len(errs), errs)
	}
	return nil
}

func containsNoneCommand(comments []*github.IssueComment) bool {
	for _, c := range comments {
		if releaseNoteNoneRe.MatchString(c.GetBody()) {
			return true
		}
	}
	return false
}

// getReleaseNote returns the release note from a PR body
// assumes that the PR body followed the PR template
func getReleaseNote(body string) string {
	potentialMatch := noteMatcherRE.FindStringSubmatch(body)
	if potentialMatch == nil {
		return ""
	}
	return strings.TrimSpace(potentialMatch[1])
}

// determineReleaseNoteLabel returns the label to be added based on the contents of the 'release-note'
// section of a PR's body text, as well as the set of PR's labels.
func determineReleaseNoteLabel(body string, prLabels sets.String) string {
	composedReleaseNote := strings.ToLower(strings.TrimSpace(getReleaseNote(body)))
	hasNoneNoteInPRBody := noneRe.MatchString(composedReleaseNote)
	hasDeprecationLabel := prLabels.Has(deprecationLabel)

	switch {
	case composedReleaseNote == "" && hasDeprecationLabel:
		return ReleaseNoteLabelNeeded
	case composedReleaseNote == "":
		return ReleaseNoteLabelNeeded
	case hasNoneNoteInPRBody && hasDeprecationLabel:
		return ReleaseNoteLabelNeeded
	case hasNoneNoteInPRBody:
		return releaseNoteNone
	case strings.Contains(composedReleaseNote, actionRequiredNote):
		return releaseNoteActionRequired
	default:
		return releaseNote
	}
}
