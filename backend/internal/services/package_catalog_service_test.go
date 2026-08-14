package services

import (
	"testing"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSyncApprovedPackageCatalogIsCompleteAndIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:approved_package_catalog?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Package{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })

	created, updated, err := SyncApprovedPackageCatalog()
	if err != nil || created != 40 || updated != 0 {
		t.Fatalf("unexpected first sync: created=%d updated=%d err=%v", created, updated, err)
	}
	created, updated, err = SyncApprovedPackageCatalog()
	if err != nil || created != 0 || updated != 40 {
		t.Fatalf("unexpected second sync: created=%d updated=%d err=%v", created, updated, err)
	}

	var count int64
	if err := db.Model(&models.Package{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 40 {
		t.Fatalf("expected 40 workbook packages, got %d", count)
	}
	var pkg models.Package
	if err := db.Where("name = ?", "N_Go_P25").First(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	if pkg.PackageCode != "CAT-000089" || pkg.Price != 550 || pkg.MikroTikProfile != "Go_P25" || pkg.Commission != 220 || pkg.DownloadSpeed != 25 {
		t.Fatalf("unexpected synchronized package: %+v", pkg)
	}
	var gamer models.Package
	if err := db.Where("name = ?", "Gamer_30MB").First(&gamer).Error; err != nil {
		t.Fatal(err)
	}
	if gamer.DownloadSpeed != 30 || gamer.UploadSpeed != 30 {
		t.Fatalf("expected Gamer_30MB speed to be inferred as 30 Mbps: %+v", gamer)
	}
}
