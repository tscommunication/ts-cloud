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

func TestMigrateCreatesDistributionHierarchy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"pops", "agents"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s", table)
		}
	}
	for _, column := range []string{"pop_id", "agent_id"} {
		if !db.Migrator().HasColumn("customers", column) {
			t.Fatalf("expected customers.%s", column)
		}
	}
}

func TestCorrectPPPoESessionTableNameRenamesLegacyTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE network_router_pp_po_e_sessions (id integer primary key)").Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateCorrectPPPoESessionTableName(db); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasTable("network_router_pppoe_sessions") {
		t.Fatal("expected corrected PPPoE session table")
	}
	if db.Migrator().HasTable("network_router_pp_po_e_sessions") {
		t.Fatal("legacy PPPoE session table still exists")
	}
}
