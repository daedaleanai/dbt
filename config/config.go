package config

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Mirror string
}

var environment map[string]string
var config *Config

const configFileName string = "config.yaml"

func init() {
	environment = make(map[string]string)
	for _, v := range os.Environ() {
		parts := strings.Split(v, "=")
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			environment[key] = value
		}
	}
}

func getDbtConfigDir() (string, error) {
	if dbtConfigDir, ok := environment["DBT_CONFIG_DIR"]; ok {
		return dbtConfigDir, nil
	}

	if xdgConfigHome, ok := environment["XDG_CONFIG_HOME"]; ok {
		return path.Join(xdgConfigHome, "dbt"), nil
	}

	if homeDir, ok := environment["HOME"]; ok {
		return path.Join(path.Join(homeDir, ".config"), "dbt"), nil
	}

	return "", fmt.Errorf("Unable to locate the configuration directory")
}

func loadConfiguration() Config {
	var config Config

	configDir, err := getDbtConfigDir()
	if err != nil {
		log.Debug("Unable to find dbt config directory. Using default configuration\n")
		return config
	}

	configFilePath := path.Join(configDir, configFileName)
	err := yaml.Unmarshal(util.ReadFile(configFilePath), &config)
	if err != nil {
		log.Debug("Error reading configuration file at `%s`: `%s`. Using default configuration\n", configFilePath, err)
		return config
	}

	log.Debug("Loaded configuration from `%s`\n", configFilePath)
	log.Debug("Running with configuration: %+v\n", config)
	return config
}

func GetConfig() Config {
	if config == nil {
		loadedConfig := loadConfiguration()
		config = &loadedConfig
	}

	return *config
}
