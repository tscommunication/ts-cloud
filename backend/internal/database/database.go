package database

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/models"
)

var DB *gorm.DB

func Connect(cfg *config.Config) error {

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		return err
	}

	DB = db

	// Auto Migration
	if err := DB.AutoMigrate(

		// Authentication
		&models.User{},

		// Core
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},

		// Billing
		&models.Invoice{},
		&models.Payment{},

		// Sprint 14 - FTP Service
		&models.FTPServer{},
		&models.FTPUser{},

	); err != nil {
		return err
	}

	log.Println("Database migrated")
	log.Println("Database connected")

	return nil
}
