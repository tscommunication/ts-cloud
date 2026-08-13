package services

import (
	"log"
	"time"

	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func StartSubscriptionExpiryWorker() {
	go func() {
		processSubscriptionExpiries()

		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			processSubscriptionExpiries()
		}
	}()
}

func processSubscriptionExpiries() {
	count, err := repositories.ExpireOverdueSubscriptions(time.Now())
	if err != nil {
		log.Printf("Subscription expiry worker: %v", err)
		return
	}
	if count > 0 {
		log.Printf("Subscription expiry worker: marked %d subscription(s) expired", count)
	}
}
