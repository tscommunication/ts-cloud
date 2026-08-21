package services

import (
	"log"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

type subscriptionExpiryReconciliationRunner func(
	subscription *models.Subscription,
	action SubscriptionLifecycleAction,
	keyMaterial string,
) (SubscriptionLifecycleReconciliationResult, error)

func StartSubscriptionExpiryWorker(
	keyMaterial string,
) {
	go func() {
		processSubscriptionExpiries(
			time.Now(),
			keyMaterial,
			ReconcileSubscriptionLifecycleWithMikroTikPostCommit,
		)

		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for now := range ticker.C {
			processSubscriptionExpiries(
				now,
				keyMaterial,
				ReconcileSubscriptionLifecycleWithMikroTikPostCommit,
			)
		}
	}()
}

func processSubscriptionExpiries(
	now time.Time,
	keyMaterial string,
	runner subscriptionExpiryReconciliationRunner,
) {
	expiredSubscriptions, err :=
		repositories.ExpireOverdueSubscriptions(
			now,
		)
	if err != nil {
		log.Printf(
			"Subscription expiry worker: %v",
			err,
		)
		return
	}

	if len(expiredSubscriptions) == 0 {
		return
	}

	log.Printf(
		"Subscription expiry worker: marked %d subscription(s) expired",
		len(expiredSubscriptions),
	)

	if runner == nil {
		log.Printf(
			"Subscription expiry worker: PPP reconciliation runner is not configured",
		)
		return
	}

	for index := range expiredSubscriptions {
		subscription := &expiredSubscriptions[index]

		result, reconcileErr := runner(
			subscription,
			SubscriptionLifecycleExpire,
			keyMaterial,
		)

		if reconcileErr != nil {
			log.Printf(
				"Subscription expiry worker: subscription=%d PPP reconciliation setup failed: %v",
				subscription.ID,
				reconcileErr,
			)
			continue
		}

		if result.ReconciliationError != "" {
			log.Printf(
				"Subscription expiry worker: subscription=%d PPP reconciliation failed: %s",
				subscription.ID,
				result.ReconciliationError,
			)
		}
	}
}
