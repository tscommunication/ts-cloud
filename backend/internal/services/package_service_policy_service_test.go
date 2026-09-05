package services

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupPackageServicePolicyTestDB(
	t *testing.T,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			"file:"+t.Name()+"?mode=memory&cache=shared",
		),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previousDB
	})

	if err := db.AutoMigrate(
		&models.Package{},
		&models.PackageServicePolicy{},
	); err != nil {
		t.Fatal(err)
	}

	return db
}

func createPackageServicePolicyTestPackage(
	t *testing.T,
	db *gorm.DB,
) models.Package {
	t.Helper()

	pkg := models.Package{
		PackageCode: "PKG-SERVICE-POLICY",
		Name:        "Service Policy Package",
		Status:      "ACTIVE",
	}

	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	return pkg
}

func TestSavePackageServicePolicyNormalizesAndUpsertsFTP(
	t *testing.T,
) {
	db := setupPackageServicePolicyTestDB(t)
	pkg := createPackageServicePolicyTestPackage(t, db)

	row := models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: " ftp ",
		Enabled:     true,
		QuotaGB:     10,
		ConfigJSON:  `{"mode":"managed"}`,
		Remarks:     "  default FTP  ",
	}

	if err := SavePackageServicePolicy(&row); err != nil {
		t.Fatal(err)
	}

	if row.ID == 0 {
		t.Fatal("expected saved policy id")
	}
	if row.ServiceType != "FTP" {
		t.Fatalf(
			"service type = %q, want FTP",
			row.ServiceType,
		)
	}
	if row.Remarks != "default FTP" {
		t.Fatalf("remarks = %q", row.Remarks)
	}

	firstID := row.ID

	row.Enabled = true
	row.QuotaGB = 25

	if err := SavePackageServicePolicy(&row); err != nil {
		t.Fatal(err)
	}

	if row.ID != firstID {
		t.Fatalf(
			"upsert created id %d, want existing id %d",
			row.ID,
			firstID,
		)
	}

	var count int64
	if err := db.Model(&models.PackageServicePolicy{}).
		Where(
			"package_id = ? AND service_type = ?",
			pkg.ID,
			"FTP",
		).
		Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf("FTP policy count = %d, want 1", count)
	}

	saved, err := GetPackageServicePolicy(
		pkg.ID,
		"ftp",
	)
	if err != nil {
		t.Fatal(err)
	}
	if saved.QuotaGB != 25 {
		t.Fatalf(
			"saved quota = %d, want 25",
			saved.QuotaGB,
		)
	}
}

func TestSavePackageServicePolicyRejectsEnabledFTPWithoutQuota(
	t *testing.T,
) {
	db := setupPackageServicePolicyTestDB(t)
	pkg := createPackageServicePolicyTestPackage(t, db)

	row := models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "FTP",
		Enabled:     true,
		QuotaGB:     0,
	}

	if err := SavePackageServicePolicy(&row); err == nil {
		t.Fatal(
			"expected enabled FTP without quota to be rejected",
		)
	}
}

func TestCreatePackageWithFTPPolicyIsAtomic(t *testing.T) {
	db := setupPackageServicePolicyTestDB(t)
	pkg := &models.Package{
		PackageCode: "PKG-ATOMIC-CREATE",
		Name:        "Atomic create",
		Status:      "ACTIVE",
	}

	if err := CreatePackageWithFTPPolicy(pkg, true, 0); err == nil {
		t.Fatal("expected enabled FTP without quota to fail")
	}

	var packageCount int64
	if err := db.Model(&models.Package{}).Count(&packageCount).Error; err != nil {
		t.Fatal(err)
	}
	if packageCount != 0 {
		t.Fatalf("package count = %d, want 0 after rejected save", packageCount)
	}

	if err := CreatePackageWithFTPPolicy(pkg, true, 25); err != nil {
		t.Fatal(err)
	}
	policy, err := GetPackageServicePolicy(pkg.ID, "FTP")
	if err != nil {
		t.Fatal(err)
	}
	if !policy.Enabled || policy.QuotaGB != 25 {
		t.Fatalf("FTP policy = enabled:%t quota:%d, want enabled: true quota:25", policy.Enabled, policy.QuotaGB)
	}
}

