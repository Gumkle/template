package config

import "github.com/spf13/viper"

type CategoryConfig struct {
	Timebetweenupdates time.Duration `yaml:"timebetweenupdates"`
}

// NewCategoryConfig unmarshalls yaml data to struct and returns a pointer to it
func NewCategoryConfig() (*CategoryConfig, error) {
	viper.SetConfigFile("config/category.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	config := &CategoryConfig{}
	err = viper.Unmarshal(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
