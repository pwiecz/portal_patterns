package configuration

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type Configuration struct {
	PortalsDirectory string `json:"portals_directory"`
	ProjectDirectory string `json:"project_directory"`
	ExportDirectory  string `json:"export_directory"`
}

func ConfigDir() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userConfigDir, "portal_patterns"), nil
}
func ConfigPath() (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

func LoadConfiguration() (*Configuration, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(configPath); errors.Is(err, fs.ErrNotExist) {
		return &Configuration{}, nil
	}
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	conf := &Configuration{}
	json.Unmarshal(bytes, &conf)
	return conf, nil
}

func SaveConfiguration(config *Configuration) error {
	configDir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	configPath, err := ConfigPath()
	if err != nil {
		return err
	}
	bytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configPath, bytes, 0644); err != nil {
		return err
	}
	return nil
}
