package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DBUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configFileName), nil
}

func write(cfg Config) error {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}
	configJson, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	err = os.WriteFile(configFilePath, configJson, 0644)
	if err != nil {
		return err
	}
	return nil
}

func Read() (Config, error) {
	userConfigPath, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}
	userConfig, err := os.ReadFile(userConfigPath)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(userConfig, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func (c *Config) SetUser(userName string) error {
	c.CurrentUserName = userName
	return write(*c)
}
