package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	PrivateKey string `mapstructure:"PRIVATE_KEY"`
}

func LoadConfig() (config Config) {
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	v, err := viper.UnmarshalE(&config)
	if err != nil {
		// Log the error but don't fatal, as we'll handle missing key in main.go
		// This allows the application to proceed if the .env file is missing but env vars are set.
		// Or if the PRIVATE_KEY is simply not set, which is handled by main.go
		// For now, we just return the default config struct.
		return
	}
	return v
}