package main

import (
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	BulwarkAuthUrl string
	Port           int
	CORSEnabled    bool
	AllowedOrigins []string
	Domain         string
	DbConnection   string
	DbNameSeed     string
}

func NewAppConfig() (*AppConfig, error) {
	config := &AppConfig{}
	config.BulwarkAuthUrl = getEnv("BULWARK_AUTH_URL", "http://localhost:8080")
	config.Port = getEnvAsInt("PORT", 8080)
	config.DbConnection = getEnv("DB_CONNECTION", "mongodb://localhost:27017/?connect=direct")

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
