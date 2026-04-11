package config

import (
	"github.com/BurntSushi/toml"
)

var Configuration *Config

func LoadConfig() error {
	var cfg Config
	if _, err := toml.DecodeFile("config.toml", &cfg); err != nil {
		return err
	}
	Configuration = &cfg
	return nil
}
