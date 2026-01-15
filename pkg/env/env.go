// Package env provides helper functions for reading environment variables with
// default values and type conversion. These utilities eliminate code duplication
// across services while providing consistent error handling and type safety.
//
// Example usage:
//
//	port := env.GetString("PORT", "8080")
//	timeout := env.GetDuration("TIMEOUT", 30*time.Second)
//	maxRetries := env.GetInt("MAX_RETRIES", 3)
//	enableDebug := env.GetBool("DEBUG", false)
//
// For required variables:
//
//	apiKey := env.MustGetString("API_KEY") // panics if not set
package env

import (
	"os"
	"strconv"
	"time"
)

// GetString returns the environment variable value or default if not set or empty.
func GetString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetInt returns the environment variable as int or default if not set, empty, or invalid.
// If the value cannot be parsed as an integer, returns the default value.
func GetInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetBool returns the environment variable as bool or default if not set, empty, or invalid.
// Accepts: 1, t, T, true, TRUE, True, 0, f, F, false, FALSE, False.
// If the value cannot be parsed as a boolean, returns the default value.
func GetBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// GetDuration returns the environment variable as duration or default if not set, empty, or invalid.
// Accepts duration strings like "300ms", "1.5h", "2h45m".
// If the value cannot be parsed as a duration, returns the default value.
func GetDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// MustGetString returns the environment variable value or panics if not set or empty.
// Use this for required configuration that must be present at startup.
func MustGetString(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("required environment variable not set: " + key)
	}
	return value
}
