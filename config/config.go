// config/config.go

package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version  int      `yaml:"version"`
	Settings Settings `yaml:"settings"`
}

type Settings struct {
	WsAddress              string   `yaml:"ws_address"`
	AppID                  uint64   `yaml:"app_id"`
	Token                  string   `yaml:"token"`
	ClientSecret           string   `yaml:"client_secret"`
	TextIntent             []string `yaml:"text_intent"`
	GlobalChannelToGroup   bool     `yaml:"global_channel_to_group"`
	GlobalPrivateToChannel bool     `yaml:"global_private_to_channel"`
	Array                  bool     `yaml:"array"`
}

// LoadConfig 从文件中加载配置
func LoadConfig(path string) (*Config, error) {
	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = yaml.Unmarshal(configData, conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
