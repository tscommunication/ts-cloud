package services

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
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
		&models.PackageServicePolicy{},
		&models.CustomerInternetAccount{},
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
	oldSetPassword := ftpLinuxSetPassword
	oldRenameUser := ftpLinuxRenameUser
	oldUpdateUser := ftpUpdateUser
	oldProvision := ftpProvisionUser
	oldProvisionWithPassword := ftpProvisionUserWithPassword

	t.Cleanup(func() {
		database.DB = previousDB
		ftpLinuxUserExists = oldExists
		ftpLinuxLockUser = oldLock
		ftpLinuxUnlockUser = oldUnlock
		ftpLinuxSetPassword = oldSetPassword
		ftpLinuxRenameUser = oldRenameUser
		ftpUpdateUser = oldUpdateUser
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
	if err := db.Create(&models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "FTP",
		Enabled:     true,
		QuotaGB:     25,
	}).Error; err != nil {
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

func TestEnsureManagedFTPUserProjectionCreatesAndReuses(t *testing.T) {
	db := setupFTPReconcileTestDB(t)

	customer := models.Customer{
		CustomerCode: "CUS-FTP-P01",
		FullName:     "Managed FTP Projection Customer",
		Mobile:       "01000001001",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	account := models.CustomerInternetAccount{
		AccountCode:            "NET-FTP-P01",
		CustomerID:             customer.ID,
		RouterID:               1,
		PPPoEUsername:          "Par_002_morad",
		PPPoEPasswordEncrypted: "encrypted-placeholder",
		Status:                 "ACTIVE",
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-FTP-P01",
		Name:        "FTP Projection Package",
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "FTP",
		Enabled:     true,
		QuotaGB:     25,
	}).Error; err != nil {
		t.Fatal(err)
	}

	subscription := models.Subscription{
		SubscriptionCode:  "SUB-FTP-P01",
		CustomerID:        customer.ID,
		PackageID:         pkg.ID,
		InternetAccountID: &account.ID,
		Status:            "ACTIVE",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	server := models.FTPServer{
		Name:     "Primary FTP",
		Driver:   "linux",
		Host:     "127.0.0.1",
		Port:     21,
		Username: "admin",
		Password: "server-secret",
		RootPath: "/data/ftp",
		Status:   "ACTIVE",
	}
	if err := db.Create(&server).Error; err != nil {
		t.Fatal(err)
	}

	key := "PPPOE_FTP:" + fmt.Sprint(account.ID)

	entitlement := models.ServiceEntitlement{
		CustomerID:     customer.ID,
		SubscriptionID: &subscription.ID,
		ManagedKey:     &key,
		ServiceType:    "FTP",
		ServiceName:    "PPPoE FTP",
		Username:       account.PPPoEUsername,
		Status:         "ACTIVE",
		QuotaGB:        10,
	}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}

	first, created, err := EnsureManagedFTPUserProjection(
		&subscription,
		&account,
		&entitlement,
		&server,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected managed FTP projection to be created")
	}
	if first.ServiceEntitlementID == nil ||
		*first.ServiceEntitlementID != entitlement.ID {
		t.Fatalf("unexpected entitlement link: %v", first.ServiceEntitlementID)
	}
	if first.Username != account.PPPoEUsername {
		t.Fatalf("username = %q", first.Username)
	}
	if first.HomeDirectory != "/data/ftp/Par_002_morad" {
		t.Fatalf("home = %q", first.HomeDirectory)
	}
	if first.Password != "" {
		t.Fatal("managed FTP projection stored plaintext credential")
	}
	if first.StorageQuotaGB != 10 {
		t.Fatalf("quota = %d, want 10", first.StorageQuotaGB)
	}

	second, created, err := EnsureManagedFTPUserProjection(
		&subscription,
		&account,
		&entitlement,
		&server,
	)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("second reconciliation created duplicate projection")
	}
	if second.ID != first.ID {
		t.Fatalf("projection id = %d, want %d", second.ID, first.ID)
	}

	var count int64
	if err := db.Model(&models.FTPUser{}).
		Where("service_entitlement_id = ?", entitlement.ID).
		Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("managed FTP projection count = %d, want 1", count)
	}
}

