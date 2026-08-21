package services

import (
	"log"
	"time"
)

func StartTemporaryInternetAccessWorker(keyMaterial string) {
	go func() {
		processDueTemporaryInternetAccess(time.Now(), keyMaterial)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for now := range ticker.C {
			processDueTemporaryInternetAccess(now, keyMaterial)
		}
	}()
}

func processDueTemporaryInternetAccess(now time.Time, keyMaterial string) {
	due, err := ExpireDueTemporaryInternetAccess(now)
	if err != nil {
		log.Printf("Temporary access worker: %v", err)
		return
	}
	for _, item := range due {
		if _, err := ReconcileSubscriptionPPPSecretWithMikroTik(item.SubscriptionID, keyMaterial); err != nil {
			log.Printf("Temporary access worker: subscription=%d reconciliation failed: %v", item.SubscriptionID, err)
		}
	}
}
