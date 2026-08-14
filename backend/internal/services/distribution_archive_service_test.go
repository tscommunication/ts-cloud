package services

import (
	"testing"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDistributionArchiveTestDB(
	t *testing.T,
	name string,
) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			"file:"+name+"?mode=memory&cache=shared",
		),
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
		&models.User{},
		&models.AgentCollection{},
		&models.AgentSettlement{},
		&models.Payment{},
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

func TestDeleteAndRestoreAgentPreservesMemberships(
	t *testing.T,
) {
	db := setupDistributionArchiveTestDB(
		t,
		"agent_archive_restore",
	)

	pop := models.POP{
		Code:   "POP-ARCHIVE-A",
		Name:   "Archive POP A",
		Status: "ACTIVE",
	}
	if err := db.Create(&pop).Error; err != nil {
		t.Fatal(err)
	}

	agent := models.Agent{
		Code:   "AGENT-ARCHIVE-A",
		Name:   "Archive Agent A",
		POPID:  pop.ID,
		Status: "ACTIVE",
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatal(err)
	}

	link := models.AgentPOP{
		AgentID: agent.ID,
		POPID:   pop.ID,
	}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}

	if err := DeleteAgent(agent.ID); err != nil {
		t.Fatal(err)
	}

	var activeCount int64
	if err := db.Model(&models.Agent{}).
		Where("id = ?", agent.ID).
		Count(&activeCount).Error; err != nil {
		t.Fatal(err)
	}

	if activeCount != 0 {
		t.Fatalf(
			"archived agent should not appear in active scope, got %d",
			activeCount,
		)
	}

	var archived models.Agent
	if err := db.
		Unscoped().
		First(&archived, agent.ID).Error; err != nil {
		t.Fatal(err)
	}

	if !archived.DeletedAt.Valid {
		t.Fatal("agent should be soft-deleted")
	}

	if archived.Status != "INACTIVE" {
		t.Fatalf(
			"archived agent should be INACTIVE, got %s",
			archived.Status,
		)
	}

	var membershipCount int64
	if err := db.Model(&models.AgentPOP{}).
		Where("agent_id = ?", agent.ID).
		Count(&membershipCount).Error; err != nil {
		t.Fatal(err)
	}

	if membershipCount != 1 {
		t.Fatalf(
			"archived agent should retain POP membership, got %d",
			membershipCount,
		)
	}

	rows, err := ListArchivedAgents()
	if err != nil {
		t.Fatal(err)
	}

	if len(rows) != 1 || rows[0].ID != agent.ID {
		t.Fatalf(
			"expected archived agent %d, got %+v",
			agent.ID,
			rows,
		)
	}

	restored, err := RestoreAgent(agent.ID)
	if err != nil {
		t.Fatal(err)
	}

	if restored.Status != "INACTIVE" {
		t.Fatalf(
			"restored agent should remain INACTIVE, got %s",
			restored.Status,
		)
	}

	if restored.DeletedAt.Valid {
		t.Fatal("restored agent should not be deleted")
	}

	var restoredMembershipCount int64
	if err := db.Model(&models.AgentPOP{}).
		Where("agent_id = ?", agent.ID).
		Count(&restoredMembershipCount).Error; err != nil {
		t.Fatal(err)
	}

	if restoredMembershipCount != 1 {
		t.Fatalf(
			"restored agent should retain POP membership, got %d",
			restoredMembershipCount,
		)
	}
}

func TestDeleteAndRestorePOP(t *testing.T) {
	db := setupDistributionArchiveTestDB(
		t,
		"pop_archive_restore",
	)

	pop := models.POP{
		Code:   "POP-ARCHIVE-B",
		Name:   "Archive POP B",
		Status: "ACTIVE",
	}
	if err := db.Create(&pop).Error; err != nil {
		t.Fatal(err)
	}

	if err := DeletePOP(pop.ID); err != nil {
		t.Fatal(err)
	}

	var activeCount int64
	if err := db.Model(&models.POP{}).
		Where("id = ?", pop.ID).
		Count(&activeCount).Error; err != nil {
		t.Fatal(err)
	}

	if activeCount != 0 {
		t.Fatalf(
			"archived POP should not appear in active scope, got %d",
			activeCount,
		)
	}

	var archived models.POP
	if err := db.
		Unscoped().
		First(&archived, pop.ID).Error; err != nil {
		t.Fatal(err)
	}

	if !archived.DeletedAt.Valid {
		t.Fatal("POP should be soft-deleted")
	}

	if archived.Status != "INACTIVE" {
		t.Fatalf(
			"archived POP should be INACTIVE, got %s",
			archived.Status,
		)
	}

	rows, err := ListArchivedPOPs()
	if err != nil {
		t.Fatal(err)
	}

	if len(rows) != 1 || rows[0].ID != pop.ID {
		t.Fatalf(
			"expected archived POP %d, got %+v",
			pop.ID,
			rows,
		)
	}

	restored, err := RestorePOP(pop.ID)
	if err != nil {
		t.Fatal(err)
	}

	if restored.Status != "INACTIVE" {
		t.Fatalf(
			"restored POP should remain INACTIVE, got %s",
			restored.Status,
		)
	}

	if restored.DeletedAt.Valid {
		t.Fatal("restored POP should not be deleted")
	}

	var restoredPOP models.POP
	if err := db.First(
		&restoredPOP,
		pop.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if restoredPOP.Status != "INACTIVE" {
		t.Fatalf(
			"restored POP persisted with wrong status %s",
			restoredPOP.Status,
		)
	}
}
