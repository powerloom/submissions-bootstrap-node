package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	PrivateKey string `mapstructure:"PRIVATE_KEY"`
}

func LoadConfig() (config Config, err error) {
	viper.AddConfigPath(".")
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