func TestUpdatePackageWithFTPPolicyPreservesLegacyNoPolicy(t *testing.T) {
	db := setupPackageServicePolicyTestDB(t)
	pkg := createPackageServicePolicyTestPackage(t, db)
	pkg.Name = "Updated legacy package"

	if err := UpdatePackageWithFTPPolicy(&pkg, false, 0); err != nil {
		t.Fatal(err)
	}

	var policyCount int64
	if err := db.Model(&models.PackageServicePolicy{}).Where("package_id = ?", pkg.ID).Count(&policyCount).Error; err != nil {
		t.Fatal(err)
	}
	if policyCount != 0 {
		t.Fatalf("policy count = %d, want legacy package to remain policy-free", policyCount)
	}

	pkg.Name = "Rejected update"
	if err := UpdatePackageWithFTPPolicy(&pkg, true, 0); err == nil {
		t.Fatal("expected enabled FTP without quota to fail")
	}
	var saved models.Package
	if err := db.First(&saved, pkg.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Name != "Updated legacy package" {
		t.Fatalf("package name = %q, want transaction rollback", saved.Name)
	}
}

func TestUpdatePackageWithFTPPolicyUpdatesExistingPolicy(t *testing.T) {
	db := setupPackageServicePolicyTestDB(t)
	pkg := createPackageServicePolicyTestPackage(t, db)
	if err := SavePackageServicePolicy(&models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "FTP",
		Enabled:     true,
		QuotaGB:     15,
	}); err != nil {
		t.Fatal(err)
	}

	if err := UpdatePackageWithFTPPolicy(&pkg, false, 0); err != nil {
		t.Fatal(err)
	}
	policy, err := GetPackageServicePolicy(pkg.ID, "FTP")
	if err != nil {
		t.Fatal(err)
	}
	if policy.Enabled || policy.QuotaGB != 0 {
		t.Fatalf("FTP policy = enabled:%t quota:%d, want disabled zero-quota policy", policy.Enabled, policy.QuotaGB)
	}
}

func TestSavePackageServicePolicyAllowsDisabledFTPWithoutQuota(
	t *testing.T,
) {
	db := setupPackageServicePolicyTestDB(t)
	pkg := createPackageServicePolicyTestPackage(t, db)

	row := models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "FTP",
		Enabled:     false,
		QuotaGB:     0,
	}

	if err := SavePackageServicePolicy(&row); err != nil {
		t.Fatal(err)
	}
}

func TestSavePackageServicePolicyAllowsQuotaFreeJellyfin(
	t *testing.T,
) {
	db := setupPackageServicePolicyTestDB(t)
	pkg := createPackageServicePolicyTestPackage(t, db)

	row := models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "JELLYFIN",
		Enabled:     true,
		QuotaGB:     0,
	}

	if err := SavePackageServicePolicy(&row); err != nil {
		t.Fatal(err)
	}
}

func TestSavePackageServicePolicyRejectsInvalidTypeAndJSON(
	t *testing.T,
) {
	db := setupPackageServicePolicyTestDB(t)
	pkg := createPackageServicePolicyTestPackage(t, db)

	invalidType := models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "UNKNOWN",
	}

	if err := SavePackageServicePolicy(
		&invalidType,
	); err == nil {
		t.Fatal("expected invalid service type rejection")
	}

	invalidJSON := models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "JELLYFIN",
		Enabled:     true,
		ConfigJSON:  "{broken",
	}

	if err := SavePackageServicePolicy(
		&invalidJSON,
	); err == nil {
		t.Fatal("expected invalid config JSON rejection")
	}
}

func TestSavePackageServicePolicyRestoresSoftDeletedPolicy(
	t *testing.T,
) {
	db := setupPackageServicePolicyTestDB(t)
	pkg := createPackageServicePolicyTestPackage(t, db)

	first := models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "FTP",
		Enabled:     true,
		QuotaGB:     10,
	}

	if err := SavePackageServicePolicy(&first); err != nil {
		t.Fatal(err)
	}

	if err := db.Delete(&first).Error; err != nil {
		t.Fatal(err)
	}

	replacement := models.PackageServicePolicy{
		PackageID:   pkg.ID,
		ServiceType: "FTP",
		Enabled:     true,
		QuotaGB:     30,
	}

	if err := SavePackageServicePolicy(
		&replacement,
	); err != nil {
		t.Fatal(err)
	}

	if replacement.ID != first.ID {
		t.Fatalf(
			"restored id = %d, want %d",
			replacement.ID,
			first.ID,
		)
	}

	if replacement.DeletedAt.Valid {
		t.Fatal("restored policy remains soft deleted")
	}

	if replacement.QuotaGB != 30 {
		t.Fatalf(
			"restored quota = %d, want 30",
			replacement.QuotaGB,
		)
	}
}
