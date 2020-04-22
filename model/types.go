package model

const (
	// PullRequestActionAssigned means assignees were added.
	PullRequestActionAssigned = "assigned"
	// PullRequestActionUnassigned means assignees were removed.
	PullRequestActionUnassigned = "unassigned"
	// PullRequestActionReviewRequested means review requests were added.
	PullRequestActionReviewRequested = "review_requested"
	// PullRequestActionReviewRequestRemoved means review requests were removed.
	PullRequestActionReviewRequestRemoved = "review_request_removed"
	// PullRequestActionLabeled means labels were added.
	PullRequestActionLabeled = "labeled"
	// PullRequestActionUnlabeled means labels were removed
	PullRequestActionUnlabeled = "unlabeled"
	// PullRequestActionOpened means the PR was created
	PullRequestActionOpened = "opened"
	// PullRequestActionEdited means the PR body changed.
	PullRequestActionEdited = "edited"
	// PullRequestActionClosed means the PR was closed (or was merged).
	PullRequestActionClosed = "closed"
	// PullRequestActionReopened means the PR was reopened.
	PullRequestActionReopened = "reopened"
	// PullRequestActionSynchronize means the git state changed.
	PullRequestActionSynchronize = "synchronize"
	// PullRequestActionReadyForReview means the PR is no longer a draft PR.
	PullRequestActionReadyForReview = "ready_for_review"

	// IssueCommentActionCreated means the comment was created.
	IssueCommentActionCreated = "created"
	// IssueCommentActionEdited means the comment was edited.
	IssueCommentActionEdited = "edited"
	// IssueCommentActionDeleted means the comment was deleted.
	IssueCommentActionDeleted = "deleted"
)
