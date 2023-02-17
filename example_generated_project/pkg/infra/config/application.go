package config

import (
	"github.com/spf13/viper"
)

type ApplicationConfig struct {
	applicationName string `yaml:"application_name"`
}

func NewApplicationConfig() (*ApplicationConfig, error) {
	viper.SetConfigType("yaml")
	viper.SetConfigName("config/application")
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

func (ac *ApplicationConfig) ApplicationName() string {
	return ac.applicationName
}
