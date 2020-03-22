package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/maelvls/slack-missing-chan/pkg/envvar"
	"github.com/maelvls/slack-missing-chan/pkg/logutil"
	"github.com/mattn/go-isatty"
	"github.com/slack-go/slack"
)

var (
	slackToken    = flag.String("token", "", `Slack OAuth token, create one at https://api.slack.com/apps. You can also pass it with SLACK_TOKEN. The --token has priority over SLACK_TOKEN.`)
	slackTokenEnv = envvar.Getenv("SLACK_TOKEN", "Slack OAuth token, see https://api.slack.com/apps.")

	showVersion = flag.Bool("version", false, "Watch out, returns 'n/a (commit none, built on unknown)' when built with 'go get'.")
	// The 'version' var is set during build, using something like:
	//  go build  -ldflags"-X main.version=$(git describe --tags)".
	// Note: "version", "commit" and "date" are set automatically by
	// goreleaser.
	version = "n/a"
	commit  = "none"
	date    = "unknown"
)

func main() {
	color.NoColor = (os.Getenv("TERM") == "dumb") ||
		(!isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()))

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "Environment variables:\n%s", envvar.Usage())
	}
	flag.Parse()
	envvar.Parse()

	if *showVersion {
		fmt.Printf("%s (commit %s, built on %s)\n", version, commit, date)
		return
	}

	if *slackTokenEnv == "" && *slackToken == "" {
		logutil.Errorf("either use --token or set SLACK_TOKEN")
		os.Exit(124)
	}
	token := *slackTokenEnv
	if *slackToken != "" {
		token = *slackToken
	}

	api := slack.New(token)

	userList, err := api.GetUsers()
	if err != nil {
		logutil.Errorf("%v", err)
		os.Exit(1)
	}

	userMap := make(map[string]slack.User, len(userList))
	for _, u := range userList {
		userMap[u.ID] = u
	}

	conversations, _, err := api.GetConversations(&slack.GetConversationsParameters{ExcludeArchived: "true"})
	if err != nil {
		logutil.Errorf("%v", err)
		os.Exit(1)
	}

	chanNameToUsersMap := make(map[string][]slack.User, len(conversations))

	group := sync.WaitGroup{}
	for _, channel := range conversations {
		ch := channel
		group.Add(1)
		go func() {
			defer group.Done()

			users, _, err := api.GetUsersInConversation(&slack.GetUsersInConversationParameters{ChannelID: ch.ID})
			if err != nil {
				logutil.Errorf("can't fetch users for channel %s, got error: %v", ch.Name, err)
				return
			}
			chanNameToUsersMap[ch.Name] = idsToUsers(users, userMap)
		}()
	}
	group.Wait()

	for _, ch := range conversations {
		users := chanNameToUsersMap[ch.Name]
		fmt.Printf("%s: %v\n", logutil.Yel(ch.Name), usersToNames(users))
	}

	if err != nil {
		logutil.Errorf("%v", err)
		os.Exit(1)
	}
}

func idsToUsers(ids []string, mapping map[string]slack.User) []slack.User {
	var users []slack.User
	for _, id := range ids {
		user, ok := mapping[id]
		if !ok {
			logutil.Debugf("user id %s not found", id)
			continue
		}

		users = append(users, user)
	}
	return users
}

func usersToNames(users []slack.User) []string {
	var names []string
	for _, user := range users {
		names = append(names, user.Name)
	}
	return names
}
