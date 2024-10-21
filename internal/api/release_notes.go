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
		pr.GetAction() != model.PullRequestActionReopened &&
		pr.GetAction() != model.PullRequestActionEdited &&
		pr.GetAction() != model.PullRequestActionLabeled {
		return
	}

	if pr.GetPullRequest().GetState() == "closed" {
		return
	}

	org := pr.GetRepo().GetOwner().GetLogin()
	repo := pr.GetRepo().GetName()
	number := pr.GetNumber()
	user := pr.GetPullRequest().GetUser().GetLogin()
	branchName := pr.GetPullRequest().GetHead().GetRef()
	repoLabels, err := c.GitHub.ListRepoLabels(org, repo)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to list repo labels on repo #%s", repo)
		return
	}

	repolabelsexisting := sets.String{}
	for _, l := range repoLabels {
		repolabelsexisting.Insert(strings.ToLower(l.GetName()))
	}
	branchList := []string{
		"feat/",
		"fix/",
		"test/",
		"chore/",
		"refactor/",
	}

	branchToLabel := map[string]string{
		"feat/":     "kind/feature",
		"docs/":     "kind/documentation",
		"fix/":      "kind/bug",
		"test/":     "kind/testing",
		"chore/":    "kind/chore",
		"refactor/": "kind/refactor",
	}
	labelsToColours := map[string]string{
		"kind/feature":       "c7def8",
		"kind/documentation": "c7def8",
		"kind/bug":           "e11d21",
		"kind/chore":         "c7def8",
		"kind/refactor":      "c7def8",
		"kind/testing":       "79D6D6",
	}
	labelsToDescriptions := map[string]string{
		"kind/feature":       "Categorizes issue or PR as related to a new feature.",
		"kind/documentation": "Categorizes issue or PR as related to documentation.",
		"kind/bug":           "Categorizes issue or PR as related to a bug.",
		"kind/chore":         "Categorizes issue or PR as related to updates that are not production code.",
		"kind/refactor":      "Categorizes issue or PR as related to refactor of production code.",
		"kind/testing":       "Categorizes issue or PR as related to addition or refactoring of tests.",
	}

	var branchLabels []string
	for _, conventionSubstring := range branchList {
		if strings.Contains(branchName, conventionSubstring) {
			branchLabels = append(branchLabels, branchToLabel[conventionSubstring])
			continue
		}
	}
	if len(branchLabels) > 0 {
		if !repolabelsexisting.Has(branchLabels[0]) {
			err = c.GitHub.CreateLabel(org, repo, buildGhLabel(branchLabels[0], labelsToDescriptions[branchLabels[0]], labelsToColours[branchLabels[0]]))
			if err != nil {
				c.Logger.WithError(err).Error("Failed to create label")
			}
		}
		err = c.GitHub.AddLabels(org, repo, number, branchLabels)
		if err != nil {
			c.Logger.WithError(err).Errorf("failed to add branch labels on PR #%d", number)
			return
		}
	}

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
				err = c.GitHub.CreateComment(org, repo, number, comment)
				if err != nil {
					c.Logger.WithError(err).Error("Failed to create comment")
				}
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
		c.Logger.WithError(err).Error("failed to get the membership")
		return err
	}

	isAuthor := utils.IsAuthor(ic.GetIssue().GetUser().GetLogin(), ic.GetComment().GetUser().GetLogin())

	if !isMember && !isAuthor {
		c.Logger.Info("not member or author")
		format := "you can only set the release note label to %s if you are the PR author or an org member."
		resp := fmt.Sprintf(format, releaseNoteNone)
		c.GitHub.CreateComment(org, repo, number, utils.FormatICResponse(ic.GetComment(), resp))
		return nil
	}

	// Don't allow the /release-note-none command if the release-note block contains a valid release note.
	blockNL := determineReleaseNoteLabel(ic.GetIssue().GetBody(), utils.LabelsSet(ic.GetIssue().Labels))
	if blockNL == releaseNote || blockNL == releaseNoteActionRequired {
		c.Logger.Info("there is a release note already or it is a blocker: %s", blockNL)
		format := "you can only set the release note label to %s if the release-note block in the PR body text is empty or \"none\"."
		resp := fmt.Sprintf(format, releaseNoteNone)
		c.GitHub.CreateComment(org, repo, number, utils.FormatICResponse(ic.GetComment(), resp))
		return nil
	}

	if !utils.HasLabel(releaseNoteNone, ic.GetIssue().Labels) {
		c.Logger.Info("adding relese note none label")
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
