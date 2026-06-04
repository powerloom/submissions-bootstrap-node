package config

import (
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	PrivateKey                  string
	ConnManagerLowWater         int
	ConnManagerHighWater        int
	PublicIP                    string
	EnableRelayService          bool
	RelayMaxReservations        int
	RelayMaxReservationsPerIP   int
	RelayMaxCircuits            int
	RcmgrMemoryLimitMB          int
	LogPeerConnections          bool
}

func LoadConfig() Config {
	return Config{
		PrivateKey:                os.Getenv("PRIVATE_KEY"),
		ConnManagerLowWater:       getEnvAsInt("CONN_MANAGER_LOW_WATER", 32),
		ConnManagerHighWater:      getEnvAsInt("CONN_MANAGER_HIGH_WATER", 256),
		PublicIP:                  os.Getenv("PUBLIC_IP"),
		EnableRelayService:        getEnvAsBool("ENABLE_RELAY_SERVICE", true),
		RelayMaxReservations:      getEnvAsInt("RELAY_MAX_RESERVATIONS", 256),
		RelayMaxReservationsPerIP: getEnvAsInt("RELAY_MAX_RESERVATIONS_PER_IP", 32),
		RelayMaxCircuits:          getEnvAsInt("RELAY_MAX_CIRCUITS", 8),
		RcmgrMemoryLimitMB:        getEnvAsInt("RCMGR_MEMORY_LIMIT_MB", 512),
		LogPeerConnections:        getEnvAsBool("LOG_PEER_CONNECTIONS", false),
	}
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

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		log.Warnf("Invalid boolean value for environment variable %s: %s. Using default %t", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}
