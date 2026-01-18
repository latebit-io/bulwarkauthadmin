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
	EmailTemplatesDir    string
}

func NewAppConfig() (*AppConfig, error) {
	config := &AppConfig{}
	config.CORSEnabled = getEnvAsBool("CORS_ENABLED", false)
	config.BulwarkAuthUrl = getEnv("BULWARK_AUTH_URL", "http://localhost:8080")
	config.Port = getEnvAsInt("PORT", 8081)
	config.DbConnection = getEnv("DB_CONNECTION", "mongodb://localhost:27017/?connect=direct")
	config.DbNameSeed = getEnv("DB_NAME_SEED", "")
	config.AllowedOrigins = getEnvAsStringSlice("ALLOWED_WEB_ORIGINS", []string{"http://localhost:5173"})
	config.EmailTemplatesDir = getEnv("EMAIL_TEMPLATES_DIR", "")
	// SECURITY WARNING:
	// These environment variables are intended only for initial provisioning.
	// After the first successful startup (once the admin account is created or linked),
	// immediately remove ADMIN_ACCOUNT and ADMIN_ACCOUNT_PASSWORD from the environment
	// and from any configuration files or deployment manifests.
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
