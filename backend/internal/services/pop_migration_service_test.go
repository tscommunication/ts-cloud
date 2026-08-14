package services

import (
	"testing"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPOPMigrationTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open("file:"+name+"?mode=memory&cache=shared"),
		&gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.POP{},
		&models.Agent{},
		&models.AgentPOP{},
		&models.Customer{},
		&models.NetworkRouter{},
	); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previousDB
	})

	return db
}

func TestMigratePOPMovesAssignmentsAndKeepsSourceInactive(t *testing.T) {
	db := setupPOPMigrationTestDB(t, "pop_migration_success")

	sourcePOP := models.POP{
		Code:   "POP-SOURCE",
		Name:   "Source POP",
		Status: "ACTIVE",
	}

	targetPOP := models.POP{
		Code:   "POP-TARGET",
		Name:   "Target POP",
		Status: "ACTIVE",
	}

	if err := db.Create(&sourcePOP).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&targetPOP).Error; err != nil {
		t.Fatal(err)
	}

	agent := models.Agent{
		Code:   "AGENT-1",
		Name:   "Agent One",
		POPID:  sourcePOP.ID,
		Status: "ACTIVE",
	}

	if err := db.Create(&agent).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&models.AgentPOP{
		AgentID: agent.ID,
		POPID:   sourcePOP.ID,
	}).Error; err != nil {
		t.Fatal(err)
	}

	customerPOPID := sourcePOP.ID

	customer := models.Customer{
		CustomerCode: "CUSTOMER-1",
		FullName:     "Test Customer",
		Mobile:       "01700000000",
		PopID:        &customerPOPID,
		Status:       "ACTIVE",
		BillingDay:   1,
	}

	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	routerPOPID := sourcePOP.ID

	router := models.NetworkRouter{
		Code:                 "RTR-1",
		Name:                 "Router One",
		POPID:                &routerPOPID,
		Host:                 "192.0.2.1",
		APIPort:              8729,
		APIUsername:          "api-user",
		APIPasswordEncrypted: "encrypted",
		UseTLS:               true,
		Status:               "ACTIVE",
		ConnectivityStatus:   "UNKNOWN",
		APIStatus:            "UNKNOWN",
	}

	if err := db.Create(&router).Error; err != nil {
		t.Fatal(err)
	}

	result, err := MigratePOP(sourcePOP.ID, targetPOP.ID)
	if err != nil {
		t.Fatal(err)
	}

	if result.SourcePOPID != sourcePOP.ID {
		t.Fatalf(
			"expected source POP ID %d, got %d",
			sourcePOP.ID,
			result.SourcePOPID,
		)
	}

	if result.TargetPOPID != targetPOP.ID {
		t.Fatalf(
			"expected target POP ID %d, got %d",
			targetPOP.ID,
			result.TargetPOPID,
		)
	}

	if result.Customers != 1 {
		t.Fatalf(
			"expected 1 migrated customer, got %d",
			result.Customers,
		)
	}

	if result.PrimaryAgents != 1 {
		t.Fatalf(
			"expected 1 migrated primary agent, got %d",
			result.PrimaryAgents,
		)
	}

	if result.AgentMemberships != 1 {
		t.Fatalf(
			"expected 1 migrated agent POP membership, got %d",
			result.AgentMemberships,
		)
	}

	if result.Routers != 1 {
		t.Fatalf(
			"expected 1 migrated router, got %d",
			result.Routers,
		)
	}

	if err := db.First(&customer, customer.ID).Error; err != nil {
		t.Fatal(err)
	}

	if customer.PopID == nil || *customer.PopID != targetPOP.ID {
		t.Fatal("customer POP assignment was not migrated")
	}

	if err := db.First(&agent, agent.ID).Error; err != nil {
		t.Fatal(err)
	}

	if agent.POPID != targetPOP.ID {
		t.Fatal("agent primary POP assignment was not migrated")
	}

	var sourceMembershipCount int64
	if err := db.Model(&models.AgentPOP{}).
		Where(
			"agent_id = ? AND pop_id = ?",
			agent.ID,
			sourcePOP.ID,
		).
		Count(&sourceMembershipCount).Error; err != nil {
		t.Fatal(err)
	}

	if sourceMembershipCount != 0 {
		t.Fatal("source POP membership should be removed after migration")
	}

	var targetMembershipCount int64
	if err := db.Model(&models.AgentPOP{}).
		Where(
			"agent_id = ? AND pop_id = ?",
			agent.ID,
			targetPOP.ID,
		).
		Count(&targetMembershipCount).Error; err != nil {
		t.Fatal(err)
	}

	if targetMembershipCount != 1 {
		t.Fatalf(
			"expected target POP membership to exist once, got %d",
			targetMembershipCount,
		)
	}

	if err := db.First(&router, router.ID).Error; err != nil {
		t.Fatal(err)
	}

	if router.POPID == nil || *router.POPID != targetPOP.ID {
		t.Fatal("router POP assignment was not migrated")
	}

	var migratedSource models.POP
	if err := db.First(&migratedSource, sourcePOP.ID).Error; err != nil {
		t.Fatal(err)
	}

	if migratedSource.Status != "INACTIVE" {
		t.Fatalf(
			"source POP should be INACTIVE after migration, got %s",
			migratedSource.Status,
		)
	}

	if migratedSource.DeletedAt.Valid {
		t.Fatal("source POP should not be deleted by migration")
	}

	var migratedTarget models.POP
	if err := db.First(&migratedTarget, targetPOP.ID).Error; err != nil {
		t.Fatal(err)
	}

	if migratedTarget.Status != "ACTIVE" {
		t.Fatalf(
			"target POP should remain ACTIVE, got %s",
			migratedTarget.Status,
		)
	}
}

