package slk

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/lk16/slk/internal/models"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

const (
	configBaseName          = ".slk.json"
	configFileExpectedPerms = 0600
)

// Slk is the controlling struct of the slk application
type Slk struct {
	config       models.Config
	client       *slack.RTM
	userCache    map[string]slack.User
	channelCache map[string]slack.Channel
}

func getConfigPath(configPathflag string) (string, error) {

	if configPathflag != "" {
		return configPathflag, nil
	}

	homeFolder, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "getting home folder failed")
	}

	paths := []string{
		fmt.Sprintf("./%s", configBaseName),
		fmt.Sprintf("%s/%s", homeFolder, configBaseName)}

	for _, path := range paths {
		_, err := os.Stat(path)
		if err == nil {
			return path, nil
		}
	}

	return "", errors.Wrap(err, "no configuration file found")
}

func crash(err error) {
	fmt.Printf("slk crashed: %s\n", err.Error())
	os.Exit(1)
}

// NewSlk creates a new slk from commandline arguments
func NewSlk(cmdLineArgs []string) *Slk {

	var flagSet flag.FlagSet

	var configPathFlag string
	flagSet.StringVar(&configPathFlag, "config", "", "path to configuration file")

	err := flagSet.Parse(cmdLineArgs)
	if err != nil {
		crash(errors.Wrap(err, "parsing commandline arguments failed"))
		return nil
	}

	configPath, err := getConfigPath(configPathFlag)
	if err != nil {
		crash(errors.Wrap(err, "could not get config path"))
		return nil
	}

	fileInfo, err := os.Stat(configPath)
	if err != nil {
		crash(errors.Wrap(err, "could not stat config path"))
		return nil
	}

	if fileInfo.Mode().Perm() != configFileExpectedPerms {
		crash(fmt.Errorf("expected %s to have perms %#o",
			configPath, configFileExpectedPerms))
		return nil
	}

	configContent, err := ioutil.ReadFile(configPath)
	if err != nil {
		crash(errors.Wrap(err, "config file json parsing error"))
		return nil
	}

	var config models.Config
	err = json.Unmarshal(configContent, &config)
	if err != nil {
		crash(errors.Wrap(err, "json parsing error"))
	}

	if config.APIToken == "" {
		crash(errors.New("empty api token"))
	}

	return &Slk{
		config: config}

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

	// log.Printf("%+v", event.Data)
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

		// TODO
		if err != nil {
			crash(err)
		}

		channels = append(channels, channelsChunk...)
		iterations++
	}

	slk.channelCache = make(map[string]slack.Channel, len(channels))
	for _, channel := range channels {
		slk.channelCache[channel.ID] = channel
	}

	log.Printf("Loaded %d channels", len(channels))
}

// LoadUsers loads a map of all non-deleted users
func (slk *Slk) LoadUsers() {
	users, err := slk.client.GetUsers()
	if err != nil {
		crash(err) // TODO
	}

	slk.userCache = make(map[string]slack.User, len(users))
	for _, user := range users {
		if !user.Deleted {
			slk.userCache[user.ID] = user
		}
	}

	log.Printf("Loaded %d users", len(users))
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

// Run runs the slk application as configured
func (slk *Slk) Run() {
	api := slack.New(slk.config.APIToken)

	slk.client = api.NewRTM()

	go slk.client.ManageConnection()

	slk.LoadChannels()
	slk.LoadUsers()

	for event := range slk.client.IncomingEvents {
		slk.OnIncomingEvent(event)
	}
}