func TestEnsureManagedFTPUserProjectionRejectsLegacyUsernameCollision(t *testing.T) {
	db := setupFTPReconcileTestDB(t)

	customer := models.Customer{
		CustomerCode: "CUS-FTP-P02",
		FullName:     "FTP Collision Customer",
		Mobile:       "01000001002",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	account := models.CustomerInternetAccount{
		AccountCode:            "NET-FTP-P02",
		CustomerID:             customer.ID,
		RouterID:               1,
		PPPoEUsername:          "legacy-user",
		PPPoEPasswordEncrypted: "encrypted-placeholder",
		Status:                 "ACTIVE",
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-FTP-P02",
		Name:        "FTP Collision Package",
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	subscription := models.Subscription{
		SubscriptionCode:  "SUB-FTP-P02",
		CustomerID:        customer.ID,
		PackageID:         pkg.ID,
		InternetAccountID: &account.ID,
		Status:            "ACTIVE",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	server := models.FTPServer{
		Name:     "Primary FTP",
		Driver:   "linux",
		Host:     "127.0.0.1",
		Port:     21,
		Username: "admin",
		Password: "server-secret",
		RootPath: "/data/ftp",
		Status:   "ACTIVE",
	}
	if err := db.Create(&server).Error; err != nil {
		t.Fatal(err)
	}

	legacy := models.FTPUser{
		CustomerID:           customer.ID,
		SubscriptionID:       subscription.ID,
		FTPServerID:          server.ID,
		Username:             "legacy-user",
		Password:             "legacy-secret",
		HomeDirectory:        "/data/ftp/legacy-user",
		StorageQuotaGB:       5,
		Status:               "active",
		ServiceEntitlementID: nil,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}

	key := "PPPOE_FTP:" + fmt.Sprint(account.ID)

	entitlement := models.ServiceEntitlement{
		CustomerID:     customer.ID,
		SubscriptionID: &subscription.ID,
		ManagedKey:     &key,
		ServiceType:    "FTP",
		ServiceName:    "PPPoE FTP",
		Username:       account.PPPoEUsername,
		Status:         "ACTIVE",
	}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}

	_, _, err := EnsureManagedFTPUserProjection(
		&subscription,
		&account,
		&entitlement,
		&server,
	)
	if err == nil {
		t.Fatal("expected legacy FTP username collision to be rejected")
	}

	if !strings.Contains(err.Error(), "already owned") {
		t.Fatalf("unexpected collision error: %v", err)
	}

	var stored models.FTPUser
	if err := db.First(&stored, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ServiceEntitlementID != nil {
		t.Fatal("legacy FTP user must remain unlinked")
	}
}

func createManagedFTPReconciliationFixture(
	t *testing.T,
	db *gorm.DB,
	subscriptionStatus string,
	password string,
) (*models.Subscription, string) {
	t.Helper()

	customer := models.Customer{
		CustomerCode: "CUS-MFTP-" + strings.ReplaceAll(t.Name(), "/", "-"),
		FullName:     "Managed FTP Customer",
		Mobile:       "01090000001",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	keyMaterial := "0123456789abcdef0123456789abcdef"

	encrypted := ""
	if password != "" {
		var err error
		encrypted, err = security.EncryptSecret(
			password,
			keyMaterial,
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	account := models.CustomerInternetAccount{
		AccountCode:            "NET-MFTP-" + fmt.Sprint(customer.ID),
		CustomerID:             customer.ID,
		RouterID:               1,
		PPPoEUsername:          "Par_002_morad",
		PPPoEPasswordEncrypted: encrypted,
		Status:                 subscriptionStatus,
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-MFTP-" + fmt.Sprint(customer.ID),
		Name:        "Managed FTP Package",
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "FTP",
		Enabled:     true,
		QuotaGB:     25,
	}).Error; err != nil {
		t.Fatal(err)
	}

	subscription := models.Subscription{
		SubscriptionCode:  "SUB-MFTP-" + fmt.Sprint(customer.ID),
		CustomerID:        customer.ID,
		PackageID:         pkg.ID,
		InternetAccountID: &account.ID,
		Status:            subscriptionStatus,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	server := models.FTPServer{
		Name:     "Managed FTP Server",
		Driver:   "linux",
		Host:     "163.128.79.10",
		Port:     21,
		Username: "admin",
		Password: "server-secret",
		RootPath: "/data/ftp",
		Status:   "ACTIVE",
	}
	if err := db.Create(&server).Error; err != nil {
		t.Fatal(err)
	}

	return &subscription, keyMaterial
}

func TestReconcileManagedFTPForSubscriptionProvisionsFromPPPoECredential(
	t *testing.T,
) {
	db := setupFTPReconcileTestDB(t)

	subscription, keyMaterial :=
		createManagedFTPReconciliationFixture(
			t,
			db,
			"ACTIVE",
			"pppoe-shared-password",
		)

	ftpLinuxUserExists = func(string) bool {
		return false
	}

	var provisionedPassword string
	ftpProvisionUserWithPassword = func(
		user *models.FTPUser,
		password string,
	) error {
		if user.Password != "" {
			t.Fatal("managed FTP user persisted plaintext password")
		}
		provisionedPassword = password
		return nil
	}

	result, err := ReconcileManagedFTPForSubscription(
		subscription,
		keyMaterial,
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Action != "PROVISION" || !result.Executed {
		t.Fatalf("unexpected result: %+v", result)
	}

	if provisionedPassword != "pppoe-shared-password" {
		t.Fatal("canonical PPPoE password was not supplied to provisioning")
	}

	var user models.FTPUser
	if err := db.First(&user, result.FTPUserID).Error; err != nil {
		t.Fatal(err)
	}

	if user.Password != "" {
		t.Fatal("PPPoE password was duplicated in ftp_users")
	}

	if user.ServiceEntitlementID == nil {
		t.Fatal("managed FTP user is not linked to entitlement")
	}
	if user.StorageQuotaGB != 25 {
		t.Fatalf("FTP quota = %d, want package policy quota 25", user.StorageQuotaGB)
	}
}

func TestReconcileManagedFTPForSubscriptionDisablesOnlyManagedPolicyFTP(t *testing.T) {
	db := setupFTPReconcileTestDB(t)
	subscription, keyMaterial := createManagedFTPReconciliationFixture(
		t,
		db,
		"ACTIVE",
		"pppoe-shared-password",
	)

	ftpLinuxUserExists = func(string) bool {
		return false
	}
	ftpProvisionUserWithPassword = func(*models.FTPUser, string) error {
		return nil
	}
	first, err := ReconcileManagedFTPForSubscription(subscription, keyMaterial)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Model(&models.PackageServicePolicy{}).
		Where("package_id = ? AND service_type = ?", subscription.PackageID, "FTP").
		Updates(map[string]interface{}{"enabled": false, "quota_gb": 0}).Error; err != nil {
		t.Fatal(err)
	}

	locked := false
	ftpLinuxUserExists = func(string) bool {
		return true
	}
	ftpLinuxLockUser = func(string) error {
		locked = true
		return nil
	}
	result, err := ReconcileManagedFTPForSubscription(subscription, keyMaterial)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "LOCK_POLICY_DISABLED" || !locked {
		t.Fatalf("unexpected disabled-policy result: %+v", result)
	}

	var entitlement models.ServiceEntitlement
	if err := db.First(&entitlement, first.ServiceEntitlementID).Error; err != nil {
		t.Fatal(err)
	}
	if entitlement.Status != "DISABLED" || entitlement.QuotaGB != 0 {
		t.Fatalf("managed entitlement = status:%s quota:%d, want disabled zero-quota", entitlement.Status, entitlement.QuotaGB)
	}

	var user models.FTPUser
	if err := db.First(&user, first.FTPUserID).Error; err != nil {
		t.Fatal(err)
	}
	if user.Status != "DISABLED" {
		t.Fatalf("managed FTP user status = %s, want DISABLED", user.Status)
	}
}

func TestReconcileManagedFTPForSubscriptionSkipsPackageWithoutFTPPolicy(t *testing.T) {
	db := setupFTPReconcileTestDB(t)
	subscription, keyMaterial := createManagedFTPReconciliationFixture(
		t,
		db,
		"ACTIVE",
		"pppoe-shared-password",
	)
	if err := db.Where("package_id = ? AND service_type = ?", subscription.PackageID, "FTP").
		Delete(&models.PackageServicePolicy{}).Error; err != nil {
		t.Fatal(err)
	}

	result, err := ReconcileManagedFTPForSubscription(subscription, keyMaterial)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "SKIPPED_PACKAGE_POLICY_DISABLED" {
		t.Fatalf("action = %q, want no-policy skip", result.Action)
	}
	var count int64
	if err := db.Model(&models.ServiceEntitlement{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("entitlement count = %d, want no managed entitlement", count)
	}
}

func TestReconcileManagedFTPForSubscriptionSynchronizesExistingLinuxPassword(
	t *testing.T,
) {
	db := setupFTPReconcileTestDB(t)

	subscription, keyMaterial :=
		createManagedFTPReconciliationFixture(
			t,
			db,
			"ACTIVE",
			"rotated-pppoe-password",
		)

	ftpLinuxUserExists = func(string) bool {
		return true
	}

	var passwordSet string
	var unlocked bool

	ftpLinuxSetPassword = func(
		username,
		password string,
	) error {
		passwordSet = password
		return nil
	}

	ftpLinuxUnlockUser = func(string) error {
		unlocked = true
		return nil
	}

	result, err := ReconcileManagedFTPForSubscription(
		subscription,
		keyMaterial,
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Action != "PASSWORD_SYNC_AND_UNLOCK" {
		t.Fatalf("unexpected action: %s", result.Action)
	}

	if passwordSet != "rotated-pppoe-password" {
		t.Fatal("Linux FTP password was not synchronized")
	}

	if !unlocked {
		t.Fatal("existing Linux FTP user was not unlocked")
	}
}

func TestReconcileManagedFTPForSubscriptionLocksWithoutCredential(
	t *testing.T,
) {
	db := setupFTPReconcileTestDB(t)

	subscription, keyMaterial :=
		createManagedFTPReconciliationFixture(
			t,
			db,
			"SUSPENDED",
			"",
		)

	ftpLinuxUserExists = func(string) bool {
		return true
	}

	var locked bool
	ftpLinuxLockUser = func(string) error {
		locked = true
		return nil
	}

	result, err := ReconcileManagedFTPForSubscription(
		subscription,
		keyMaterial,
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Status != "SUSPENDED" ||
		result.Action != "LOCK" ||
		!locked {
		t.Fatalf("unexpected suspension result: %+v", result)
	}
}

func TestReconcileManagedFTPForSubscriptionRejectsActiveBlankCredential(
	t *testing.T,
) {
	db := setupFTPReconcileTestDB(t)

	subscription, keyMaterial :=
		createManagedFTPReconciliationFixture(
			t,
			db,
			"ACTIVE",
			"",
		)

	called := false
	ftpProvisionUserWithPassword = func(
		*models.FTPUser,
		string,
	) error {
		called = true
		return nil
	}

	_, err := ReconcileManagedFTPForSubscription(
		subscription,
		keyMaterial,
	)
	if err == nil {
		t.Fatal("expected active blank PPPoE credential to fail")
	}

	if called {
		t.Fatal("blank credential must not trigger Linux provisioning")
	}

	if !strings.Contains(
		err.Error(),
		"PPPoE credential is not configured",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func createManagedFTPProjectionRenameFixture(
	t *testing.T,
	db *gorm.DB,
) (
	*models.Subscription,
	*models.CustomerInternetAccount,
	*models.ServiceEntitlement,
	*models.FTPServer,
	*models.FTPUser,
) {
	t.Helper()

	customer := models.Customer{
		CustomerCode: "CUS-FTP-REN-" + strings.ReplaceAll(t.Name(), "/", "-"),
		FullName:     "FTP Rename Customer",
		Mobile:       "01080000001",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	account := models.CustomerInternetAccount{
		AccountCode:            "NET-FTP-REN-" + fmt.Sprint(customer.ID),
		CustomerID:             customer.ID,
		RouterID:               1,
		PPPoEUsername:          "new-pppoe-user",
		PPPoEPasswordEncrypted: "encrypted-placeholder",
		Status:                 "ACTIVE",
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-FTP-REN-" + fmt.Sprint(customer.ID),
		Name:        "FTP Rename Package",
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	subscription := models.Subscription{
		SubscriptionCode:  "SUB-FTP-REN-" + fmt.Sprint(customer.ID),
		CustomerID:        customer.ID,
		PackageID:         pkg.ID,
		InternetAccountID: &account.ID,
		Status:            "ACTIVE",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}

	server := models.FTPServer{
		Name:     "FTP Rename Server",
		Driver:   "linux",
		Host:     "127.0.0.1",
		Port:     21,
		Username: "admin",
		Password: "server-secret",
		RootPath: "/data/ftp",
		Status:   "ACTIVE",
	}
	if err := db.Create(&server).Error; err != nil {
		t.Fatal(err)
	}

	key := "PPPOE_FTP:" + fmt.Sprint(account.ID)
	entitlement := models.ServiceEntitlement{
		CustomerID:     customer.ID,
		SubscriptionID: &subscription.ID,
		ManagedKey:     &key,
		ServiceType:    "FTP",
		ServiceName:    "PPPoE FTP",
		Username:       account.PPPoEUsername,
		Status:         "ACTIVE",
		QuotaGB:        10,
	}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}

	user := models.FTPUser{
		CustomerID:           customer.ID,
		SubscriptionID:       subscription.ID,
		ServiceEntitlementID: &entitlement.ID,
		FTPServerID:          server.ID,
		Username:             "old-pppoe-user",
		Password:             "",
		HomeDirectory:        "/data/ftp/old-pppoe-user",
		StorageQuotaGB:       10,
		Status:               "ACTIVE",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	return &subscription, &account, &entitlement, &server, &user
}

func TestEnsureManagedFTPUserProjectionRenamesExistingLinuxIdentity(
	t *testing.T,
) {
	db := setupFTPReconcileTestDB(t)

	subscription, account, entitlement, server, user :=
		createManagedFTPProjectionRenameFixture(t, db)

	ftpLinuxUserExists = func(username string) bool {
		return username == "old-pppoe-user"
	}

	var calls int
	ftpLinuxRenameUser = func(
		oldUsername,
		newUsername,
		oldHome,
		newHome string,
	) error {
		calls++

		if oldUsername != "old-pppoe-user" ||
			newUsername != "new-pppoe-user" ||
			oldHome != "/data/ftp/old-pppoe-user" ||
			newHome != "/data/ftp/new-pppoe-user" {
			t.Fatalf(
				"unexpected rename: %q %q %q %q",
				oldUsername,
				newUsername,
				oldHome,
				newHome,
			)
		}

		return nil
	}

	updated, created, err := EnsureManagedFTPUserProjection(
		subscription,
		account,
		entitlement,
		server,
	)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("existing projection must not be recreated")
	}
	if calls != 1 {
		t.Fatalf("rename calls = %d, want 1", calls)
	}
	if updated.ID != user.ID {
		t.Fatalf("projection id = %d, want %d", updated.ID, user.ID)
	}

	var stored models.FTPUser
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Username != "new-pppoe-user" {
		t.Fatalf("stored username = %q", stored.Username)
	}
	if stored.HomeDirectory != "/data/ftp/new-pppoe-user" {
		t.Fatalf("stored home = %q", stored.HomeDirectory)
	}
}

func TestEnsureManagedFTPUserProjectionRejectsRenameCollisionBeforeLinux(
	t *testing.T,
) {
	db := setupFTPReconcileTestDB(t)

	subscription, account, entitlement, server, _ :=
		createManagedFTPProjectionRenameFixture(t, db)

	collision := models.FTPUser{
		CustomerID:     subscription.CustomerID,
		SubscriptionID: subscription.ID,
		FTPServerID:    server.ID,
		Username:       account.PPPoEUsername,
		Password:       "legacy",
		HomeDirectory:  "/data/ftp/" + account.PPPoEUsername,
		Status:         "active",
	}
	if err := db.Create(&collision).Error; err != nil {
		t.Fatal(err)
	}

	called := false
	ftpLinuxRenameUser = func(
		string,
		string,
		string,
		string,
	) error {
		called = true
		return nil
	}

	_, _, err := EnsureManagedFTPUserProjection(
		subscription,
		account,
		entitlement,
		server,
	)
	if err == nil {
		t.Fatal("expected FTP username collision")
	}
	if called {
		t.Fatal("Linux rename ran before DB collision was rejected")
	}
}

func TestEnsureManagedFTPUserProjectionAllowsRenameWhenOldLinuxMissing(
	t *testing.T,
) {
	db := setupFTPReconcileTestDB(t)

	subscription, account, entitlement, server, user :=
		createManagedFTPProjectionRenameFixture(t, db)

	ftpLinuxUserExists = func(string) bool {
		return false
	}

	called := false
	ftpLinuxRenameUser = func(
		string,
		string,
		string,
		string,
	) error {
		called = true
		return nil
	}

	_, _, err := EnsureManagedFTPUserProjection(
		subscription,
		account,
		entitlement,
		server,
	)
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("Linux rename must not run when old Linux user is absent")
	}

	var stored models.FTPUser
	if err := db.First(&stored, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Username != account.PPPoEUsername {
		t.Fatalf("stored username = %q", stored.Username)
	}
}

func TestEnsureManagedFTPUserProjectionRejectsUnexpectedNewLinuxUser(
	t *testing.T,
) {
	db := setupFTPReconcileTestDB(t)

	subscription, account, entitlement, server, _ :=
		createManagedFTPProjectionRenameFixture(t, db)

	ftpLinuxUserExists = func(username string) bool {
		return username == account.PPPoEUsername
	}

	_, _, err := EnsureManagedFTPUserProjection(
		subscription,
		account,
		entitlement,
		server,
	)
	if err == nil {
		t.Fatal("expected unexpected new Linux user to be rejected")
	}
	if !strings.Contains(err.Error(), "exists while managed FTP projection") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureManagedFTPUserProjectionRollsBackLinuxRenameOnDBFailure(
	t *testing.T,
) {
	db := setupFTPReconcileTestDB(t)

	subscription, account, entitlement, server, _ :=
		createManagedFTPProjectionRenameFixture(t, db)

	ftpLinuxUserExists = func(username string) bool {
		return username == "old-pppoe-user"
	}

	var renames [][2]string
	ftpLinuxRenameUser = func(
		oldUsername,
		newUsername,
		oldHome,
		newHome string,
	) error {
		renames = append(
			renames,
			[2]string{oldUsername, newUsername},
		)
		return nil
	}

	ftpUpdateUser = func(*models.FTPUser) error {
		return errors.New("simulated FTP projection update failure")
	}

	_, _, err := EnsureManagedFTPUserProjection(
		subscription,
		account,
		entitlement,
		server,
	)
	if err == nil {
		t.Fatal("expected DB update failure")
	}

	if len(renames) != 2 {
		t.Fatalf("rename calls = %d, want 2", len(renames))
	}

	if renames[0] != [2]string{"old-pppoe-user", "new-pppoe-user"} {
		t.Fatalf("forward rename = %#v", renames[0])
	}

	if renames[1] != [2]string{"new-pppoe-user", "old-pppoe-user"} {
		t.Fatalf("rollback rename = %#v", renames[1])
	}
}

func TestBuildManagedFTPUserRejectsLinuxUnsafeUsername(
	t *testing.T,
) {
	subscription := &models.Subscription{
		CustomerID: 1,
	}
	subscription.ID = 1

	entitlement := &models.ServiceEntitlement{
		Status: "ACTIVE",
	}
	entitlement.ID = 1

	server := &models.FTPServer{
		RootPath: "/data/ftp",
	}
	server.ID = 1

	tests := []string{
		"bad user",
		"bad:name",
		"-leading-hyphen",
	}

	for _, username := range tests {
		t.Run(username, func(t *testing.T) {
			account := &models.CustomerInternetAccount{
				PPPoEUsername: username,
			}
			account.ID = 1

			if _, err := BuildManagedFTPUser(
				subscription,
				account,
				entitlement,
				server,
			); err == nil {
				t.Fatalf(
					"expected username %q to be rejected",
					username,
				)
			}
		})
	}
}
