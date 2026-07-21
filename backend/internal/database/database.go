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

        if err := DB.AutoMigrate(
                &models.User{},
                &models.Customer{},
        ); err != nil {
                return err
        }

        log.Println("Database migrated")
        log.Println("Database connected")

        return nil
}
