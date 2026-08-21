package services

import (
	"log"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

var customerPPPoESyncTick = time.Minute

func StartCustomerPPPoESyncWorker(keyMaterial string) {
	go func() {
		runCustomerPPPoESync(keyMaterial, time.Now())
		ticker := time.NewTicker(customerPPPoESyncTick)
		defer ticker.Stop()
		for now := range ticker.C {
			runCustomerPPPoESync(keyMaterial, now)
		}
	}()
}

func runCustomerPPPoESync(keyMaterial string, now time.Time) {
	var accounts []models.CustomerInternetAccount
	if err := database.DB.Where("status IN ?", []string{"ACTIVE", TemporaryInternetStatusActive}).Find(&accounts).Error; err != nil {
		log.Printf("Customer PPPoE sync worker: load accounts: %v", err)
		return
	}
	for _, account := range accounts {
		interval := 30
		if account.LastSyncedAt != nil && account.LastSyncedAt.Add(time.Duration(interval)*time.Minute).After(now) {
			continue
		}

		var subscription models.Subscription
		err := database.DB.Where("customer_id = ? AND internet_account_id = ? AND status = ?", account.CustomerID, account.ID, "ACTIVE").Order("id").First(&subscription).Error
		if err != nil {
			continue
		}

		_, syncErr := ReconcileSubscriptionPPPSecretWithMikroTik(subscription.ID, keyMaterial)
		updates := map[string]interface{}{"last_synced_at": now, "last_sync_error": ""}
		if syncErr != nil {
			updates["last_sync_error"] = strings.TrimSpace(syncErr.Error())
			log.Printf("Customer PPPoE sync worker: customer=%d subscription=%d: %v", account.CustomerID, subscription.ID, syncErr)
		}
		if err := database.DB.Model(&models.CustomerInternetAccount{}).Where("id = ?", account.ID).Updates(updates).Error; err != nil {
			log.Printf("Customer PPPoE sync worker: update account=%d: %v", account.ID, err)
		}
	}
}
