package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v1"
)

type Config struct {
	APIEndpoint   string   `yaml:"api_endpoint"`
	APIKey        string   `yaml:"api_key"`
	ProjectSlug   string   `yaml:"project_slug"`
	ProjectBranch string   `yaml:"project_branch"`
	IgnoredPaths  []string `yaml:"ignored_paths"`
	RawFormat     bool     `yaml:"raw_format"`
}

// Load config from config file
//
func NewConfig(config_data []byte) (*Config, error) {
	config := &Config{
		APIEndpoint:   "https://api.gemnasium.com/v1",
		ProjectBranch: "master",
	}
	yaml.Unmarshal(config_data, config)
	return config, nil

}

func LoadConfigFile(filepath string) (*Config, error) {
	config_data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return NewConfig(config_data)
}
