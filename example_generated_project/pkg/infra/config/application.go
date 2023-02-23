package config

import (
	"github.com/spf13/viper"
)

type ApplicationConfig struct {
	ApplicationName string `yaml:"ApplicationName"`
	Newprop         string `yaml:"newprop"`
}

func NewApplicationConfig() (*ApplicationConfig, error) {
	viper.SetConfigFile("config/application.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	config := &ApplicationConfig{}
	err = viper.Unmarshal(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
