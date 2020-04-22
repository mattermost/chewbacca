package utils

import (
	"fmt"
	"strings"

	"github.com/google/go-github/v31/github"
)

// AboutThisBotCommands contains the message that links to the commands the bot understand.
const AboutThisBotCommands = "I understand the commands that are listed [here](https://chewbacca.core.cloud.mattermost.com/command-help.html)"

// AboutThisBot contains the text of both AboutThisBotWithoutCommands and AboutThisBotCommands.
const AboutThisBot = AboutThisBotCommands

// FormatSimpleResponse formats a response that does not warrant additional explanation in the
// details section.
func FormatSimpleResponse(to, message string) string {
	format := `@%s: %s

<details>

%s
</details>`

	return fmt.Sprintf(format, to, message, AboutThisBot)
}

// FormatICResponse nicely formats a response to an issue comment.
func FormatICResponse(ic *github.IssueComment, s string) string {
	return FormatResponseRaw(ic.GetBody(), ic.GetHTMLURL(), ic.GetUser().GetLogin(), s)
}

// FormatResponse nicely formats a response to a generic reason.
func FormatResponse(to, message, reason string) string {
	format := `@%s: %s

<details>

%s

%s
</details>`

	return fmt.Sprintf(format, to, message, reason, AboutThisBot)
}

// FormatResponseRaw nicely formats a response for one does not have an issue comment
func FormatResponseRaw(body, bodyURL, login, reply string) string {
	format := `In response to [this](%s):

%s
`
	// Quote the user's comment by prepending ">" to each line.
	var quoted []string
	for _, l := range strings.Split(body, "\n") {
		quoted = append(quoted, ">"+l)
	}
	return FormatResponse(login, reply, fmt.Sprintf(format, bodyURL, strings.Join(quoted, "\n")))
}
