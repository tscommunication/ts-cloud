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
	for _, table := range []string{"pops", "agents", "agent_pops"} {
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

func TestMigrateCustomerStructuredAddressBackfillsLegacyData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	// Simulate the pre-migration customer address schema.
	if err := db.Exec(`
		CREATE TABLE customers (
			id integer primary key,
			customer_code text,
			full_name text,
			mobile text,
			division text,
			district text,
			upazila text,
			"union" text,
			village text,
			address text
		)
	`).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Exec(`
		INSERT INTO customers
			(id, customer_code, full_name, mobile, village, address)
		VALUES
			(1, 'CUS-000001', 'Test Customer', '01700000000', '', ''),
			(2, 'CUS-000002', 'Saiful', '01321000000', 'kalinagor', 'kalinagor')
	`).Error; err != nil {
		t.Fatal(err)
	}

	if err := migrateCustomerStructuredAddress(db); err != nil {
		t.Fatal(err)
	}

	for _, column := range []string{"country", "post_office", "road_or_area", "village_or_holding"} {
		if !db.Migrator().HasColumn("customers", column) {
			t.Fatalf("expected customers.%s", column)
		}
	}

	type result struct {
		ID               uint
		Country          string
		RoadOrArea       string
		VillageOrHolding string
	}

	var rows []result
	if err := db.Table("customers").
		Select("id, country, road_or_area, village_or_holding").
		Order("id").
		Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 customers, got %d", len(rows))
	}

	if rows[0].Country != "Bangladesh" {
		t.Fatalf("expected customer 1 country Bangladesh, got %q", rows[0].Country)
	}
	if rows[0].RoadOrArea != "" || rows[0].VillageOrHolding != "" {
		t.Fatalf("expected empty legacy backfill for customer 1, got road=%q village=%q",
			rows[0].RoadOrArea, rows[0].VillageOrHolding)
	}

	if rows[1].Country != "Bangladesh" {
		t.Fatalf("expected customer 2 country Bangladesh, got %q", rows[1].Country)
	}
	if rows[1].RoadOrArea != "kalinagor" {
		t.Fatalf("expected road_or_area kalinagor, got %q", rows[1].RoadOrArea)
	}
	if rows[1].VillageOrHolding != "kalinagor" {
		t.Fatalf("expected village_or_holding kalinagor, got %q", rows[1].VillageOrHolding)
	}
}

func TestMigrateCustomerStructuredAddressPreservesExistingStructuredData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a database that already has the structured columns populated
	// while still retaining different legacy address values.
	if err := db.Exec(`
		CREATE TABLE customers (
			id integer primary key,
			customer_code text,
			full_name text,
			mobile text,
			division text,
			district text,
			upazila text,
			"union" text,
			village text,
			address text,
			country text,
			post_office text,
			road_or_area text,
			village_or_holding text
		)
	`).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Exec(`
		INSERT INTO customers (
			id,
			customer_code,
			full_name,
			mobile,
			village,
			address,
			country,
			road_or_area,
			village_or_holding
		)
		VALUES (
			1,
			'CUS-000001',
			'Preserve Test',
			'01700000000',
			'Legacy Village',
			'Legacy Road',
			'Bangladesh',
			'Structured Road',
			'Structured Village'
		)
	`).Error; err != nil {
		t.Fatal(err)
	}

	if err := migrateCustomerStructuredAddress(db); err != nil {
		t.Fatal(err)
	}

	type result struct {
		Country          string
		RoadOrArea       string
		VillageOrHolding string
	}

	var row result
	if err := db.Table("customers").
		Select("country, road_or_area, village_or_holding").
		Where("id = ?", 1).
		Scan(&row).Error; err != nil {
		t.Fatal(err)
	}

	if row.Country != "Bangladesh" {
		t.Fatalf("expected country Bangladesh, got %q", row.Country)
	}
	if row.RoadOrArea != "Structured Road" {
		t.Fatalf("expected existing road_or_area to be preserved, got %q", row.RoadOrArea)
	}
	if row.VillageOrHolding != "Structured Village" {
		t.Fatalf("expected existing village_or_holding to be preserved, got %q", row.VillageOrHolding)
	}
}
