package database

import (
	"errors"
	"fmt"
	"strings"
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
	{version: 20, name: "agent_multiple_pop_locations", up: migrateAgentMultiplePOPLocations},
	{version: 21, name: "distribution_archive_lifecycle", up: migrateDistributionArchiveLifecycle},
	{version: 22, name: "customer_structured_address", up: migrateCustomerStructuredAddress},
	{version: 23, name: "bangladesh_location_master", up: migrateBangladeshLocationMaster},
	{version: 24, name: "customer_identity_uniqueness", up: migrateCustomerIdentityUniqueness},
	{version: 25, name: "customer_provision_requests", up: migrateCustomerProvisionRequests},
	{version: 26, name: "customer_extended_domain", up: migrateCustomerExtendedDomain},
	{version: 27, name: "pppoe_password_encryption_columns", up: migratePPPoEPasswordEncryptionColumns},
	{version: 28, name: "subscription_date_adjustment_audit", up: migrateSubscriptionDateAdjustmentAudit},
	{version: 29, name: "payment_void_audit", up: migratePaymentVoidAudit},
	{version: 30, name: "subscription_renewal_ledger", up: migrateSubscriptionRenewalLedger},
	{version: 31, name: "subscription_renewal_reversal", up: migrateSubscriptionRenewalReversal},
	{version: 32, name: "customer_portal_user_identity", up: migrateCustomerPortalUserIdentity},
	{version: 33, name: "unified_customer_identity", up: migrateUnifiedCustomerIdentity},
	{version: 34, name: "customer_internet_accounts", up: migrateCustomerInternetAccounts},
	{version: 35, name: "customer_pppoe_sync_schedule", up: migrateCustomerPPPoESyncSchedule},
	{version: 36, name: "customer_pppoe_sync_every_30_minutes", up: migrateCustomerPPPoESyncEvery30Minutes},
	{version: 37, name: "agent_package_assignments", up: migrateAgentPackageAssignments},
	{version: 38, name: "code_change_audit", up: migrateCodeChangeAudit},
	{version: 39, name: "customer_internet_lifecycle", up: migrateCustomerInternetLifecycle},
	{version: 40, name: "temporary_internet_access_ledger", up: migrateTemporaryInternetAccessLedger},
	{version: 41, name: "agent_router_permissions", up: migrateAgentRouterPermissions},
	{version: 42, name: "customer_ftp_entitlements", up: migrateCustomerFTPEntitlements},
	{version: 43, name: "network_router_ppp_secrets", up: migrateNetworkRouterPPPSecrets},
	{version: 44, name: "unified_service_entitlements", up: migrateUnifiedServiceEntitlements},
	{version: 45, name: "in_app_notifications", up: migrateInAppNotifications},
	{version: 46, name: "network_device_inventory", up: migrateNetworkDeviceInventory},
	{version: 47, name: "olt_device_configuration", up: migrateOLTDeviceConfiguration},
	{version: 48, name: "customer_geo_coordinates", up: migrateCustomerGeoCoordinates},
	{version: 49, name: "network_device_telemetry", up: migrateNetworkDeviceTelemetry},
	{version: 50, name: "network_device_onu_sample_fk_column", up: migrateNetworkDeviceONUSampleFKColumn},
	{version: 51, name: "network_device_sample_uniqueness", up: migrateNetworkDeviceSampleUniqueness},
}

