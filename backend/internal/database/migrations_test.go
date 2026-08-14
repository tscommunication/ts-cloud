package database

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&schemaMigration{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != int64(len(migrations)) {
		t.Fatalf("expected %d applied migrations, got %d", len(migrations), count)
	}
}
