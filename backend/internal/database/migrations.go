package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type schemaMigration struct {
	Version   uint      `gorm:"primaryKey"`
	Name      string    `gorm:"size:200;not null"`
	AppliedAt time.Time `gorm:"not null"`
}

type migration struct {
	version uint
	name    string
	up      func(*gorm.DB) error
}

var migrations = []migration{
	{version: 1, name: "initial_application_schema", up: migrateInitialSchema},
}

func runMigrations(db *gorm.DB) error {
	return Migrate(db)
}

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}
	for _, item := range migrations {
		var count int64
		if err := db.Model(&schemaMigration{}).Where("version = ?", item.version).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := item.up(tx); err != nil {
				return err
			}
			return tx.Create(&schemaMigration{Version: item.version, Name: item.name, AppliedAt: time.Now().UTC()}).Error
		}); err != nil {
			return fmt.Errorf("migration %03d %s: %w", item.version, item.name, err)
		}
	}
	return nil
}

func migrateInitialSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{}, &models.Customer{}, &models.Package{}, &models.Subscription{},
		&models.Invoice{}, &models.Payment{}, &models.BillingRun{}, &models.BillingRunItem{},
		&models.FTPServer{}, &models.FTPUser{}, &models.FTPLoginLog{},
		&models.FTPTransferLog{}, &models.SystemLogOffset{},
	)
}
