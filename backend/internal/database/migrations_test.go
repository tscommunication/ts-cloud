package database

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/models"
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

func TestUnifiedCustomerIdentityMigrationBackfillsMissingAccount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:unified_customer_identity?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.User{}); err != nil {
		t.Fatal(err)
	}

	customer := models.Customer{CustomerCode: "CUS-000101", FullName: "Backfill Customer", Mobile: "01700000101", NID: "1234567101"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateUnifiedCustomerIdentity(db); err != nil {
		t.Fatal(err)
	}

	var identity models.User
	if err := db.Where("customer_id = ?", customer.ID).First(&identity).Error; err != nil {
		t.Fatal(err)
	}
	if identity.Username != customer.CustomerCode || identity.Role != "customer" || identity.Active {
		t.Fatalf("unexpected migrated identity: %+v", identity)
	}
}

func TestCustomerInternetAccountMigrationMovesPPPoEOwnership(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:customer_internet_accounts?mode=memory&cache=shared"),
		&gorm.Config{DisableForeignKeyConstraintWhenMigrating: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
	); err != nil {
		t.Fatal(err)
	}

	customer := models.Customer{CustomerCode: "CUS-000201", FullName: "Internet Customer", Mobile: "01700000201", NID: "1234567201"}
	pkg := models.Package{PackageCode: "PKG-201", Name: "Internet", Price: 1000}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	subscription := models.Subscription{
		SubscriptionCode: "SUB-000201", CustomerID: customer.ID, PackageID: pkg.ID,
		ActivationDate: time.Now(), NextBillingDate: time.Now(), ExpiryDate: time.Now(),
		BillingDay: 1, Status: "ACTIVE", RouterID: 9,
		PPPoEUsername: "customer-201", PPPoEPasswordEncrypted: "encrypted-secret",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateCustomerInternetAccounts(db); err != nil {
		t.Fatal(err)
	}

	if err := db.First(&subscription, subscription.ID).Error; err != nil {
		t.Fatal(err)
	}
	if subscription.InternetAccountID == nil {
		t.Fatal("expected subscription to link to customer internet account")
	}
	var account models.CustomerInternetAccount
	if err := db.First(&account, *subscription.InternetAccountID).Error; err != nil {
		t.Fatal(err)
	}
	if account.CustomerID != customer.ID || account.PPPoEUsername != "customer-201" || account.RouterID != 9 {
		t.Fatalf("unexpected internet account: %+v", account)
	}
}

func TestCustomerInternetLifecycleMigrationBackfillsLinkedSubscription(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:customer_internet_lifecycle?mode=memory&cache=shared"),
		&gorm.Config{DisableForeignKeyConstraintWhenMigrating: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
		&models.CustomerInternetAccount{},
	); err != nil {
		t.Fatal(err)
	}

	customer := models.Customer{CustomerCode: "CUS-000301", FullName: "Lifecycle Customer", Mobile: "01700000301"}
	pkg := models.Package{PackageCode: "PKG-301", Name: "Internet 30M", Price: 1300}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	account := models.CustomerInternetAccount{
		AccountCode: "NET-000301", CustomerID: customer.ID, RouterID: 3,
		PPPoEUsername: "customer-301", PPPoEPasswordEncrypted: "encrypted-secret",
		Status: "ACTIVE", BillingDay: 1,
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}

	activation := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	expiry := time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC)
	subscription := models.Subscription{
		SubscriptionCode: "SUB-000301", CustomerID: customer.ID, PackageID: pkg.ID,
		InternetAccountID: &account.ID, ActivationDate: activation,
		NextBillingDate: expiry, ExpiryDate: expiry, BillingDay: 18, Status: "ACTIVE",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	if err := migrateCustomerInternetLifecycle(db); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&account, account.ID).Error; err != nil {
		t.Fatal(err)
	}
	if account.PackageID != pkg.ID || account.BillingDay != 18 || account.Status != "ACTIVE" {
		t.Fatalf("unexpected internet lifecycle backfill: %+v", account)
	}
	if account.ActivationDate == nil || !account.ActivationDate.Equal(activation) {
		t.Fatalf("unexpected activation date: %v", account.ActivationDate)
	}
	if account.NextBillingDate == nil || !account.NextBillingDate.Equal(expiry) {
		t.Fatalf("unexpected next billing date: %v", account.NextBillingDate)
	}
	if account.ExpiryDate == nil || !account.ExpiryDate.Equal(expiry) {
		t.Fatalf("unexpected expiry date: %v", account.ExpiryDate)
	}
}

func TestTemporaryInternetAccessLedgerMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:temporary_access_ledger?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasTable(&models.TemporaryInternetAccess{}) {
		t.Fatal("expected temporary_internet_accesses table")
	}
	for _, column := range []string{
		"customer_id", "internet_account_id", "subscription_id", "status",
		"starts_at", "ends_at", "granted_duration_seconds", "promised_payment_at",
		"promised_amount", "request_source", "reason", "granted_by_user_id",
		"settlement_payment_id", "pre_settlement_status", "settled_at",
	} {
		if !db.Migrator().HasColumn(&models.TemporaryInternetAccess{}, column) {
			t.Fatalf("expected temporary_internet_accesses.%s", column)
		}
	}
}

func TestCustomerFTPEntitlementMigrationBackfillsOwner(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:customer_ftp_entitlements?mode=memory&cache=shared"),
		&gorm.Config{DisableForeignKeyConstraintWhenMigrating: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Customer{}, &models.Package{}, &models.Subscription{}, &models.FTPServer{},
	); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE ftp_users (
		id integer primary key, subscription_id integer not null,
		ftp_server_id integer not null, username text not null,
		password text not null, home_directory text not null
	)`).Error; err != nil {
		t.Fatal(err)
	}

	customer := models.Customer{
		CustomerCode: "CUS-000401", FullName: "FTP Customer",
		Mobile: "01700000401", NID: "1234567401",
	}
	pkg := models.Package{PackageCode: "PKG-401", Name: "FTP", Price: 100}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	subscription := models.Subscription{
		SubscriptionCode: "SUB-000401", CustomerID: customer.ID, PackageID: pkg.ID,
		ActivationDate: time.Now(), NextBillingDate: time.Now(), ExpiryDate: time.Now(),
		BillingDay: 1, Status: "ACTIVE",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}
	server := models.FTPServer{
		Name: "FTP-1", Host: "127.0.0.1", Username: "admin",
		Password: "encrypted", RootPath: "/srv/ftp",
	}
	if err := db.Create(&server).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(
		`INSERT INTO ftp_users (subscription_id, ftp_server_id, username, password, home_directory)
		 VALUES (?, ?, ?, ?, ?)`,
		subscription.ID, server.ID, "ftp-401", "encrypted", "/srv/ftp/ftp-401",
	).Error; err != nil {
		t.Fatal(err)
	}

	if err := migrateCustomerFTPEntitlements(db); err != nil {
		t.Fatal(err)
	}
	var entitlement models.FTPUser
	if err := db.First(&entitlement).Error; err != nil {
		t.Fatal(err)
	}
	if entitlement.CustomerID != customer.ID {
		t.Fatalf("expected customer %d, got %d", customer.ID, entitlement.CustomerID)
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

	for _, column := range []string{"country", "post_office", "postal_code", "road_or_area", "village_or_holding"} {
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

func TestMigrateCreatesBangladeshLocationMaster(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	for _, table := range []string{
		"divisions",
		"districts",
		"upazilas",
		"post_offices",
	} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s", table)
		}
	}

	if !db.Migrator().HasColumn("customers", "postal_code") {
		t.Fatal("expected customers.postal_code")
	}
}

func TestBangladeshLocationMasterHierarchy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	division := models.Division{
		Name: "Khulna",
	}
	if err := db.Create(&division).Error; err != nil {
		t.Fatal(err)
	}

	district := models.District{
		DivisionID: division.ID,
		Name:       "Magura",
	}
	if err := db.Create(&district).Error; err != nil {
		t.Fatal(err)
	}

	upazila := models.Upazila{
		DistrictID: district.ID,
		Name:       "Magura Sadar",
	}
	if err := db.Create(&upazila).Error; err != nil {
		t.Fatal(err)
	}

	postOffice := models.PostOffice{
		UpazilaID:  upazila.ID,
		Name:       "Magura",
		PostalCode: "7600",
	}
	if err := db.Create(&postOffice).Error; err != nil {
		t.Fatal(err)
	}

	var got models.PostOffice
	if err := db.First(&got, postOffice.ID).Error; err != nil {
		t.Fatal(err)
	}

	if got.UpazilaID != upazila.ID {
		t.Fatalf("expected upazila_id %d, got %d", upazila.ID, got.UpazilaID)
	}

	if got.Name != "Magura" {
		t.Fatalf("expected post office Magura, got %q", got.Name)
	}

	if got.PostalCode != "7600" {
		t.Fatalf("expected postal code 7600, got %q", got.PostalCode)
	}
}

func TestCustomerIdentityUniquenessMigrationCreatesUniqueIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	if !db.Migrator().HasIndex(&models.Customer{}, "idx_customers_mobile_unique") {
		t.Fatal("expected unique customer mobile index")
	}

	if !db.Migrator().HasIndex(&models.Customer{}, "idx_customers_nid_unique") {
		t.Fatal("expected unique customer NID index")
	}

	first := models.Customer{
		CustomerCode: "CUS-TEST-001",
		FullName:     "Identity One",
		Mobile:       "01711111111",
		NID:          "1234567890",
	}

	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first customer: %v", err)
	}

	duplicateMobile := models.Customer{
		CustomerCode: "CUS-TEST-002",
		FullName:     "Identity Two",
		Mobile:       "01711111111",
		NID:          "1234567891",
	}

	if err := db.Create(&duplicateMobile).Error; err == nil {
		t.Fatal("expected duplicate mobile to be rejected")
	}

	blankNIDOne := models.Customer{
		CustomerCode: "CUS-BLANK-NID-001",
		FullName:     "Blank NID One",
		Mobile:       "01911111111",
		NID:          "",
	}

	if err := db.Create(&blankNIDOne).Error; err != nil {
		t.Fatalf("create first blank NID customer: %v", err)
	}

	blankNIDTwo := models.Customer{
		CustomerCode: "CUS-BLANK-NID-002",
		FullName:     "Blank NID Two",
		Mobile:       "01611111111",
		NID:          "",
	}

	if err := db.Create(&blankNIDTwo).Error; err != nil {
		t.Fatalf("expected multiple blank NIDs to be allowed: %v", err)
	}

	duplicateNID := models.Customer{
		CustomerCode: "CUS-TEST-003",
		FullName:     "Identity Three",
		Mobile:       "01811111111",
		NID:          "1234567890",
	}

	if err := db.Create(&duplicateNID).Error; err == nil {
		t.Fatal("expected duplicate NID to be rejected")
	}
}

func TestCustomerIdentityUniquenessMigrationRejectsExistingDuplicates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Exec(`
		CREATE TABLE customers (
			id integer primary key,
			customer_code text,
			full_name text,
			mobile text NOT NULL,
			n_id text
		)
	`).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Exec(`
		INSERT INTO customers
			(id, customer_code, full_name, mobile, n_id)
		VALUES
			(1, 'CUS-000001', 'One', '01700000000', '1234567890'),
			(2, 'CUS-000002', 'Two', '01700000000', '1234567891')
	`).Error; err != nil {
		t.Fatal(err)
	}

	err = migrateCustomerIdentityUniqueness(db)
	if err == nil {
		t.Fatal("expected duplicate mobile migration failure")
	}
	if !strings.Contains(err.Error(), "duplicate mobile") {
		t.Fatalf("unexpected migration error: %v", err)
	}
}

func TestCustomerProvisionRequestMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:customer_provision_request_migration?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	if !db.Migrator().HasTable(&models.CustomerProvisionRequest{}) {
		t.Fatal("expected customer_provision_requests table")
	}

	requiredColumns := []string{
		"request_code",
		"source",
		"status",
		"agent_id",
		"pop_id",
		"full_name",
		"mobile",
		"n_id",
		"package_id",
		"router_id",
		"pp_po_e_username",
		"billing_day",
		"activation_date",
		"requested_by_user_id",
		"requested_at",
		"reviewed_by_user_id",
		"reviewed_at",
		"rejection_reason",
		"customer_id",
		"subscription_id",
	}

	for _, column := range requiredColumns {
		if !db.Migrator().HasColumn(&models.CustomerProvisionRequest{}, column) {
			t.Fatalf("expected column %q in customer_provision_requests", column)
		}
	}

	if !db.Migrator().HasIndex(&models.CustomerProvisionRequest{}, "RequestCode") {
		t.Fatal("expected unique index for request_code")
	}
}

func TestCustomerExtendedDomainMigration(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:customer_extended_domain?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	customerColumns := []string{
		"joining_date",
		"occupation",
		"company_name",
		"designation",
		"date_of_birth",
		"nid_birth_date",
		"nid_issue_date",
		"nid_address",
		"present_address",
		"permanent_address",
		"tin",
		"customer_note",
	}

	for _, column := range customerColumns {
		if !db.Migrator().HasColumn(&models.Customer{}, column) {
			t.Fatalf("expected customers.%s", column)
		}
	}

	if !db.Migrator().HasTable(&models.CustomerTechnicalProfile{}) {
		t.Fatal("expected customer_technical_profiles table")
	}

	if !db.Migrator().HasTable(&models.CustomerReference{}) {
		t.Fatal("expected customer_references table")
	}

	technicalColumns := []string{
		"customer_id",
		"onu_mac",
		"olt_pon",
		"olt_slot",
		"olt_port",
		"onu_type",
		"onu_model",
		"onu_ip",
		"onu_password_encrypted",
		"onu_serial",
		"onu_sn",
		"router_brand",
		"router_model",
		"router_ip",
		"router_password_encrypted",
		"cable_type",
		"cable_length",
		"media_converter_mac",
		"media_converter_ip",
		"media_converter_password_encrypted",
		"switch_model",
		"switch_port",
		"switch_ip",
		"switch_password_encrypted",
		"additional_note",
	}

	for _, column := range technicalColumns {
		if !db.Migrator().HasColumn(
			&models.CustomerTechnicalProfile{},
			column,
		) {
			t.Fatalf(
				"expected customer_technical_profiles.%s",
				column,
			)
		}
	}

	referenceColumns := []string{
		"customer_id",
		"name",
		"mobile",
		"address",
		"relation",
		"note",
	}

	for _, column := range referenceColumns {
		if !db.Migrator().HasColumn(
			&models.CustomerReference{},
			column,
		) {
			t.Fatalf(
				"expected customer_references.%s",
				column,
			)
		}
	}

	customer := models.Customer{
		CustomerCode: "CUS-EXT-001",
		FullName:     "Extended Domain Test",
		Mobile:       "01712345678",
		NID:          "1234567890123",
	}

	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}

	firstProfile := models.CustomerTechnicalProfile{
		CustomerID: customer.ID,
		ONUMAC:     "AA:BB:CC:DD:EE:FF",
	}

	if err := db.Create(&firstProfile).Error; err != nil {
		t.Fatalf("create technical profile: %v", err)
	}

	duplicateProfile := models.CustomerTechnicalProfile{
		CustomerID: customer.ID,
	}

	if err := db.Create(&duplicateProfile).Error; err == nil {
		t.Fatal(
			"expected one technical profile per customer",
		)
	}

	firstReference := models.CustomerReference{
		CustomerID: customer.ID,
		Name:       "Reference One",
	}

	secondReference := models.CustomerReference{
		CustomerID: customer.ID,
		Name:       "Reference Two",
	}

	if err := db.Create(&firstReference).Error; err != nil {
		t.Fatalf("create first reference: %v", err)
	}

	if err := db.Create(&secondReference).Error; err != nil {
		t.Fatalf(
			"expected multiple references per customer: %v",
			err,
		)
	}
}

func TestPPPoEPasswordEncryptionColumnsMigration(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:pppoe-encryption?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	checks := []struct {
		model  interface{}
		column string
	}{
		{
			model:  &models.Subscription{},
			column: "pp_po_e_password",
		},
		{
			model:  &models.Subscription{},
			column: "pp_po_e_password_encrypted",
		},
		{
			model:  &models.CustomerProvisionRequest{},
			column: "pp_po_e_password",
		},
		{
			model:  &models.CustomerProvisionRequest{},
			column: "pp_po_e_password_encrypted",
		},
	}

	for _, check := range checks {
		if !db.Migrator().HasColumn(
			check.model,
			check.column,
		) {
			t.Fatalf(
				"expected column %q",
				check.column,
			)
		}
	}
}

func TestCustomerPortalUserIdentityMigration(
	t *testing.T,
) {
	db, err := gorm.Open(
		sqlite.Open("file:customer-portal-user-identity?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	if !db.Migrator().HasColumn(
		&models.User{},
		"customer_id",
	) {
		t.Fatal("expected users.customer_id column")
	}

	if !db.Migrator().HasIndex(
		&models.User{},
		"CustomerID",
	) {
		t.Fatal("expected unique customer_id index")
	}

	customer := models.Customer{
		CustomerCode: "CUS-PORTAL-001",
		FullName:     "Portal Customer",
		Mobile:       "01790000001",
	}

	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}

	customerUser := models.User{
		Name:       "Portal Customer",
		Username:   "portal-customer-1",
		Email:      "portal1@example.com",
		Password:   "hashed-test-password",
		Role:       "customer",
		Active:     true,
		CustomerID: &customer.ID,
	}

	if err := db.Create(&customerUser).Error; err != nil {
		t.Fatalf("create customer user: %v", err)
	}

	duplicateCustomerUser := models.User{
		Name:       "Duplicate Portal Customer",
		Username:   "portal-customer-2",
		Email:      "portal2@example.com",
		Password:   "hashed-test-password",
		Role:       "customer",
		Active:     true,
		CustomerID: &customer.ID,
	}

	if err := db.Create(&duplicateCustomerUser).Error; err == nil {
		t.Fatal("expected one login account per customer")
	}

	firstStaff := models.User{
		Name:     "Staff One",
		Username: "staff-one",
		Email:    "staff1@example.com",
		Password: "hashed-test-password",
		Role:     "admin",
		Active:   true,
	}

	secondStaff := models.User{
		Name:     "Staff Two",
		Username: "staff-two",
		Email:    "staff2@example.com",
		Password: "hashed-test-password",
		Role:     "user",
		Active:   true,
	}

	if err := db.Create(&firstStaff).Error; err != nil {
		t.Fatalf("create first staff user: %v", err)
	}

	if err := db.Create(&secondStaff).Error; err != nil {
		t.Fatalf(
			"expected multiple users with null customer_id: %v",
			err,
		)
	}
}

func TestCustomerGeoCoordinatesMigration(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:customer_geo_coordinates?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	for _, model := range []struct {
		name  string
		value interface{}
	}{
		{
			name:  "customers",
			value: &models.Customer{},
		},
		{
			name:  "customer_provision_requests",
			value: &models.CustomerProvisionRequest{},
		},
	} {
		for _, column := range []string{
			"latitude",
			"longitude",
		} {
			if !db.Migrator().HasColumn(
				model.value,
				column,
			) {
				t.Fatalf(
					"expected %s.%s",
					model.name,
					column,
				)
			}
		}
	}
}

func TestNetworkDeviceSampleUniquenessMigration(
	t *testing.T,
) {
	db, err := gorm.Open(
		sqlite.Open(
			"file:network_device_sample_uniqueness?mode=memory&cache=shared",
		),
		&gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.NetworkDevicePort{},
		&models.NetworkDevicePortSample{},
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
	); err != nil {
		t.Fatal(err)
	}

	if err := migrateNetworkDeviceSampleUniqueness(
		db,
	); err != nil {
		t.Fatal(err)
	}

	if !db.Migrator().HasIndex(
		&models.NetworkDevicePortSample{},
		"idx_network_device_port_sample_unique",
	) {
		t.Fatal(
			"missing port sample unique index",
		)
	}

	if !db.Migrator().HasIndex(
		&models.NetworkDeviceONUSample{},
		"idx_network_device_onu_sample_unique",
	) {
		t.Fatal(
			"missing ONU sample unique index",
		)
	}

	sampledAt := time.Date(
		2026,
		time.August,
		23,
		6,
		15,
		0,
		0,
		time.UTC,
	)

	portSample := models.NetworkDevicePortSample{
		NetworkDevicePortID: 1,
		SampledAt:           sampledAt,
	}

	if err := db.Create(
		&portSample,
	).Error; err != nil {
		t.Fatal(err)
	}

	duplicatePortSample :=
		models.NetworkDevicePortSample{
			NetworkDevicePortID: 1,
			SampledAt:           sampledAt,
		}

	if err := db.Create(
		&duplicatePortSample,
	).Error; err == nil {
		t.Fatal(
			"expected duplicate port sample rejection",
		)
	}

	onuSample := models.NetworkDeviceONUSample{
		NetworkDeviceONUID: 1,
		SampledAt:          sampledAt,
	}

	if err := db.Create(
		&onuSample,
	).Error; err != nil {
		t.Fatal(err)
	}

	duplicateONUSample :=
		models.NetworkDeviceONUSample{
			NetworkDeviceONUID: 1,
			SampledAt:          sampledAt,
		}

	if err := db.Create(
		&duplicateONUSample,
	).Error; err == nil {
		t.Fatal(
			"expected duplicate ONU sample rejection",
		)
	}
}
