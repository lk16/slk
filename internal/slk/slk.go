package slk

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/lk16/slk/internal/event"
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
	flags  Flags
	config models.Config
	client *slack.RTM
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

// LoadChannels loads a map of all channels
func (slk *Slk) LoadChannels() (map[string]slack.Channel, error) {

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
			return nil, errors.Wrap(err, "loading channels failed")
		}

		channels = append(channels, channelsChunk...)
		iterations++
	}

	channelsMap := make(map[string]slack.Channel, len(channels))
	for _, channel := range channels {
		channelsMap[channel.ID] = channel
	}
	return channelsMap, nil

}

// LoadUsers loads a map of all non-deleted users
func (slk *Slk) LoadUsers() (map[string]slack.User, error) {
	users, err := slk.client.GetUsers()
	if err != nil {
		return nil, errors.Wrap(err, "loading users failed")
	}

	userMap := make(map[string]slack.User, len(users))
	for _, user := range users {
		if !user.Deleted {
			userMap[user.ID] = user
		}
	}
	return userMap, nil

}

func (slk *Slk) listUsers() error {
	users, err := slk.LoadUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		fmt.Printf("%30s %40s %30s\n", user.RealName, user.Profile.Email, user.Profile.Title)
	}
	return nil
}

func (slk *Slk) listChannels() error {
	channels, err := slk.LoadChannels()
	if err != nil {
		return err
	}

	for _, channel := range channels {
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

	tui, err := NewTUI(slk.client)
	if err != nil {
		return errors.Wrap(err, "tui failed to load")
	}

	go func() {
		channels, err := slk.LoadChannels()
		if err != nil {
			tui.Debugf("%s", err.Error())
			return
		}
		tui.Debugf("Loaded %d channels", len(channels))
		tui.SendEvent(event.NewWithID(channels, "slk:list_channels"))

		users, err := slk.LoadUsers()
		if err != nil {
			tui.Debugf("%s", err.Error())
			return
		}
		tui.Debugf("Loaded %d users", len(users))
		tui.SendEvent(event.NewWithID(users, "slk:list_users"))

	}()

	tui.Run()

	return nil
}
