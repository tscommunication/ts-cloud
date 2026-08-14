package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type schemaMigration struct {
	Version   uint      `gorm:"primaryKey"`
	Name      string    `gorm:"size:200;not null"`
	AppliedAt time.Time `gorm:"not null"`
}

type migration struct {
	version uint
	name    string
	up      func(*gorm.DB) error
}

var migrations = []migration{
	{version: 1, name: "initial_application_schema", up: migrateInitialSchema},
	{version: 2, name: "nullable_unknown_ftp_login_user", up: migrateNullableFTPLoginUser},
	{version: 3, name: "pop_agent_distribution_hierarchy", up: migratePOPAgentHierarchy},
	{version: 4, name: "agent_collection_ledger", up: migrateAgentCollectionLedger},
	{version: 5, name: "agent_commission_settlements", up: migrateAgentCommissionSettlements},
	{version: 6, name: "agent_scoped_user_accounts", up: migrateAgentScopedUsers},
	{version: 7, name: "payment_collector_audit", up: migratePaymentCollectorAudit},
	{version: 8, name: "network_router_inventory", up: migrateNetworkRouterInventory},
	{version: 9, name: "network_router_health", up: migrateNetworkRouterHealth},
	{version: 10, name: "network_router_credentials", up: migrateNetworkRouterCredentials},
	{version: 11, name: "network_router_resource_sync", up: migrateNetworkRouterResourceSync},
	{version: 12, name: "network_router_separate_errors", up: migrateNetworkRouterSeparateErrors},
	{version: 13, name: "network_router_health_history", up: migrateNetworkRouterHealthHistory},
	{version: 14, name: "network_router_resource_alerts", up: migrateNetworkRouterResourceAlerts},
	{version: 15, name: "network_router_pppoe_sessions", up: migrateNetworkRouterPPPoESessions},
	{version: 16, name: "correct_pppoe_session_table_name", up: migrateCorrectPPPoESessionTableName},
	{version: 17, name: "customer_csv_import_audit", up: migrateCustomerCSVImportAudit},
	{version: 18, name: "package_commission_catalog", up: migratePackageCommissionCatalog},
	{version: 19, name: "agent_pop_import_catalog", up: migrateAgentPOPImportCatalog},
}

func migrateNullableFTPLoginUser(db *gorm.DB) error {
	return db.AutoMigrate(&models.FTPLoginLog{})
}

func migratePOPAgentHierarchy(db *gorm.DB) error {
	return db.AutoMigrate(&models.POP{}, &models.Agent{}, &models.Customer{})
}

func migrateAgentCollectionLedger(db *gorm.DB) error {
	return db.AutoMigrate(&models.AgentCollection{})
}

func migrateAgentCommissionSettlements(db *gorm.DB) error {
	return db.AutoMigrate(&models.AgentSettlement{})
}

func migrateAgentScopedUsers(db *gorm.DB) error { return db.AutoMigrate(&models.User{}) }

func migratePaymentCollectorAudit(db *gorm.DB) error { return db.AutoMigrate(&models.Payment{}) }

func migrateNetworkRouterInventory(db *gorm.DB) error { return db.AutoMigrate(&models.NetworkRouter{}) }

func migrateNetworkRouterHealth(db *gorm.DB) error { return db.AutoMigrate(&models.NetworkRouter{}) }

func migrateNetworkRouterCredentials(db *gorm.DB) error {
	return db.AutoMigrate(&models.NetworkRouter{})
}

func migrateNetworkRouterResourceSync(db *gorm.DB) error {
	return db.AutoMigrate(&models.NetworkRouter{})
}

func migrateNetworkRouterSeparateErrors(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.NetworkRouter{}); err != nil {
		return err
	}
	if err := db.Exec("UPDATE network_routers SET last_api_error = last_connection_error WHERE api_status = ? AND last_connection_error <> ''", "AUTH_FAILED").Error; err != nil {
		return err
	}
	return db.Exec("UPDATE network_routers SET last_tcp_error = last_connection_error WHERE connectivity_status = ? AND last_connection_error <> ''", "OFFLINE").Error
}

func migrateNetworkRouterHealthHistory(db *gorm.DB) error {
	return db.AutoMigrate(&models.NetworkRouterHealth{})
}

func migrateNetworkRouterResourceAlerts(db *gorm.DB) error {
	return db.AutoMigrate(&models.NetworkRouterAlert{})
}

func migrateNetworkRouterPPPoESessions(db *gorm.DB) error {
	return db.AutoMigrate(&models.NetworkRouterPPPoESession{})
}

func migrateCorrectPPPoESessionTableName(db *gorm.DB) error {
	const legacyTable = "network_router_pp_po_e_sessions"
	const expectedTable = "network_router_pppoe_sessions"
	if db.Migrator().HasTable(legacyTable) && !db.Migrator().HasTable(expectedTable) {
		if err := db.Migrator().RenameTable(legacyTable, expectedTable); err != nil {
			return err
		}
	}
	return db.AutoMigrate(&models.NetworkRouterPPPoESession{})
}

func migrateCustomerCSVImportAudit(db *gorm.DB) error {
	return db.AutoMigrate(&models.CustomerImportBatch{}, &models.CustomerImportItem{})
}
func migratePackageCommissionCatalog(db *gorm.DB) error { return db.AutoMigrate(&models.Package{}) }
func migrateAgentPOPImportCatalog(db *gorm.DB) error {
	return db.AutoMigrate(&models.Agent{}, &models.CustomerImportBatch{})
}

func runMigrations(db *gorm.DB) error {
	return Migrate(db)
}

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}
	for _, item := range migrations {
		var count int64
		if err := db.Model(&schemaMigration{}).Where("version = ?", item.version).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := item.up(tx); err != nil {
				return err
			}
			return tx.Create(&schemaMigration{Version: item.version, Name: item.name, AppliedAt: time.Now().UTC()}).Error
		}); err != nil {
			return fmt.Errorf("migration %03d %s: %w", item.version, item.name, err)
		}
	}
	return nil
}

func migrateInitialSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{}, &models.POP{}, &models.Agent{}, &models.Customer{}, &models.Package{}, &models.Subscription{},
		&models.Invoice{}, &models.Payment{}, &models.BillingRun{}, &models.BillingRunItem{},
		&models.FTPServer{}, &models.FTPUser{}, &models.FTPLoginLog{},
		&models.FTPTransferLog{}, &models.SystemLogOffset{}, &models.AgentCollection{}, &models.AgentSettlement{},
	)
}
