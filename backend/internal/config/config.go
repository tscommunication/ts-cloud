package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName     string
	AppEnv      string
	AppPort     string
	JWTSecret   string
	DBType      string
	DBPath      string
	StoragePath string
	LogLevel    string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		AppName:     os.Getenv("APP_NAME"),
		AppEnv:      os.Getenv("APP_ENV"),
		AppPort:     os.Getenv("APP_PORT"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		DBType:      os.Getenv("DB_TYPE"),
		DBPath:      os.Getenv("DB_PATH"),
		StoragePath: os.Getenv("STORAGE_PATH"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
	}

	log.Println("Configuration loaded")

	return cfg
}