func TestMigratePOPDoesNotDuplicateExistingAgentMembership(t *testing.T) {
	db := setupPOPMigrationTestDB(t, "pop_migration_duplicate_membership")

	sourcePOP := models.POP{
		Code:   "POP-SOURCE",
		Name:   "Source POP",
		Status: "ACTIVE",
	}

	targetPOP := models.POP{
		Code:   "POP-TARGET",
		Name:   "Target POP",
		Status: "ACTIVE",
	}

	if err := db.Create(&sourcePOP).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&targetPOP).Error; err != nil {
		t.Fatal(err)
	}

	agent := models.Agent{
		Code:   "AGENT-1",
		Name:   "Agent One",
		POPID:  sourcePOP.ID,
		Status: "ACTIVE",
	}

	if err := db.Create(&agent).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&models.AgentPOP{
		AgentID: agent.ID,
		POPID:   sourcePOP.ID,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&models.AgentPOP{
		AgentID: agent.ID,
		POPID:   targetPOP.ID,
	}).Error; err != nil {
		t.Fatal(err)
	}

	result, err := MigratePOP(sourcePOP.ID, targetPOP.ID)
	if err != nil {
		t.Fatal(err)
	}

	if result.AgentMemberships != 0 {
		t.Fatalf(
			"expected 0 newly-created memberships, got %d",
			result.AgentMemberships,
		)
	}

	var sourceCount int64
	if err := db.Model(&models.AgentPOP{}).
		Where(
			"agent_id = ? AND pop_id = ?",
			agent.ID,
			sourcePOP.ID,
		).
		Count(&sourceCount).Error; err != nil {
		t.Fatal(err)
	}

	if sourceCount != 0 {
		t.Fatal("source POP membership should be removed")
	}

	var targetCount int64
	if err := db.Model(&models.AgentPOP{}).
		Where(
			"agent_id = ? AND pop_id = ?",
			agent.ID,
			targetPOP.ID,
		).
		Count(&targetCount).Error; err != nil {
		t.Fatal(err)
	}

	if targetCount != 1 {
		t.Fatalf(
			"expected exactly one target membership, got %d",
			targetCount,
		)
	}
}

