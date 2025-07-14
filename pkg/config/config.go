package config

import (
	"os"
)

type Config struct {
	PrivateKey string
}

func LoadConfig() Config {
	return Config{
		PrivateKey: os.Getenv("PRIVATE_KEY"),
	}
}