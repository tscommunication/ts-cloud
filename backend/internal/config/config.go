package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName                  string
	AppEnv                   string
	AppPort                  string
	JWTSecret                string
	CredentialKey            string
	RouterCredentialKey      string
	RouterMonitorInterval    time.Duration
	RouterCPUAlertPercent    int
	RouterMemoryAlertPercent int
	DBType                   string
	DBPath                   string
	DBDSN                    string
	StoragePath              string
	LogLevel                 string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		AppName:   os.Getenv("APP_NAME"),
		AppEnv:    os.Getenv("APP_ENV"),
		AppPort:   os.Getenv("APP_PORT"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		CredentialKey: firstNonEmpty(
			os.Getenv("CREDENTIAL_KEY"),
			os.Getenv("ROUTER_CREDENTIAL_KEY"),
		),
		RouterCredentialKey: os.Getenv("ROUTER_CREDENTIAL_KEY"),
		DBType:              os.Getenv("DB_TYPE"),
		DBPath:              os.Getenv("DB_PATH"),
		DBDSN:               firstNonEmpty(os.Getenv("DATABASE_URL"), os.Getenv("DB_DSN")),
		StoragePath:         os.Getenv("STORAGE_PATH"),
		LogLevel:            os.Getenv("LOG_LEVEL"),
	}
	monitorInterval := strings.TrimSpace(os.Getenv("ROUTER_MONITOR_INTERVAL"))
	if monitorInterval == "" {
		cfg.RouterMonitorInterval = time.Minute
	} else if monitorInterval == "0" {
		cfg.RouterMonitorInterval = 0
	} else {
		parsed, err := time.ParseDuration(monitorInterval)
		if err != nil || parsed < 10*time.Second {
			panic("ROUTER_MONITOR_INTERVAL must be 0 or a duration of at least 10s")
		}
		cfg.RouterMonitorInterval = parsed
	}
	cfg.RouterCPUAlertPercent = percentEnv("ROUTER_CPU_ALERT_PERCENT", 85)
	cfg.RouterMemoryAlertPercent = percentEnv("ROUTER_MEMORY_ALERT_PERCENT", 90)

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

func percentEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 100 {
		panic(name + " must be an integer between 1 and 100")
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
