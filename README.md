# Chewbacca - Github Bot

`Chewbacca` born to help us at Mattermost when need to interact with GitHub.
The name `Chewbacca` was choosen because Chewbacca in Star Wars movies help everybody and this bot will help us :-)

`Chewbacca` is helping today to check and set labels related to release notes. The code was fork from some plugins from https://github.com/kubernetes/test-infra/tree/master/prow which is an awesome Bot for Kubernetes community.

### Installation

To install you can deploy the manifests in the `kubernetes` folder.
But before that please change the secret manifest to add your own secrets and also the ingress manifest to add your own domain.

When this is running you can set your GitHub repo to send the webhooks for `Chewbacca`, this bot need only `issue_comments` and `pull_request` events for now.

Also is good to set, at least, those labels in your repo.

```YAML
labels:
- name: do-not-merge
  description: Should not be merged until this label is removed
  color: a32735
- name: do-not-merge/awaiting-PR
  description: ""
  color: a32735
- name: do-not-merge/release-note-label-needed
  description: ""
  color: e11d21
- name: do-not-merge/work-in-progress
  description: ""
  color: a32735
- name: do-not-merge/awaiting-submitter-action
  description: Blocked on the author
  color: e11d21
- name: kind/api-change
  description: Categorizes issue or PR as related to adding, removing, or otherwise
    changing an API
  color: e11d21
- name: kind/bug
  description: Categorizes issue or PR as related to a bug.
  color: e11d21
- name: kind/cleanup
  description: Categorizes issue or PR as related to cleaning up code, process, or
    technical debt.
  color: c7def8
- name: kind/deprecation
  description: Categorizes issue or PR as related to a feature/enhancement marked
    for deprecation.
  color: e11d21
- name: kind/design
  description: Categorizes issue or PR as related to design.
  color: c7def8
- name: kind/documentation
  description: Categorizes issue or PR as related to documentation.
  color: c7def8
- name: kind/feature
  description: Categorizes issue or PR as related to a new feature.
  color: c7def8
- name: kind/regression
  description: Categorizes issue or PR as related to a regression from a prior release.
  color: e11d21
- name: priority/critical-urgent
  description: Highest priority. Must be actively worked on as someone's top priority
    right now.
  color: e11d21
- name: priority/important-longterm
  description: Important over the long term, but may not be staffed and/or may need
    multiple releases to complete.
  color: eb6420
- name: priority/important-soon
  description: Must be staffed and worked on either currently, or very soon, ideally
    in time for the next release.
  color: eb6420
- name: release-note
  description: Denotes a PR that will be considered when it comes time to generate
    release notes.
  color: c2e0c6
- name: release-note-action-required
  description: Denotes a PR that introduces potentially breaking changes that require
    user action.
  color: c2e0c6
- name: release-note-none
  description: Denotes a PR that doesn't merit a release note.
  color: c2e0c6
```

To apply the labels in your repo you can edit manually or use a tool like https://github.com/cpanato/github-gitlab-labels


### Pull request template

Also is good to set a Pull request template to add the `release-note` section. For that in your repo add the folder `.github` and a file called `PULL_REQUEST_TEMPLATE.md`

We are using this template

```
    <!-- Thank you for contributing a pull request! Here are a few tips to help you:

    1. If this is your first contribution, make sure you've read the Contribution Checklist https://developers.mattermost.com/contribute/getting-started/contribution-checklist/
    2. Read our blog post about "Submitting Great PRs" https://developers.mattermost.com/blog/2019-01-24-submitting-great-prs
    3. Take a look at other repository specific documentation at https://developers.mattermost.com/contribute
    -->

    #### Summary
    <!--
    A description of what this pull request does.
    -->

    #### Ticket Link
    <!--
    If this pull request addresses a Help Wanted ticket, please link the relevant GitHub issue, e.g.

      Fixes https://github.com/mattermost/mattermost-server/issues/XXXXX

    Otherwise, link the JIRA ticket.
    -->

    #### Release Note
    <!--
    If no, just write "NONE" in the release-note block below.
    If yes, a release note is required:
    Enter your extended release note in the block below. If the PR requires additional action from users switching to the new release, include the string "action required".

    -->

    ```release-note

    ```
```

## Release Notes process

### Does my pull request need a release note?

Any user-visible or operator-visible change qualifies for a release note. This
could be a:

- CLI change
- API change
- UI change
- configuration schema change
- behavioral change
- change in non-functional attributes such as efficiency or availability,
  availability of a new platform
- a warning about a deprecation
- fix of a previous _Known Issue_
- fix of a vulnerability (CVE)

No release notes are required for changes to:

- tests
- build infrastructure
- fixes of bugs which have not been released

### Contents of a Release Note

A release note needs a clear, concise description of the change. This includes:

1. an indicator if the pull request _Added_, _Changed_, _Fixed_, _Removed_,
   _Deprecated_ functionality or changed enhancement/feature maturity (alpha,
   beta, stable/GA)
2. an indicator if there is user _Action required_
3. the name of the affected API or configuration fields, CLI commands/flags or
   enhancement/feature
4. a link to relevant user documentation about the enhancement/feature

### Applying a Release Note

To meet this requirement, do one of the following:
- Add notes in the release notes block, or
- Update the release note label

If you don't add release notes in the pull request template, the `do-not-merge/release-note-label-needed` label is added to your pull request automatically after you create it. There are a few ways to update it.

To add a release-note section to the pull request description:

For pull requests with a release note:

    ```release-note
    Your release note here
    ```

For pull requests that require additional action from users switching to the new release, include the string "action required" (case insensitive) in the release note:

    ```release-note
    action required: your release note here
    ```

For pull requests that don't need to be mentioned at release time, use the `/release-note-none` Chewbacca command to add the `release-note-none` label to the PR. You can also write the string "NONE" as a release note in your PR description:

    ```release-note
    NONE
    ```

### Reviewing Release Notes

Reviewing the release notes of a pull request should be a dedicated step in the
overall review process. It is necessary to rely on the same metrics as other
reviewers to be able to distinguish release notes which might need to be
rephrased.

As a guideline, a release notes entry needs to be rephrased if one of the
following cases apply:

- The release note does not communicate the full purpose of the change.
- The release note has no impact on any user.
- The release note is grammatically incorrect.

In any other case the release note should be fine.



*note: this was copy and adapt from [kubernetes/community](https://github.com/kubernetes/community/edit/master/contributors/guide/release-notes.md)*
