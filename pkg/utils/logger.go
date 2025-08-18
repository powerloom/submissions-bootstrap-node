package utils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

func InitLogger() {
	// Set log level from environment
	level, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)
	
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	
	// Check if LOG_FILE environment variable is set
	logFile := os.Getenv("LOG_FILE")
	if logFile != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Printf("Failed to create log directory: %v\n", err)
			// Fall back to stdout
			log.SetOutput(os.Stdout)
			return
		}
		
		// Parse rotation settings from environment
		maxSize := 100 // Default 100MB
		if val := os.Getenv("LOG_MAX_SIZE_MB"); val != "" {
			if size, err := strconv.Atoi(val); err == nil {
				maxSize = size
			}
		}
		
		maxBackups := 5 // Default keep 5 old files
		if val := os.Getenv("LOG_MAX_BACKUPS"); val != "" {
			if backups, err := strconv.Atoi(val); err == nil {
				maxBackups = backups
			}
		}
		
		maxAge := 30 // Default 30 days
		if val := os.Getenv("LOG_MAX_AGE_DAYS"); val != "" {
			if age, err := strconv.Atoi(val); err == nil {
				maxAge = age
			}
		}
		
		compress := false
		if val := os.Getenv("LOG_COMPRESS"); val == "true" || val == "1" {
			compress = true
		}
		
		// Create rotating log writer
		rotator := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    maxSize,    // megabytes
			MaxBackups: maxBackups, // number of old files
			MaxAge:     maxAge,     // days
			Compress:   compress,   // compress old files
			LocalTime:  true,
		}
		
		// Write to both rotating file and stdout
		multiWriter := io.MultiWriter(rotator, os.Stdout)
		log.SetOutput(multiWriter)
		
		log.WithFields(log.Fields{
			"file":       logFile,
			"maxSize":    maxSize,
			"maxBackups": maxBackups,
			"maxAge":     maxAge,
			"compress":   compress,
		}).Info("Initialized rotating logger")
	} else {
		// If no log file, just use stdout
		log.SetOutput(os.Stdout)
		log.Info("Initialized console logger")
	}
}