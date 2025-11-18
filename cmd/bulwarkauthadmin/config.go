package main

import (
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	BulwarkAuthUrl string
	Port           int
}

func NewAppConfig() (*AppConfig, error) {
	config := &AppConfig{}
	config.BulwarkAuthUrl = getEnv("BULWARK_AUTH_URL", "http://localhost:8080")
	config.Port = 8080

	return config, nil
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsStringSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}
