package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName             string
	AppEnv              string
	AppPort             string
	JWTSecret           string
	RouterCredentialKey string
	DBType              string
	DBPath              string
	DBDSN               string
	StoragePath         string
	LogLevel            string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		AppName:             os.Getenv("APP_NAME"),
		AppEnv:              os.Getenv("APP_ENV"),
		AppPort:             os.Getenv("APP_PORT"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		RouterCredentialKey: os.Getenv("ROUTER_CREDENTIAL_KEY"),
		DBType:              os.Getenv("DB_TYPE"),
		DBPath:              os.Getenv("DB_PATH"),
		DBDSN:               firstNonEmpty(os.Getenv("DATABASE_URL"), os.Getenv("DB_DSN")),
		StoragePath:         os.Getenv("STORAGE_PATH"),
		LogLevel:            os.Getenv("LOG_LEVEL"),
	}

	if cfg.DBType == "" {
		cfg.DBType = "sqlite"
	}
	cfg.DBType = strings.ToLower(strings.TrimSpace(cfg.DBType))
	if cfg.DBType != "sqlite" && cfg.DBType != "postgres" {
		panic(fmt.Sprintf("unsupported DB_TYPE %q", cfg.DBType))
	}
	if cfg.DBType == "sqlite" && strings.TrimSpace(cfg.DBPath) == "" {
		panic("DB_PATH is required for sqlite")
	}
	if cfg.DBType == "postgres" && strings.TrimSpace(cfg.DBDSN) == "" {
		panic("DATABASE_URL or DB_DSN is required for postgres")
	}

	log.Printf("Configuration loaded (database=%s)", cfg.DBType)

	return cfg
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
