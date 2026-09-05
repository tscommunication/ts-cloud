package services

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/automation/linux"
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestProcessVSFTPDEventRecordsUnknownFailedLogin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.FTPLoginLog{}); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	err = processVSFTPDEvent(&linux.VSFTPDEvent{
		Type:     linux.EventLoginFailed,
		Username: "unknown-user",
		IP:       "198.51.100.5",
	})
	if err != nil {
		t.Fatal(err)
	}

	var log models.FTPLoginLog
	if err := db.First(&log).Error; err != nil {
		t.Fatal(err)
	}
	if log.FTPUserID != nil || log.Username != "unknown-user" || log.IPAddress != "198.51.100.5" || log.LoginStatus != "FAILED" {
		t.Fatalf("unexpected failed-login audit record: %+v", log)
	}
}
