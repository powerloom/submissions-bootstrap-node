package utils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
)

func InitLogger() {
	// Check if LOG_FILE environment variable is set
	logFile := os.Getenv("LOG_FILE")
	if logFile != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Printf("Failed to create log directory: %v\n", err)
			// Fall back to stdout/stderr
			logFile = ""
		} else {
			// Open log file
			file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				fmt.Printf("Failed to open log file: %v\n", err)
				// Fall back to stdout/stderr
				logFile = ""
			} else {
				// Write to both file and stdout
				log.SetOutput(io.MultiWriter(file, os.Stdout))
			}
		}
	}

	// If no log file or failed to open, just use stdout
	if logFile == "" {
		log.SetOutput(os.Stdout)
	}

	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	
	// Set log level from environment
	level, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)
}