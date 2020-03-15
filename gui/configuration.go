package main

import "os"
import "path"
import "io/ioutil"
import "encoding/json"

type Configuration struct {
	PortalsDirectory string `json:"portals_directory"`
}

func ConfigDir() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return path.Join(userConfigDir, "portal_patterns"), nil
}
func ConfigPath() (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return path.Join(configDir, "config.json"), nil
}

func LoadConfiguration() *Configuration {
	conf := &Configuration{}
	configPath, err := ConfigPath()
	if err != nil {
		return conf
	}
	file, err := os.Open(configPath)
	if err != nil {
		return conf
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return conf
	}
	json.Unmarshal(bytes, &conf)
	return conf
}

func SaveConfiguration(config *Configuration) {
	configDir, err := ConfigDir()
	if err != nil {
		return
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return
	}
	configPath, err := ConfigPath()
	if err != nil {
		return
	}
	bytes, err := json.Marshal(config)
	if err := ioutil.WriteFile(configPath, bytes, 0644); err != nil {
		panic(err)
	}
}
