package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func Load(path string) error {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("json")

	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("unable to read config, %w", err)
	}

	return nil
}
