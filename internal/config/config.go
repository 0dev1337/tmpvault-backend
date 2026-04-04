package config

import (
	"github.com/BurntSushi/toml"
)

var Configuration *Config

func LoadConfig() error {
	if _, err := toml.DecodeFile("config.toml", &Configuration); err != nil {
		return err
	}
	return nil
}
