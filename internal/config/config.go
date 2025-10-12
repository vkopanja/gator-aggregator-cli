// Package config is used for configuring the CLI app
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func getConfigFilePath() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", userHome, configFileName), nil
}

func write(cfg Config) error {
	configFile, err := getConfigFilePath()
	if err != nil {
		return fmt.Errorf("failed writing to config file: %s", err)
	}
	confBytes, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed marshaling config file: %s", err)
	}
	return os.WriteFile(configFile, confBytes, 0644)
}

func Read() (*Config, error) {
	configPath, err := getConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed reading config dir: %s", err)
	}
	confBytes, err := os.ReadFile(fmt.Sprintf(configPath))
	if err != nil {
		return nil, fmt.Errorf("failed reading gatorconfig: %s", err)
	}
	var conf *Config
	if err := json.Unmarshal(confBytes, &conf); err != nil {
		return nil, fmt.Errorf("failed unmarshaling config file: %s", err)
	}
	return conf, nil
}

func (c *Config) SetUser(userName string) error {
	c.CurrentUserName = userName
	if err := write(*c); err != nil {
		return err
	}
	return nil
}
