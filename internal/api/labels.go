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

var (
	defaultLabels          = []string{"kind", "priority"}
	labelRegex             = regexp.MustCompile(`(?m)^/(kind|priority)\s*(.*?)\s*$`)
	removeLabelRegex       = regexp.MustCompile(`(?m)^/remove-(kind|priority)\s*(.*?)\s*$`)
	customLabelRegex       = regexp.MustCompile(`(?m)^/label\s*(.*?)\s*$`)
	customRemoveLabelRegex = regexp.MustCompile(`(?m)^/remove-label\s*(.*?)\s*$`)
)

func handleCommentLabel(c *Context, e *github.IssueCommentEvent) {
	c.Logger.Infof("Starting Label section")
	if model.IssueCommentActionDeleted == e.GetAction() || !e.GetIssue().IsPullRequest() {
		return
	}

	additionalLabels := []string{
		"kind/bug",
		"kind/feature",
		"kind/cleanup",
		"kind/api-change",
		"kind/design",
		"kind/regression",
		"kind/documentation",
		"kind/design",
		"priority/critical-urgent",
		"priority/important-longterm",
		"priority/important-soon",
	}

	commentBody := e.GetComment().GetBody()

	labelMatches := labelRegex.FindAllStringSubmatch(commentBody, -1)
	removeLabelMatches := removeLabelRegex.FindAllStringSubmatch(commentBody, -1)
	customLabelMatches := customLabelRegex.FindAllStringSubmatch(commentBody, -1)
	customRemoveLabelMatches := customRemoveLabelRegex.FindAllStringSubmatch(commentBody, -1)
	if len(labelMatches) == 0 && len(removeLabelMatches) == 0 && len(customLabelMatches) == 0 && len(customRemoveLabelMatches) == 0 {
		return
	}

	org := e.GetRepo().GetOwner().GetLogin()
	repo := e.GetRepo().GetName()
	number := e.GetIssue().GetNumber()

	repoLabels, err := c.GitHub.ListRepoLabels(org, repo)
	if err != nil {
		return
	}
	labels, err := c.GitHub.GetIssueLabels(org, repo, number)
	if err != nil {
		return
	}

	RepoLabelsExisting := sets.String{}
	for _, l := range repoLabels {
		RepoLabelsExisting.Insert(strings.ToLower(l.GetName()))
	}
	var (
		nonexistent         []string
		noSuchLabelsInRepo  []string
		noSuchLabelsOnIssue []string
		labelsToAdd         []string
		labelsToRemove      []string
	)
	// Get labels to add and labels to remove from regexp matches
	labelsToAdd = append(getLabelsFromREMatches(labelMatches), getLabelsFromGenericMatches(customLabelMatches, additionalLabels, &nonexistent)...)
	labelsToRemove = append(getLabelsFromREMatches(removeLabelMatches), getLabelsFromGenericMatches(customRemoveLabelMatches, additionalLabels, &nonexistent)...)
	// Add labels
	for _, labelToAdd := range labelsToAdd {
		if utils.HasLabel(labelToAdd, labels) {
			continue
		}

		if !RepoLabelsExisting.Has(labelToAdd) {
			noSuchLabelsInRepo = append(noSuchLabelsInRepo, labelToAdd)
			continue
		}

		if err = c.GitHub.AddLabels(org, repo, number, []string{labelToAdd}); err != nil {
			c.Logger.WithError(err).Errorf("GitHub failed to add the following label: %s", labelToAdd)
		}
	}

	// Remove labels
	for _, labelToRemove := range labelsToRemove {
		if !utils.HasLabel(labelToRemove, labels) {
			noSuchLabelsOnIssue = append(noSuchLabelsOnIssue, labelToRemove)
			continue
		}

		if !RepoLabelsExisting.Has(labelToRemove) {
			continue
		}

		if err = c.GitHub.RemoveLabel(org, repo, number, labelToRemove); err != nil {
			c.Logger.WithError(err).Errorf("GitHub failed to remove the following label: %s", labelToRemove)
		}
	}

	if len(nonexistent) > 0 {
		c.Logger.Infof("Nonexistent labels: %v", nonexistent)
		msg := fmt.Sprintf("The label(s) `%s` cannot be applied. These labels are supported: `%s`", strings.Join(nonexistent, ", "), strings.Join(additionalLabels, ", "))
		err = c.GitHub.CreateComment(org, repo, number, utils.FormatResponseRaw(e.GetComment().GetBody(), e.GetIssue().GetHTMLURL(), e.GetComment().GetUser().GetLogin(), msg))
		if err != nil {
			c.Logger.WithError(err).Error("Failed to create comment")
		}
		return
	}

	if len(noSuchLabelsInRepo) > 0 {
		c.Logger.Infof("Labels missing in repo: %v", noSuchLabelsInRepo)
		msg := fmt.Sprintf("The label(s) `%s` cannot be applied, because the repository doesn't have them", strings.Join(noSuchLabelsInRepo, ", "))
		err = c.GitHub.CreateComment(org, repo, number, utils.FormatResponseRaw(e.GetComment().GetBody(), e.GetIssue().GetHTMLURL(), e.GetComment().GetUser().GetLogin(), msg))
		if err != nil {
			c.Logger.WithError(err).Error("Failed to create comment")
		}
		return
	}

	// Tried to remove Labels that were not present on the Issue
	if len(noSuchLabelsOnIssue) > 0 {
		msg := fmt.Sprintf("Those labels are not set on the issue: `%v`", strings.Join(noSuchLabelsOnIssue, ", "))
		err = c.GitHub.CreateComment(org, repo, number, utils.FormatResponseRaw(e.GetComment().GetBody(), e.GetIssue().GetHTMLURL(), e.GetComment().GetUser().GetLogin(), msg))
		if err != nil {
			c.Logger.WithError(err).Error("Failed to create comment")
		}
		return
	}

}

func buildGhLabel(name string, description string, color string) github.Label {
	return github.Label{Name: &name, Description: &description, Color: &color}
}

// Get Labels from Regexp matches
func getLabelsFromREMatches(matches [][]string) (labels []string) {
	for _, match := range matches {
		for _, label := range strings.Split(match[0], " ")[1:] {
			label = strings.ToLower(match[1] + "/" + strings.TrimSpace(label))
			labels = append(labels, label)
		}
	}
	return
}

// getLabelsFromGenericMatches returns label matches with extra labels if those
// have been configured in the plugin config.
func getLabelsFromGenericMatches(matches [][]string, additionalLabels []string, invalidLabels *[]string) []string {
	if len(additionalLabels) == 0 {
		return nil
	}
	var labels []string
	labelFilter := sets.String{}
	for _, l := range additionalLabels {
		labelFilter.Insert(strings.ToLower(l))
	}
	for _, match := range matches {
		parts := strings.Split(strings.TrimSpace(match[0]), " ")
		if ((parts[0] != "/label") && (parts[0] != "/remove-label")) || len(parts) != 2 {
			continue
		}
		if labelFilter.Has(strings.ToLower(parts[1])) {
			labels = append(labels, parts[1])
		} else {
			*invalidLabels = append(*invalidLabels, match[0])
		}
	}
	return labels
}
