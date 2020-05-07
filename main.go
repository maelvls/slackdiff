package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/maelvls/slackdiff/pkg/envvar"
	"github.com/maelvls/slackdiff/pkg/logutil"
	"github.com/mattn/go-isatty"
	"github.com/slack-go/slack"
)

var (
	slackToken    = flag.String("token", "", `Slack OAuth token, create one at https://api.slack.com/apps. You can also pass it with SLACK_TOKEN. The --token has priority over SLACK_TOKEN.`)
	onlyUsers     = flag.String("users", "", `Comma-separated list of users to select. By default, shows all users.`)
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

	users, err := api.GetUsers()
	if err != nil {
		logutil.Errorf("%v", err)
		os.Exit(1)
	}

	userMap := make(map[string]slack.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	chans, _, err := api.GetConversations(&slack.GetConversationsParameters{ExcludeArchived: "true"})
	if err != nil {
		logutil.Errorf("%v", err)
		os.Exit(1)
	}

	chanNameToUsersMap := make(map[string][]slack.User, len(chans))
	group := sync.WaitGroup{}
	for _, channel := range chans {
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

	userNameToChansMap := make(map[string][]slack.Channel, len(users))
	for _, user := range users {
		u := user
		group.Add(1)
		go func() {
			defer group.Done()

			chans, _, err := api.GetConversationsForUser(&slack.GetConversationsForUserParameters{ExcludeArchived: true})
			if err != nil {
				logutil.Errorf("can't fetch chans for user %s, got error: %v", u.Name, err)
				return
			}
			userNameToChansMap[u.Name] = chans
		}()
	}
	group.Wait()

	if *onlyUsers == "" {
		printChannelsWithUsers(chans, chanNameToUsersMap)
		return
	}

	onlyList := strings.Split(*onlyUsers, ",")

	selected, err := selectNames(onlyList, users)
	if err != nil {
		logutil.Errorf("%v", err)
		os.Exit(1)
	}

	// tw := tableUsersOverChannels(selected, chans, userNameToChansMap)
	tw := tableChannelsOverUsers(selected, chans, chanNameToUsersMap)
	fmt.Printf("%s\n", tw.Render())

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

func chansToNames(chans []slack.Channel) []string {
	var chanNames []string
	for _, ch := range chans {
		chanNames = append(chanNames, ch.Name)
	}
	return chanNames
}

func strSliceToInterfaceSlice(str []string) []interface{} {
	var res []interface{}
	for _, s := range str {
		res = append(res, s)
	}
	return res
}

func selectNames(names []string, users []slack.User) ([]slack.User, error) {
	var selected []slack.User
	var found []string // just for showing the error

	for _, user := range users {
		if containsStr(names, user.Name) {
			selected = append(selected, user)
		}
	}

	if len(names) != len(selected) {
		return nil, fmt.Errorf("asked %v but only found %v", names, found)
	}

	return selected, nil
}

func containsStr(list []string, str string) bool {
	for _, name := range list {
		if name == str {
			return true
		}
	}
	return false
}

func isInChans(all, sparse []string, yes, no string) []string {
	// Compares two sorted slices and returns a slice that has the same len
	// as the 'all' slice and contains the strings yes or no depending if
	// the sparse slice also contains that value.
	//  all:       [a    b    c    d ]
	//  sparse:    [a         c      ]
	//  returns:   [yes  no   yes  no]
	contains := make([]string, len(all))
	i_sparse := 0
	for i_all := 0; i_all < len(all); i_all++ {
		if i_sparse < len(sparse) && all[i_all] == sparse[i_sparse] {
			i_sparse += 1
			contains[i_all] = yes
		} else {
			contains[i_all] = no
		}
	}
	return contains
}

func tableUsersOverChannels(users []slack.User, chans []slack.Channel, userNameToChansMap map[string][]slack.Channel) table.Writer {
	// table with some amount of customization
	tw := table.NewWriter()
	var header []interface{}
	header = append(header, "username")
	header = append(header, strSliceToInterfaceSlice(chansToNames(chans))...)
	tw.AppendHeader(table.Row{header})
	// append some data rows
	for _, u := range users {
		chans, ok := userNameToChansMap[u.Name]
		if !ok {
			logutil.Errorf("user %s has no channel recorded")
			continue
		}

		var r []interface{}
		r = append(r, u.Name)
		yesOrNoList := isInChans(chansToNames(chans), chansToNames(userNameToChansMap[u.Name]), "✅", "❌")
		r = append(r, strSliceToInterfaceSlice(yesOrNoList)...)
		tw.AppendRow(r)
	}
	// append a footer row
	// tw.AppendFooter(table.Row{"", "Total", 10000})
	// auto-index rows
	tw.SetAutoIndex(true)
	// sort by last name and then by salary
	// tw.SortBy([]table.SortBy{{Name: "Last Name", Mode: table.Dsc}, {Name: "Salary", Mode: table.AscNumeric}})
	// use a ready-to-use style
	tw.SetStyle(table.StyleDefault)
	// customize the style and change some stuff
	tw.Style().Format.Header = text.FormatLower
	tw.Style().Format.Row = text.FormatLower
	tw.Style().Format.Footer = text.FormatLower
	tw.Style().Options.SeparateColumns = false

	return tw
}

func tableChannelsOverUsers(users []slack.User, chans []slack.Channel, chanNameToUsersMap map[string][]slack.User) table.Writer {
	usersStr := usersToNames(users)

	tw := table.NewWriter()

	var header []interface{}
	header = append(header, "channel")
	header = append(header, strSliceToInterfaceSlice(usersStr)...)
	tw.AppendHeader(table.Row(header))

	for _, c := range chans {
		usersForThatChan, ok := chanNameToUsersMap[c.Name]
		if !ok {
			logutil.Errorf("channel %s has no user recorded", c.Name)
			continue
		}

		// Only show this channel if either user is this channel.
		isInChannel := make(map[string]struct{})
		for _, user := range users {
			for _, userForThatChan := range usersForThatChan {
				if user.Name == userForThatChan.Name {
					isInChannel[user.Name] = struct{}{}
				}
			}
		}
		if len(isInChannel) == 0 {
			continue
		}
		if len(isInChannel) == len(users) {
			continue
		}

		var r []interface{}
		r = append(r, c.Name)
		for _, user := range users {
			if _, ok := isInChannel[user.Name]; ok {
				r = append(r, "✅")
			} else {
				r = append(r, "❌")
			}
		}
		tw.AppendRow(r)
	}

	tw.SetAutoIndex(false)

	tw.SetStyle(table.StyleDefault)

	tw.Style().Format.Header = text.FormatLower
	tw.Style().Format.Row = text.FormatLower
	tw.Style().Format.Footer = text.FormatLower
	tw.Style().Options.SeparateColumns = false
	tw.Style().Options.DrawBorder = false

	return tw
}

func printChannelsWithUsers(chans []slack.Channel, chanNameToUsersMap map[string][]slack.User) {
	for _, ch := range chans {
		users := chanNameToUsersMap[ch.Name]
		fmt.Printf("%s: %v\n", logutil.Yel(ch.Name), usersToNames(users))
	}
}
