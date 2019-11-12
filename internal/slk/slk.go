package slk

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/lk16/slk/internal/models"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

const (
	configBaseName          = ".slk.json"
	configFileExpectedPerms = 0600
)

// Flags contains all flag values
type Flags struct {
	configPath   string
	listUsers    bool
	listChannels bool

	// TODO remove
	tui bool
}

// Slk is the controlling struct of the slk application
type Slk struct {
	flags        Flags
	config       models.Config
	client       *slack.RTM
	userCache    map[string]slack.User
	channelCache map[string]slack.Channel
}

// NewSlk creates a new slk from commandline arguments
func NewSlk(cmdLineArgs []string) (*Slk, error) {

	var flagSet flag.FlagSet
	slk := &Slk{}

	homeFolder, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Wrap(err, "getting home folder failed")
	}

	defaultConfigPath := fmt.Sprintf("%s/%s", homeFolder, configBaseName)

	// TODO option to generate config file skeleton
	flagSet.StringVar(&slk.flags.configPath, "config", defaultConfigPath, "path to configuration file")
	flagSet.BoolVar(&slk.flags.listUsers, "ls-users", false, "list all users and exit")
	flagSet.BoolVar(&slk.flags.listChannels, "ls-channels", false, "list all channels and exit")
	flagSet.BoolVar(&slk.flags.tui, "tui", false, "don't connect to slack, run tui and exit")

	if err := flagSet.Parse(cmdLineArgs); err != nil {
		return nil, errors.Wrap(err, "parsing commandline arguments failed")
	}

	if err := slk.LoadConfigFile(); err != nil {
		return nil, errors.Wrap(err, "config loading")
	}

	return slk, nil
}

// LoadConfigFile loads the config file from disk after checking who can access it.
func (slk *Slk) LoadConfigFile() error {

	fileInfo, err := os.Stat(slk.flags.configPath)
	if err != nil {
		return errors.Wrap(err, "stat error")
	}

	perms := fileInfo.Mode().Perm()
	if perms != configFileExpectedPerms {
		return fmt.Errorf("permission error: expected %#o but found %#o", configFileExpectedPerms, perms)
	}

	configContent, err := ioutil.ReadFile(slk.flags.configPath)
	if err != nil {
		return errors.Wrap(err, "read error")
	}

	err = json.Unmarshal(configContent, &slk.config)
	if err != nil {
		return errors.Wrap(err, "parse error")
	}

	return nil
}

// OnIncomingEvent handles incoming updates from the slack client
func (slk *Slk) OnIncomingEvent(event slack.RTMEvent) {
	var logMsg string

	switch ev := event.Data.(type) {

	case *slack.ConnectedEvent:
		logMsg = fmt.Sprintf("%s connected", ev.Info.User.Name)

	case *slack.MessageEvent:
		logMsg = fmt.Sprintf("#%s %s: %s", slk.ChannelName(ev.Channel), slk.UserName(ev.User), ev.Msg.Text)

	case *slack.PresenceChangeEvent:
		logMsg = fmt.Sprintf("%s is now %s", ev.User, ev.Presence)

	case *slack.RTMError:
		logMsg = ev.Error()

	// this spams the log
	case *slack.LatencyReport:
		return

	// TODO this list is not complete
	// ignored events
	case *slack.ConnectionErrorEvent:
	case *slack.ConnectingEvent:
	case *slack.DisconnectedEvent:
	case *slack.InvalidAuthEvent:
	case *slack.UnmarshallingErrorEvent:
	case *slack.MessageTooLongEvent:
	case *slack.RateLimitEvent:
	case *slack.OutgoingErrorEvent:
	case *slack.IncomingEventError:
	case *slack.AckErrorEvent:
	}

	log.Printf("%16s %s\n", event.Type, logMsg)
}

// LoadChannels loads a map of all channels
func (slk *Slk) LoadChannels() {

	cursor := ""
	iterations := 0

	channels := make([]slack.Channel, 0)

	for cursor != "" || iterations == 0 {
		var channelsChunk []slack.Channel
		var err error
		channelsChunk, cursor, err = slk.client.GetConversations(
			&slack.GetConversationsParameters{
				Cursor:          cursor,
				ExcludeArchived: "true",
				Limit:           1000,
				Types: []string{
					"public_channel",
					"private_channel",
					"im",
					"mpim",
				},
			},
		)

		if err != nil {
			// TODO
		}

		channels = append(channels, channelsChunk...)
		iterations++
	}

	slk.channelCache = make(map[string]slack.Channel, len(channels))
	for _, channel := range channels {
		slk.channelCache[channel.ID] = channel
	}

}

// LoadUsers loads a map of all non-deleted users
func (slk *Slk) LoadUsers() {
	users, err := slk.client.GetUsers()
	if err != nil {
		// TODO
	}

	slk.userCache = make(map[string]slack.User, len(users))
	for _, user := range users {
		if !user.Deleted {
			slk.userCache[user.ID] = user
		}
	}

}

// UserName looks up a username
func (slk *Slk) UserName(code string) string {
	if user, ok := slk.userCache[code]; ok {
		return user.RealName
	}
	return "???"
}

// ChannelName looks up a channelname
func (slk *Slk) ChannelName(code string) string {
	if channel, ok := slk.channelCache[code]; ok {
		return channel.Name
	}
	return "???"
}

func (slk *Slk) listUsers() error {
	slk.LoadUsers()
	for _, user := range slk.userCache {
		fmt.Printf("%30s %40s %30s\n", user.RealName, user.Profile.Email, user.Profile.Title)
	}
	return nil
}

func (slk *Slk) listChannels() error {
	slk.LoadChannels()
	for _, channel := range slk.channelCache {
		if channel.IsMpIM {
			continue
		}
		visibility := "public"
		if channel.IsPrivate {
			visibility = "private"
		}
		fmt.Printf("%40s %8s %4d members\n", channel.Name, visibility, channel.NumMembers)
	}
	return nil
}

// cookieHttpClient implements slack.httpclient
// It sends a cookie on every http request
type cookieHTTPClient struct {
	cookieValue string
}

func (client *cookieHTTPClient) Do(request *http.Request) (*http.Response, error) {

	cookie := &http.Cookie{
		Name:  "d",
		Value: client.cookieValue}

	request.AddCookie(cookie)

	return http.DefaultClient.Do(request)
}

// Run runs the slk application as configured
func (slk *Slk) Run() error {

	if slk.flags.tui {
		tui, err := NewTUI()
		if err != nil {
			return errors.Wrap(err, "tui failed to load")
		}
		tui.Run()

		// TODO make sure program doesn't exit
		for {
		}
	}

	httpClient := &cookieHTTPClient{cookieValue: slk.config.Cookie}

	api := slack.New(slk.config.APIToken, slack.OptionHTTPClient(httpClient))
	slk.client = api.NewRTM()

	if slk.flags.listUsers {
		return slk.listUsers()
	}

	if slk.flags.listChannels {
		return slk.listChannels()
	}

	go slk.client.ManageConnection()

	slk.LoadChannels()
	slk.LoadUsers()

	/*for event := range slk.client.IncomingEvents {
		slk.OnIncomingEvent(event)
	}*/

	// TODO use TUI
	return nil
}
