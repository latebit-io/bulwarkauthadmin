package main

import (
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	BulwarkAuthUrl       string
	Port                 int
	CORSEnabled          bool
	AllowedOrigins       []string
	Domain               string
	DbConnection         string
	DbNameSeed           string
	AdminAccount         string
	AdminAccountPassword string
}

func NewAppConfig() (*AppConfig, error) {
	config := &AppConfig{}
	config.CORSEnabled = getEnvAsBool("CORS_ENABLED", false)
	config.BulwarkAuthUrl = getEnv("BULWARK_AUTH_URL", "http://localhost:5173")
	config.Port = getEnvAsInt("PORT", 8080)
	config.DbConnection = getEnv("DB_CONNECTION", "mongodb://localhost:27017/?connect=direct")
	config.AllowedOrigins = getEnvAsStringSlice("ALLOWED_WEB_ORIGINS", []string{"http://localhost:5173"})
	// Security alert: this should be removed after first start up or use secrets to inject password
	// you can set an existing bulwark account, no password needed, or if you want one created provide a password
	config.AdminAccount = getEnv("ADMIN_ACCOUNT", "")
	config.AdminAccountPassword = getEnv("ADMIN_ACCOUNT_PASSWORD", "")

	return config, nil
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
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
