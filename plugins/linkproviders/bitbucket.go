package linkproviders

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/belak/seabird/bot"
	"github.com/belak/seabird/plugins"
	"github.com/belak/seabird/utils"
	"github.com/belak/sorcix-irc"
)

type BitbucketUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type BitbucketRepo struct {
	Scm         string `json:"scm"`
	Description string `json:"description"`
	FullName    string `json:"full_name"`
	Language    string `json:"language"`
	UpdatedOn   string `json:"updated_on"`
}

type BitbucketIssue struct {
	Status       string        `json:"status"`
	Priority     string        `json:"priority"`
	Title        string        `json:"title"`
	ReportedBy   BitbucketUser `json:"reported_by"`
	CommentCount int           `json:"comment_count"`
	CreatedOn    string        `json:"created_on"`
	Metadata     struct {
		Kind string `json:"kind"`
	} `json:"metadata"`
}

type BitbucketPullRequest struct {
	State        string        `json:"state"`
	Title        string        `json:"title"`
	Author       BitbucketUser `json:"author"`
	CommentCount int           `json:"comment_count"`
	CreatedOn    string        `json:"created_on"`
}

var bitbucketUserRegex = regexp.MustCompile(`^/([^/]+)$`)
var bitbucketRepoRegex = regexp.MustCompile(`^/([^/]+)/([^/]+)$`)
var bitbucketIssueRegex = regexp.MustCompile(`^/([^/]+)/([^/]+)/issue/([^/]+)/[^/]+$`)
var bitbucketPullRegex = regexp.MustCompile(`^/([^/]+)/([^/]+)/pull-request/([^/]+)/.*$`)
var bitbucketPrefix = "[Bitbucket]"

func NewBitbucketProvider(p *plugins.URLPlugin) error {
	p.RegisterProvider("bitbucket.org", HandleBitbucket)
	return nil
}

func HandleBitbucket(b *bot.Bot, m *irc.Message, url *url.URL) bool {
	if bitbucketUserRegex.MatchString(url.Path) {
		return bitbucketGetUser(b, m, url)
	} else if bitbucketRepoRegex.MatchString(url.Path) {
		return bitbucketGetRepo(b, m, url)
	} else if bitbucketIssueRegex.MatchString(url.Path) {
		return bitbucketGetIssue(b, m, url)
	} else if bitbucketPullRegex.MatchString(url.Path) {
		return bitbucketGetPull(b, m, url)
	}

	return false
}

func bitbucketGetUser(b *bot.Bot, m *irc.Message, url *url.URL) bool {
	matches := bitbucketUserRegex.FindStringSubmatch(url.Path)
	if len(matches) != 2 {
		return false
	}

	user := matches[1]

	bu := &BitbucketUser{}
	err := utils.JsonRequest(bu, "https://bitbucket.org/api/2.0/users/%s", user)
	if err != nil {
		return false
	}

	// Jay Vana @jsvana
	b.Reply(m, "%s %s (@%s)", bitbucketPrefix, bu.DisplayName, bu.Username)

	return true
}

func bitbucketGetRepo(b *bot.Bot, m *irc.Message, url *url.URL) bool {
	matches := bitbucketRepoRegex.FindStringSubmatch(url.Path)
	if len(matches) != 3 {
		return false
	}

	user := matches[1]
	repo := matches[2]

	br := &BitbucketRepo{}
	err := utils.JsonRequest(br, "https://bitbucket.org/api/2.0/repositories/%s/%s", user, repo)
	if err != nil {
		return false
	}

	// chriskempson/base16-iterm2 [Shell] Last pushed to 15 Nov 2014 - Base16 for iTerm2
	out := br.FullName
	if br.Language != "" {
		out += " [" + br.Language + "]"
	}
	tm, err := time.Parse(time.RFC3339, br.UpdatedOn)
	if err != nil {
		return false
	}
	out += " Last pushed to " + tm.Format("2 Jan 2006")
	b.Reply(m, "%s %s", bitbucketPrefix, out)

	return true
}

func bitbucketGetIssue(b *bot.Bot, m *irc.Message, url *url.URL) bool {
	matches := bitbucketIssueRegex.FindStringSubmatch(url.Path)
	if len(matches) != 4 {
		return false
	}

	user := matches[1]
	repo := matches[2]
	issueNum := matches[3]

	bi := &BitbucketIssue{}
	err := utils.JsonRequest(bi, "https://bitbucket.org/api/1.0/repositories/%s/%s/issues/%s", user, repo, issueNum)
	if err != nil {
		return false
	}

	// Issue #51 on belak/seabird [open] - Expand issues plugin with more of Bitbucket [created 3 Jan 2015]
	out := fmt.Sprintf("Issue #%s on %s/%s [%s]", issueNum, user, repo, bi.Status)
	if bi.Priority != "" && bi.Metadata.Kind != "" {
		out += " [" + bi.Priority + " - " + bi.Metadata.Kind + "]"
	}
	out += " by " + bi.ReportedBy.Username
	if bi.Title != "" {
		out += " - " + bi.Title
	}
	tm, err := time.Parse("2006-01-02T15:04:05.000", bi.CreatedOn)
	if err != nil {
		return false
	}
	out += " [created " + tm.Format("2 Jan 2006") + "]"
	b.Reply(m, "%s %s", bitbucketPrefix, out)

	return true
}

func bitbucketGetPull(b *bot.Bot, m *irc.Message, url *url.URL) bool {
	matches := bitbucketPullRegex.FindStringSubmatch(url.Path)
	if len(matches) != 4 {
		return false
	}

	user := matches[1]
	repo := matches[2]
	pullNum := matches[3]

	bpr := &BitbucketPullRequest{}
	err := utils.JsonRequest(bpr, "https://bitbucket.org/api/2.0/repositories/%s/%s/pullrequests/%s", user, repo, pullNum)
	if err != nil {
		return false
	}

	// Pull request #59 on belak/seabird created by jsvana [open] - Add stuff to links [created 4 Jan 2015]
	out := fmt.Sprintf("Pull request #%s on %s/%s created by %s [%s]", pullNum, user, repo, bpr.Author.Username, strings.ToLower(bpr.State))
	if bpr.Title != "" {
		out += " - " + bpr.Title
	}
	tm, err := time.Parse("2006-01-02T15:04:05.000000-07:00", bpr.CreatedOn)
	if err != nil {
		return false
	}
	out += " [created " + tm.Format("2 Jan 2006") + "]"
	b.Reply(m, "%s %s", bitbucketPrefix, out)

	return true
}
