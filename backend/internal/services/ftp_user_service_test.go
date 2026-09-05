package services

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupFTPReconcileTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open("file:ftp-reconcile-"+t.Name()+"?mode=memory&cache=shared"),
		&gorm.Config{DisableForeignKeyConstraintWhenMigrating: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
		&models.ServiceEntitlement{},
		&models.FTPServer{},
		&models.FTPUser{},
	); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db

	oldExists := ftpLinuxUserExists
	oldLock := ftpLinuxLockUser
	oldUnlock := ftpLinuxUnlockUser
	oldProvision := ftpProvisionUser
	oldProvisionWithPassword := ftpProvisionUserWithPassword

	t.Cleanup(func() {
		database.DB = previousDB
		ftpLinuxUserExists = oldExists
		ftpLinuxLockUser = oldLock
		ftpLinuxUnlockUser = oldUnlock
		ftpProvisionUser = oldProvision
		ftpProvisionUserWithPassword = oldProvisionWithPassword
	})

	return db
}

func createFTPReconcileFixture(t *testing.T, db *gorm.DB, entitlementStatus, ftpStatus string) (*models.ServiceEntitlement, *models.FTPUser) {
	t.Helper()

	customer := models.Customer{
		CustomerCode: "CUS-FTP-REC",
		FullName:     "FTP Reconcile Customer",
		Mobile:       "01000000001",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-FTP-REC",
		Name:        "FTP Package",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	subscription := models.Subscription{
		SubscriptionCode: "SUB-FTP-REC",
		CustomerID:       customer.ID,
		PackageID:        pkg.ID,
		Status:           "ACTIVE",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	server := models.FTPServer{
		Name:     "FTP Test",
		Host:     "127.0.0.1",
		Port:     21,
		Username: "admin",
		Password: "secret",
		RootPath: "/data/ftp",
		Status:   "ACTIVE",
	}
	if err := db.Create(&server).Error; err != nil {
		t.Fatal(err)
	}

	entitlement := models.ServiceEntitlement{
		CustomerID:     customer.ID,
		SubscriptionID: &subscription.ID,
		ServiceType:    "FTP",
		ServiceName:    "FTP Service",
		Status:         entitlementStatus,
	}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}

	user := models.FTPUser{
		CustomerID:           customer.ID,
		SubscriptionID:       subscription.ID,
		ServiceEntitlementID: &entitlement.ID,
		FTPServerID:          server.ID,
		Username:             "ftp-reconcile-user",
		Password:             "secret",
		HomeDirectory:        "/data/ftp/ftp-reconcile-user",
		StorageQuotaGB:       10,
		Status:               ftpStatus,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	return &entitlement, &user
}

func TestReconcileFTPServiceEntitlementActiveUnlocksExistingLinuxUser(t *testing.T) {
	db := setupFTPReconcileTestDB(t)
	entitlement, user := createFTPReconcileFixture(t, db, "ACTIVE", "SUSPENDED")

	unlockCalls := 0
	provisionCalls := 0

	ftpLinuxUserExists = func(string) bool { return true }
	ftpLinuxUnlockUser = func(username string) error {
		unlockCalls++
		if username != user.Username {
			t.Fatalf("unlock username = %q, want %q", username, user.Username)
		}
		return nil
	}
	ftpProvisionUser = func(*models.FTPUser) error {
		provisionCalls++
		return nil
	}

	if err := ReconcileFTPServiceEntitlement(entitlement); err != nil {
		t.Fatal(err)
	}

	if unlockCalls != 1 {
		t.Fatalf("unlock calls = %d, want 1", unlockCalls)
	}
	if provisionCalls != 0 {
		t.Fatalf("provision calls = %d, want 0", provisionCalls)
	}

	var saved models.FTPUser
	if err := db.First(&saved, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != "ACTIVE" {
		t.Fatalf("FTP status = %q, want ACTIVE", saved.Status)
	}
}

func TestReconcileFTPServiceEntitlementActiveReprovisionsMissingLinuxUser(t *testing.T) {
	db := setupFTPReconcileTestDB(t)
	entitlement, user := createFTPReconcileFixture(t, db, "ACTIVE", "SUSPENDED")

	provisionCalls := 0

	ftpLinuxUserExists = func(string) bool { return false }
	ftpProvisionUser = func(got *models.FTPUser) error {
		provisionCalls++
		if got.ID != user.ID {
			t.Fatalf("provision FTP user ID = %d, want %d", got.ID, user.ID)
		}
		return nil
	}

	if err := ReconcileFTPServiceEntitlement(entitlement); err != nil {
		t.Fatal(err)
	}

	if provisionCalls != 1 {
		t.Fatalf("provision calls = %d, want 1", provisionCalls)
	}

	var saved models.FTPUser
	if err := db.First(&saved, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != "ACTIVE" {
		t.Fatalf("FTP status = %q, want ACTIVE", saved.Status)
	}
}

func TestReconcileFTPServiceEntitlementSuspendedLocksExistingLinuxUser(t *testing.T) {
	db := setupFTPReconcileTestDB(t)
	entitlement, user := createFTPReconcileFixture(t, db, "SUSPENDED", "ACTIVE")

	lockCalls := 0

	ftpLinuxUserExists = func(string) bool { return true }
	ftpLinuxLockUser = func(username string) error {
		lockCalls++
		if username != user.Username {
			t.Fatalf("lock username = %q, want %q", username, user.Username)
		}
		return nil
	}

	if err := ReconcileFTPServiceEntitlement(entitlement); err != nil {
		t.Fatal(err)
	}

	if lockCalls != 1 {
		t.Fatalf("lock calls = %d, want 1", lockCalls)
	}

	var saved models.FTPUser
	if err := db.First(&saved, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != "SUSPENDED" {
		t.Fatalf("FTP status = %q, want SUSPENDED", saved.Status)
	}
}

func TestReconcileFTPServiceEntitlementDisabledWithoutLinuxUserUpdatesDatabaseOnly(t *testing.T) {
	db := setupFTPReconcileTestDB(t)
	entitlement, user := createFTPReconcileFixture(t, db, "DISABLED", "ACTIVE")

	lockCalls := 0

	ftpLinuxUserExists = func(string) bool { return false }
	ftpLinuxLockUser = func(string) error {
		lockCalls++
		return nil
	}

	if err := ReconcileFTPServiceEntitlement(entitlement); err != nil {
		t.Fatal(err)
	}

	if lockCalls != 0 {
		t.Fatalf("lock calls = %d, want 0", lockCalls)
	}

	var saved models.FTPUser
	if err := db.First(&saved, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != "DISABLED" {
		t.Fatalf("FTP status = %q, want DISABLED", saved.Status)
	}
}

func TestReconcileFTPServiceEntitlementNonFTPIsNoOp(t *testing.T) {
	setupFTPReconcileTestDB(t)

	entitlement := &models.ServiceEntitlement{
		ServiceType: "IPTV",
		Status:      "ACTIVE",
	}
	entitlement.ID = 999

	if err := ReconcileFTPServiceEntitlement(entitlement); err != nil {
		t.Fatal(err)
	}
}

func TestReconcileFTPServiceEntitlementRequiresLinkedFTPUser(t *testing.T) {
	db := setupFTPReconcileTestDB(t)

	customer := models.Customer{
		CustomerCode: "CUS-FTP-NOLINK",
		FullName:     "No Link",
		Mobile:       "01000000002",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	entitlement := models.ServiceEntitlement{
		CustomerID:  customer.ID,
		ServiceType: "FTP",
		ServiceName: "FTP Missing Projection",
		Status:      "ACTIVE",
	}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}

	err := ReconcileFTPServiceEntitlement(&entitlement)
	if err == nil {
		t.Fatal("expected missing linked FTP user error")
	}
	if !strings.Contains(err.Error(), "load FTP user for service entitlement") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildManagedFTPUserUsesPPPoEIdentity(t *testing.T) {
	subscription := &models.Subscription{
		CustomerID: 44,
	}
	subscription.ID = 55

	account := &models.CustomerInternetAccount{
		PPPoEUsername: "Par_002_morad",
	}
	account.ID = 66

	entitlement := &models.ServiceEntitlement{
		QuotaGB: 15,
		Status:  "ACTIVE",
	}
	entitlement.ID = 77

	server := &models.FTPServer{
		RootPath: "/data/ftp",
	}
	server.ID = 88

	user, err := BuildManagedFTPUser(
		subscription,
		account,
		entitlement,
		server,
	)
	if err != nil {
		t.Fatal(err)
	}

	if user.CustomerID != 44 {
		t.Fatalf("customer id = %d, want 44", user.CustomerID)
	}
	if user.SubscriptionID != 55 {
		t.Fatalf("subscription id = %d, want 55", user.SubscriptionID)
	}
	if user.ServiceEntitlementID == nil || *user.ServiceEntitlementID != 77 {
		t.Fatalf("unexpected entitlement link: %v", user.ServiceEntitlementID)
	}
	if user.FTPServerID != 88 {
		t.Fatalf("FTP server id = %d, want 88", user.FTPServerID)
	}
	if user.Username != "Par_002_morad" {
		t.Fatalf("username = %q", user.Username)
	}
	if user.HomeDirectory != "/data/ftp/Par_002_morad" {
		t.Fatalf("home directory = %q", user.HomeDirectory)
	}
	if user.StorageQuotaGB != 15 {
		t.Fatalf("quota = %d, want 15", user.StorageQuotaGB)
	}
	if user.Password != "" {
		t.Fatal("managed FTP projection must not store plaintext password")
	}
}

func TestBuildManagedFTPUserRejectsUnsafeUsernamePath(t *testing.T) {
	subscription := &models.Subscription{CustomerID: 1}
	subscription.ID = 1

	account := &models.CustomerInternetAccount{
		PPPoEUsername: "../escape",
	}
	account.ID = 1

	entitlement := &models.ServiceEntitlement{
		Status: "ACTIVE",
	}
	entitlement.ID = 1

	server := &models.FTPServer{
		RootPath: "/data/ftp",
	}
	server.ID = 1

	if _, err := BuildManagedFTPUser(
		subscription,
		account,
		entitlement,
		server,
	); err == nil {
		t.Fatal("expected unsafe username path to be rejected")
	}
}

func TestProvisionManagedFTPUserPassesPasswordOnlyInMemory(t *testing.T) {
	setupFTPReconcileTestDB(t)

	called := 0
	ftpProvisionUserWithPassword = func(
		user *models.FTPUser,
		password string,
	) error {
		called++

		if user.Password != "" {
			t.Fatal("managed FTP projection contains plaintext password")
		}
		if password != "pppoe-secret-123" {
			t.Fatalf("password = %q", password)
		}

		return nil
	}

	user := &models.FTPUser{
		Username:      "managed-user",
		HomeDirectory: "/data/ftp/managed-user",
	}

	if err := ProvisionManagedFTPUser(
		user,
		"pppoe-secret-123",
	); err != nil {
		t.Fatal(err)
	}

	if called != 1 {
		t.Fatalf("provision calls = %d, want 1", called)
	}
}

func TestGetSingleActiveFTPServer(t *testing.T) {
	db := setupFTPReconcileTestDB(t)

	server := models.FTPServer{
		Name:     "Primary FTP",
		Driver:   "linux",
		Host:     "127.0.0.1",
		Port:     21,
		Username: "admin",
		Password: "secret",
		RootPath: "/data/ftp",
		Status:   "ACTIVE",
	}
	if err := db.Create(&server).Error; err != nil {
		t.Fatal(err)
	}

	found, err := GetSingleActiveFTPServer()
	if err != nil {
		t.Fatal(err)
	}

	if found.ID != server.ID {
		t.Fatalf("server id = %d, want %d", found.ID, server.ID)
	}
}

func TestGetSingleActiveFTPServerRejectsMissingServer(t *testing.T) {
	setupFTPReconcileTestDB(t)

	if _, err := GetSingleActiveFTPServer(); err == nil {
		t.Fatal("expected missing active FTP server error")
	}
}

func TestGetSingleActiveFTPServerRejectsAmbiguousServers(t *testing.T) {
	db := setupFTPReconcileTestDB(t)

	for _, name := range []string{"FTP A", "FTP B"} {
		server := models.FTPServer{
			Name:     name,
			Driver:   "linux",
			Host:     "127.0.0.1",
			Port:     21,
			Username: "admin",
			Password: "secret",
			RootPath: "/data/ftp",
			Status:   "ACTIVE",
		}
		if err := db.Create(&server).Error; err != nil {
			t.Fatal(err)
		}
	}

	if _, err := GetSingleActiveFTPServer(); err == nil {
		t.Fatal("expected multiple active FTP servers to be rejected")
	}
}
