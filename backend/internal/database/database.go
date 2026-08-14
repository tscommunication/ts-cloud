package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/config"
)

var DB *gorm.DB

func Connect(cfg *config.Config) error {

	var dialector gorm.Dialector
	switch cfg.DBType {
	case "sqlite":
		dialector = sqlite.Open(cfg.DBPath)
	case "postgres":
		dialector = postgres.Open(cfg.DBDSN)
	default:
		return fmt.Errorf("unsupported database type %q", cfg.DBType)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return err
	}

	DB = db

	if err := runMigrations(DB); err != nil {
		return err
	}

	log.Printf("Database connected and migrations applied (%s)", cfg.DBType)

	return nil
}