func TestMigratePOPRejectsSameSourceAndTarget(t *testing.T) {
	setupPOPMigrationTestDB(t, "pop_migration_same_pop")

	_, err := MigratePOP(1, 1)
	if err == nil {
		t.Fatal("expected same source and target POP migration to fail")
	}
}

func TestMigratePOPRejectsInactiveTarget(t *testing.T) {
	db := setupPOPMigrationTestDB(t, "pop_migration_inactive_target")

	sourcePOP := models.POP{
		Code:   "POP-SOURCE",
		Name:   "Source POP",
		Status: "ACTIVE",
	}

	targetPOP := models.POP{
		Code:   "POP-TARGET",
		Name:   "Target POP",
		Status: "INACTIVE",
	}

	if err := db.Create(&sourcePOP).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&targetPOP).Error; err != nil {
		t.Fatal(err)
	}

	_, err := MigratePOP(sourcePOP.ID, targetPOP.ID)
	if err == nil {
		t.Fatal("expected migration to inactive target POP to fail")
	}

	var sourceAfter models.POP
	if err := db.First(&sourceAfter, sourcePOP.ID).Error; err != nil {
		t.Fatal(err)
	}

	if sourceAfter.Status != "ACTIVE" {
		t.Fatal("failed migration must not change source POP status")
	}

	if sourceAfter.DeletedAt.Valid {
		t.Fatal("failed migration must not delete source POP")
	}
}

func TestDeletePOPBlocksLinkedDependencies(t *testing.T) {
	db := setupPOPMigrationTestDB(t, "pop_delete_dependencies")

	pop := models.POP{
		Code:   "POP-1",
		Name:   "POP One",
		Status: "ACTIVE",
	}

	if err := db.Create(&pop).Error; err != nil {
		t.Fatal(err)
	}

	customerPOPID := pop.ID

	customer := models.Customer{
		CustomerCode: "CUSTOMER-1",
		FullName:     "Customer One",
		Mobile:       "01700000001",
		PopID:        &customerPOPID,
		Status:       "ACTIVE",
		BillingDay:   1,
	}

	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	if err := DeletePOP(pop.ID); err == nil {
		t.Fatal("expected deleting POP with dependencies to fail")
	}

	var existing models.POP
	if err := db.First(&existing, pop.ID).Error; err != nil {
		t.Fatal(err)
	}

	if existing.DeletedAt.Valid {
		t.Fatal("POP with dependencies must not be deleted")
	}
}

func TestDeletePOPDeletesEmptyPOPOnlyWhenExplicitlyRequested(t *testing.T) {
	db := setupPOPMigrationTestDB(t, "pop_delete_empty")

	pop := models.POP{
		Code:   "POP-EMPTY",
		Name:   "Empty POP",
		Status: "ACTIVE",
	}

	if err := db.Create(&pop).Error; err != nil {
		t.Fatal(err)
	}

	if err := DeletePOP(pop.ID); err != nil {
		t.Fatal(err)
	}

	var visibleCount int64
	if err := db.Model(&models.POP{}).
		Where("id = ?", pop.ID).
		Count(&visibleCount).Error; err != nil {
		t.Fatal(err)
	}

	if visibleCount != 0 {
		t.Fatal("explicit delete should soft-delete an empty POP")
	}

	var deleted models.POP
	if err := db.Unscoped().
		First(&deleted, pop.ID).Error; err != nil {
		t.Fatal(err)
	}

	if !deleted.DeletedAt.Valid {
		t.Fatal("explicit delete should set deleted_at")
	}

	if deleted.Status != "INACTIVE" {
		t.Fatalf(
			"deleted POP should be INACTIVE, got %s",
			deleted.Status,
		)
	}
}