func migrateNetworkDeviceSampleUniqueness(
	db *gorm.DB,
) error {
	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_network_device_port_sample_unique
		 ON network_device_port_samples (network_device_port_id, sampled_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_network_device_onu_sample_unique
		 ON network_device_onu_samples (network_device_onu_id, sampled_at)`,
	}

	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf(
				"create telemetry sample unique index: %w",
				err,
			)
		}
	}

	return nil
}

func migrateNetworkDeviceONUSampleFKColumn(db *gorm.DB) error {
	if db.Migrator().HasColumn(&models.NetworkDeviceONUSample{}, "network_device_on_uid") &&
		!db.Migrator().HasColumn(&models.NetworkDeviceONUSample{}, "network_device_onu_id") {
		return db.Migrator().RenameColumn(
			&models.NetworkDeviceONUSample{},
			"network_device_on_uid",
			"network_device_onu_id",
		)
	}
	return nil
}

func migrateNetworkDeviceTelemetry(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.NetworkDevicePort{},
		&models.NetworkDevicePortSample{},
		&models.NetworkDeviceONU{},
		&models.NetworkDeviceONUSample{},
	)
}

func migrateCustomerGeoCoordinates(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Customer{},
		&models.CustomerProvisionRequest{},
	)
}

func migrateOLTDeviceConfiguration(db *gorm.DB) error { return db.AutoMigrate(&models.NetworkDevice{}) }

func migrateNetworkDeviceInventory(db *gorm.DB) error { return db.AutoMigrate(&models.NetworkDevice{}) }

func migrateInAppNotifications(db *gorm.DB) error {
	return db.AutoMigrate(&models.Notification{}, &models.NotificationRead{})
}

func migrateUnifiedServiceEntitlements(db *gorm.DB) error {
	return db.AutoMigrate(&models.ServiceEntitlement{})
}

func migrateNetworkRouterPPPSecrets(db *gorm.DB) error {
	return db.AutoMigrate(&models.NetworkRouterPPPSecret{})
}

func migrateCustomerFTPEntitlements(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.FTPUser{}); err != nil {
		return err
	}
	if err := db.Exec(`UPDATE ftp_users SET customer_id = (
		SELECT subscriptions.customer_id FROM subscriptions
		WHERE subscriptions.id = ftp_users.subscription_id
	) WHERE EXISTS (
		SELECT 1 FROM subscriptions WHERE subscriptions.id = ftp_users.subscription_id
	)`).Error; err != nil {
		return err
	}
	var orphaned int64
	if err := db.Model(&models.FTPUser{}).
		Where("customer_id IS NULL OR customer_id = 0").
		Count(&orphaned).Error; err != nil {
		return err
	}
	if orphaned > 0 {
		return fmt.Errorf("cannot map %d FTP user(s) to a customer", orphaned)
	}
	return nil
}

func migrateAgentRouterPermissions(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.AgentRouter{}); err != nil {
		return err
	}
	return db.Exec(`INSERT INTO agent_routers (agent_id, router_id)
		SELECT DISTINCT ap.agent_id, nr.id FROM agent_pops ap
		JOIN network_routers nr ON nr.pop_id = ap.pop_id AND nr.deleted_at IS NULL
		WHERE NOT EXISTS (SELECT 1 FROM agent_routers ar WHERE ar.agent_id = ap.agent_id AND ar.router_id = nr.id)`).Error
}

func migrateTemporaryInternetAccessLedger(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.TemporaryInternetAccess{},
		&models.SubscriptionRenewal{},
	)
}

func migrateCustomerInternetLifecycle(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.CustomerInternetAccount{}); err != nil {
		return err
	}

	var accounts []models.CustomerInternetAccount
	if err := db.Order("id ASC").Find(&accounts).Error; err != nil {
		return err
	}

	for _, account := range accounts {
		var subscription models.Subscription
		query := db.Where("internet_account_id = ?", account.ID).Order("id ASC")
		if account.LegacySubscriptionID != nil && *account.LegacySubscriptionID != 0 {
			query = db.Where("id = ?", *account.LegacySubscriptionID)
		}
		if err := query.First(&subscription).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}

		updates := map[string]interface{}{
			"package_id":        subscription.PackageID,
			"activation_date":   subscription.ActivationDate,
			"billing_day":       subscription.BillingDay,
			"next_billing_date": subscription.NextBillingDate,
			"expiry_date":       subscription.ExpiryDate,
			"status":            subscription.Status,
		}
		if err := db.Model(&models.CustomerInternetAccount{}).
			Where("id = ?", account.ID).
			Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

func migrateCodeChangeAudit(db *gorm.DB) error {
	return db.AutoMigrate(&models.CodeChangeAudit{})
}

func migrateAgentPackageAssignments(db *gorm.DB) error {
	return db.AutoMigrate(&models.AgentPackage{})
}

func migrateCustomerPPPoESyncEvery30Minutes(db *gorm.DB) error {
	return db.Model(&models.CustomerInternetAccount{}).Where("sync_interval_minutes <> ? OR sync_interval_minutes IS NULL", 30).Update("sync_interval_minutes", 30).Error
}

func migrateCustomerPPPoESyncSchedule(db *gorm.DB) error {
	return db.AutoMigrate(&models.CustomerInternetAccount{})
}

func migrateCustomerProvisionRequests(db *gorm.DB) error {
	return db.AutoMigrate(&models.CustomerProvisionRequest{})
}

func migrateCustomerExtendedDomain(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Customer{},
		&models.CustomerTechnicalProfile{},
		&models.CustomerReference{},
	)
}

func migratePPPoEPasswordEncryptionColumns(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Subscription{},
		&models.CustomerProvisionRequest{},
	)
}

func migrateSubscriptionDateAdjustmentAudit(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.SubscriptionDateAdjustment{},
	)
}

func migratePaymentVoidAudit(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.PaymentVoidAudit{},
	)
}

func migrateSubscriptionRenewalLedger(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.SubscriptionRenewal{},
	)
}

func migrateSubscriptionRenewalReversal(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.SubscriptionRenewalReversal{},
	)
}

func migrateCustomerPortalUserIdentity(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
	)
}

func migrateUnifiedCustomerIdentity(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.Customer{}, &models.User{}); err != nil {
		return err
	}

	var customers []models.Customer
	if err := db.Unscoped().Where("deleted_at IS NULL").Find(&customers).Error; err != nil {
		return err
	}

	for _, customer := range customers {
		cid := strings.TrimSpace(customer.CustomerCode)
		if cid == "" {
			return fmt.Errorf("customer %d has no permanent CID", customer.ID)
		}

		var linked models.User
		linkedErr := db.Where("customer_id = ?", customer.ID).First(&linked).Error
		if linkedErr == nil {
			if linked.Role != "customer" {
				return fmt.Errorf(
					"customer %s is linked to non-customer user %d",
					cid,
					linked.ID,
				)
			}
			continue
		}
		if !errors.Is(linkedErr, gorm.ErrRecordNotFound) {
			return linkedErr
		}

		var usernameCount int64
		if err := db.Model(&models.User{}).
			Where("LOWER(username) = LOWER(?)", cid).
			Count(&usernameCount).Error; err != nil {
			return err
		}
		if usernameCount > 0 {
			return fmt.Errorf("permanent CID %s conflicts with an existing username", cid)
		}

		customerID := customer.ID
		identity := models.User{
			Name:       customer.FullName,
			Username:   cid,
			Email:      strings.ToLower(cid) + "@customer.invalid",
			Password:   "!C19-D9-A0-PASSWORD-NOT-PROVISIONED!",
			Role:       "customer",
			Active:     false,
			CustomerID: &customerID,
		}
		if err := db.Create(&identity).Error; err != nil {
			return err
		}
		if err := db.Model(&identity).Update("active", false).Error; err != nil {
			return err
		}
	}

	return nil
}

func migrateCustomerInternetAccounts(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.CustomerInternetAccount{},
		&models.Subscription{},
	); err != nil {
		return err
	}

	var subscriptions []models.Subscription
	if err := db.Where("TRIM(pp_po_e_username) <> ''").
		Order("id ASC").
		Find(&subscriptions).Error; err != nil {
		return err
	}

	for _, subscription := range subscriptions {
		if subscription.InternetAccountID != nil {
			continue
		}

		username := strings.TrimSpace(subscription.PPPoEUsername)
		if strings.TrimSpace(subscription.PPPoEPasswordEncrypted) == "" {
			return fmt.Errorf(
				"subscription %s has no encrypted PPPoE credential",
				subscription.SubscriptionCode,
			)
		}

		var account models.CustomerInternetAccount
		err := db.Where("LOWER(pp_po_e_username) = LOWER(?)", username).
			First(&account).Error
		if err == nil {
			if account.CustomerID != subscription.CustomerID {
				return fmt.Errorf(
					"PPPoE username %s belongs to multiple customers",
					username,
				)
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			legacySubscriptionID := subscription.ID
			account = models.CustomerInternetAccount{
				AccountCode:            fmt.Sprintf("NET-%06d", subscription.ID),
				CustomerID:             subscription.CustomerID,
				RouterID:               subscription.RouterID,
				PPPoEUsername:          username,
				PPPoEPasswordEncrypted: subscription.PPPoEPasswordEncrypted,
				Status:                 subscription.Status,
				LegacySubscriptionID:   &legacySubscriptionID,
			}
			if err := db.Create(&account).Error; err != nil {
				return err
			}
		} else {
			return err
		}

		if err := db.Model(&models.Subscription{}).
			Where("id = ?", subscription.ID).
			Update("internet_account_id", account.ID).Error; err != nil {
			return err
		}
	}

	return nil
}

func migrateCustomerIdentityUniqueness(db *gorm.DB) error {
	type duplicateIdentity struct {
		Value string
		Count int64
	}

	var duplicateMobile duplicateIdentity
	if err := db.Table("customers").
		Select("TRIM(mobile) AS value, COUNT(*) AS count").
		Where("TRIM(mobile) <> ''").
		Group("TRIM(mobile)").
		Having("COUNT(*) > 1").
		Limit(1).
		Scan(&duplicateMobile).Error; err != nil {
		return err
	}
	if duplicateMobile.Count > 1 {
		return fmt.Errorf(
			"cannot enforce customer mobile uniqueness: duplicate mobile %q exists",
			duplicateMobile.Value,
		)
	}

	var duplicateNID duplicateIdentity
	if err := db.Table("customers").
		Select("TRIM(n_id) AS value, COUNT(*) AS count").
		Where("TRIM(COALESCE(n_id, '')) <> ''").
		Group("TRIM(n_id)").
		Having("COUNT(*) > 1").
		Limit(1).
		Scan(&duplicateNID).Error; err != nil {
		return err
	}
	if duplicateNID.Count > 1 {
		return fmt.Errorf(
			"cannot enforce customer NID uniqueness: duplicate NID %q exists",
			duplicateNID.Value,
		)
	}

	if err := db.Exec(`
		UPDATE customers
		SET mobile = TRIM(mobile)
		WHERE mobile <> TRIM(mobile)
	`).Error; err != nil {
		return err
	}

	if err := db.Exec(`
		UPDATE customers
		SET n_id = TRIM(n_id)
		WHERE n_id IS NOT NULL
		  AND n_id <> TRIM(n_id)
	`).Error; err != nil {
		return err
	}

	if !db.Migrator().HasIndex(&models.Customer{}, "idx_customers_mobile_unique") {
		if err := db.Migrator().CreateIndex(
			&models.Customer{},
			"idx_customers_mobile_unique",
		); err != nil {
			return err
		}
	}

	if !db.Migrator().HasIndex(&models.Customer{}, "idx_customers_nid_unique") {
		if err := db.Exec(`
			CREATE UNIQUE INDEX idx_customers_nid_unique
			ON customers (n_id)
			WHERE n_id IS NOT NULL
			  AND TRIM(n_id) <> ''
		`).Error; err != nil {
			return err
		}
	}

	return nil
}

func migrateBangladeshLocationMaster(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Division{},
		&models.District{},
		&models.Upazila{},
		&models.PostOffice{},
		&models.Customer{},
	)
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
func migrateDistributionArchiveLifecycle(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.POP{},
		&models.Agent{},
	)
}

func migrateCustomerStructuredAddress(db *gorm.DB) error {
	// Snapshot legacy address values before AutoMigrate. SQLite may rebuild the
	// customers table while changing column definitions, which can otherwise
	// lose legacy address data before it is copied into the structured fields.
	type legacyCustomerAddress struct {
		ID               uint
		Village          string
		Address          string
		RoadOrArea       string
		VillageOrHolding string
	}

	var legacyRows []legacyCustomerAddress
	if db.Migrator().HasTable("customers") {
		selectColumns := []string{"id", "village", "address"}
		hasRoadOrArea := db.Migrator().HasColumn("customers", "road_or_area")
		hasVillageOrHolding := db.Migrator().HasColumn("customers", "village_or_holding")

		if hasRoadOrArea {
			selectColumns = append(selectColumns, "road_or_area")
		}
		if hasVillageOrHolding {
			selectColumns = append(selectColumns, "village_or_holding")
		}

		if err := db.Table("customers").
			Select(selectColumns).
			Scan(&legacyRows).Error; err != nil {
			return err
		}
	}

	if err := db.AutoMigrate(&models.Customer{}); err != nil {
		return err
	}

	if err := db.Exec(`
		UPDATE customers
		SET country = ?
		WHERE country IS NULL OR TRIM(country) = ''
	`, "Bangladesh").Error; err != nil {
		return err
	}

	for _, row := range legacyRows {
		updates := map[string]interface{}{}

		if value := strings.TrimSpace(row.VillageOrHolding); value != "" {
			updates["village_or_holding"] = value
		} else if value := strings.TrimSpace(row.Village); value != "" {
			updates["village_or_holding"] = value
		}

		if value := strings.TrimSpace(row.RoadOrArea); value != "" {
			updates["road_or_area"] = value
		} else if value := strings.TrimSpace(row.Address); value != "" {
			updates["road_or_area"] = value
		}

		if len(updates) == 0 {
			continue
		}

		if road, ok := updates["road_or_area"]; ok {
			query := db.Table("customers").Where("id = ?", row.ID)
			if strings.TrimSpace(row.RoadOrArea) == "" {
				query = query.Where("road_or_area IS NULL OR TRIM(road_or_area) = ''")
			}
			if err := query.Update("road_or_area", road).Error; err != nil {
				return err
			}
		}

		if village, ok := updates["village_or_holding"]; ok {
			query := db.Table("customers").Where("id = ?", row.ID)
			if strings.TrimSpace(row.VillageOrHolding) == "" {
				query = query.Where("village_or_holding IS NULL OR TRIM(village_or_holding) = ''")
			}
			if err := query.Update("village_or_holding", village).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func migrateAgentMultiplePOPLocations(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.AgentPOP{}); err != nil {
		return err
	}
	var agents []models.Agent
	if err := db.Find(&agents).Error; err != nil {
		return err
	}
	for _, agent := range agents {
		if agent.POPID == 0 {
			continue
		}
		link := models.AgentPOP{AgentID: agent.ID, POPID: agent.POPID}
		if err := db.Where(link).FirstOrCreate(&link).Error; err != nil {
			return err
		}
	}
	return nil
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
