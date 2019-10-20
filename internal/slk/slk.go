package slk

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/lk16/slk/internal/models"
	"github.com/pkg/errors"
)

const (
	configBaseName          = ".slk.json" // TODO use this
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

	path := "./.slk.json"

	_, err := os.Stat(path)
	if err == nil {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "getting home folder failed")
	}

	path = fmt.Sprintf("%s/.slk.json", home)
	if err == nil {
		return path, nil
	}

	return "", errors.Wrap(err, "no configuration file found")
}

// NewSlk creates a new slk from commandline arguments
func NewSlk(cmdLineArgs []string) (*Slk, error) {

	var flagSet flag.FlagSet

	var configPathFlag string
	flagSet.StringVar(&configPathFlag, "config", "", "path to configuration file")

	err := flagSet.Parse(cmdLineArgs)
	if err != nil {
		return nil, errors.Wrap(err, "parsing commandline arguments failed")
	}

	configPath, err := getConfigPath(configPathFlag)
	if err != nil {
		return nil, errors.Wrap(err, "could not get config path")
	}

	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not stat config path")
	}

	if fileInfo.Mode().Perm() != configFileExpectedPerms {
		return nil, errors.Wrapf(err, "expected %s to have perms %#o",
			configPath, configFileExpectedPerms)
	}

	configContent, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "config file json parsing error")
	}

	var config models.Config
	err = json.Unmarshal(configContent, &config)

	// TODO set up other stuff
	slk := &Slk{
		config: config}

	return slk, nil
}
