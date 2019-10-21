package slk

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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
	config models.Config
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

// Run runs the slk application as configured
func (slk *Slk) Run() {
	api := slack.New(slk.config.APIToken)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		fmt.Print("Event Received: ")
		switch ev := msg.Data.(type) {

		case *slack.HelloEvent:
			// Ignore hello

		case *slack.ConnectedEvent:
			fmt.Println("Infos:", ev.Info)
			fmt.Println("Connection counter:", ev.ConnectionCount)

		case *slack.MessageEvent:
			fmt.Printf("Message: %v\n", ev)

		case *slack.PresenceChangeEvent:
			fmt.Printf("Presence Change: %v\n", ev)

		case *slack.LatencyReport:
			fmt.Printf("Current latency: %v\n", ev.Value)

		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")
			return

		default:
			fmt.Printf("Unhandled event %s\n", msg.Type)
		}
	}

}
