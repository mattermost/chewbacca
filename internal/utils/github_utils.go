package utils

import (
	"strings"

	"github.com/google/go-github/v31/github"
	"k8s.io/apimachinery/pkg/util/sets"
)

// HasLabel checks if label is in the label set "issueLabels".
func HasLabel(label string, issueLabels []*github.Label) bool {
	for _, l := range issueLabels {
		if strings.ToLower(l.GetName()) == strings.ToLower(label) {
			return true
		}
	}
	return false
}

// HasLabels checks if all labels are in the github.label set "issueLabels".
func HasLabels(labels []string, issueLabels []*github.Label) bool {
	for _, label := range labels {
		if !HasLabel(label, issueLabels) {
			return false
		}
	}
	return true
}

// IsAuthor checks if a user is the author of the issue.
func IsAuthor(issueUser, commentUser string) bool {
	return NormLogin(issueUser) == NormLogin(commentUser)
}

// NormLogin normalizes GitHub login strings
func NormLogin(login string) string {
	return strings.TrimPrefix(strings.ToLower(login), "@")
}

// LabelsSet create a label set based on the github labels to make easier the manipulation
func LabelsSet(labels []*github.Label) sets.String {
	prLabels := sets.String{}
	for _, label := range labels {
		prLabels.Insert(label.GetName())
	}
	return prLabels
}
