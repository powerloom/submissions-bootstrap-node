package config

import (
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	PrivateKey           string
	ConnManagerLowWater  int
	ConnManagerHighWater int
	PublicIP             string
}

func LoadConfig() Config {
	return Config{
		PrivateKey:           os.Getenv("PRIVATE_KEY"),
		ConnManagerLowWater:  getEnvAsInt("CONN_MANAGER_LOW_WATER", 20000),
		ConnManagerHighWater: getEnvAsInt("CONN_MANAGER_HIGH_WATER", 50000),
		PublicIP:             os.Getenv("PUBLIC_IP"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Warnf("Invalid integer value for environment variable %s: %s. Using default value %d", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}
