package services

import (
	"fmt"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type SubscriptionLifecycleAction string

const (
	SubscriptionLifecycleSuspend     SubscriptionLifecycleAction = "SUSPEND"
	SubscriptionLifecycleActivate    SubscriptionLifecycleAction = "ACTIVATE"
	SubscriptionLifecycleRenew       SubscriptionLifecycleAction = "RENEW"
	SubscriptionLifecycleDisconnect  SubscriptionLifecycleAction = "DISCONNECT"
	SubscriptionLifecycleExpire      SubscriptionLifecycleAction = "EXPIRE"
	SubscriptionLifecyclePaymentVoid SubscriptionLifecycleAction = "PAYMENT_VOID"
)

type SubscriptionPPPReconciliationRunner func(
	subscriptionID uint,
	keyMaterial string,
) (PPPSecretReconciliationResult, error)

type SubscriptionLifecycleReconciliationResult struct {
	Action SubscriptionLifecycleAction

	SubscriptionID uint
	Status         string

	ReconciliationAttempted bool
	Reconciliation          PPPSecretReconciliationResult
	ReconciliationError     string
}

// ReconcileSubscriptionLifecyclePostCommit runs PPP reconciliation only
// after the caller has successfully persisted the subscription lifecycle
// mutation.
//
// Reconciliation failure is deliberately non-transactional: the persisted
// subscription lifecycle state remains authoritative and is not rolled back.
func ReconcileSubscriptionLifecyclePostCommit(
	subscription *models.Subscription,
	action SubscriptionLifecycleAction,
	keyMaterial string,
	runner SubscriptionPPPReconciliationRunner,
) (SubscriptionLifecycleReconciliationResult, error) {
	if subscription == nil {
		return SubscriptionLifecycleReconciliationResult{},
			fmt.Errorf("subscription is required")
	}

	if subscription.ID == 0 {
		return SubscriptionLifecycleReconciliationResult{},
			fmt.Errorf("subscription id is required")
	}

	result := SubscriptionLifecycleReconciliationResult{
		Action:         action,
		SubscriptionID: subscription.ID,
		Status:         subscription.Status,
	}

	if runner == nil {
		return result,
			fmt.Errorf("PPP reconciliation runner is required")
	}

	result.ReconciliationAttempted = true

	reconciliation, err := runner(
		subscription.ID,
		keyMaterial,
	)

	result.Reconciliation = reconciliation

	if err != nil {
		result.ReconciliationError = err.Error()

		// Important lifecycle contract:
		// DB state was already committed by the caller.
		// Router reconciliation failure must not roll it back.
		return result, nil
	}

	return result, nil
}

func ReconcileSubscriptionLifecycleWithMikroTikPostCommit(
	subscription *models.Subscription,
	action SubscriptionLifecycleAction,
	keyMaterial string,
) (SubscriptionLifecycleReconciliationResult, error) {
	return ReconcileSubscriptionLifecyclePostCommit(
		subscription,
		action,
		keyMaterial,
		ReconcileSubscriptionPPPSecretWithMikroTik,
	)
}
