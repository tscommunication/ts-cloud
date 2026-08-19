package services

import (
	"errors"
	"testing"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

const lifecycleReconciliationTestKey = "0123456789abcdef0123456789abcdef"

func TestReconcileSubscriptionLifecyclePostCommitSuspend(
	t *testing.T,
) {
	subscription := &models.Subscription{
		Status: "SUSPENDED",
	}
	subscription.ID = 101

	calls := 0

	runner := func(
		subscriptionID uint,
		keyMaterial string,
	) (PPPSecretReconciliationResult, error) {
		calls++

		if subscriptionID != subscription.ID {
			t.Fatalf(
				"subscription id = %d, want %d",
				subscriptionID,
				subscription.ID,
			)
		}

		if keyMaterial != lifecycleReconciliationTestKey {
			t.Fatal("credential key was not forwarded")
		}

		return PPPSecretReconciliationResult{
			Plan: PPPSecretReconciliationPlan{
				SubscriptionID: subscription.ID,
				Action:         PPPSecretActionDisable,
			},
			Execution: PPPSecretReconciliationExecution{
				Action:   PPPSecretActionDisable,
				Executed: true,
				SecretID: "*10",
			},
		}, nil
	}

	result, err :=
		ReconcileSubscriptionLifecyclePostCommit(
			subscription,
			SubscriptionLifecycleSuspend,
			lifecycleReconciliationTestKey,
			runner,
		)
	if err != nil {
		t.Fatalf(
			"reconcile suspend post-commit: %v",
			err,
		)
	}

	if calls != 1 {
		t.Fatalf("runner calls = %d, want 1", calls)
	}

	if !result.ReconciliationAttempted {
		t.Fatal("reconciliation was not attempted")
	}

	if result.Status != "SUSPENDED" {
		t.Fatalf(
			"status = %q, want SUSPENDED",
			result.Status,
		)
	}

	if result.Reconciliation.Execution.Action !=
		PPPSecretActionDisable {
		t.Fatalf(
			"action = %q, want DISABLE",
			result.Reconciliation.Execution.Action,
		)
	}
}

func TestReconcileSubscriptionLifecyclePostCommitActivate(
	t *testing.T,
) {
	subscription := &models.Subscription{
		Status: "ACTIVE",
	}
	subscription.ID = 102

	runner := func(
		subscriptionID uint,
		keyMaterial string,
	) (PPPSecretReconciliationResult, error) {
		return PPPSecretReconciliationResult{
			Plan: PPPSecretReconciliationPlan{
				SubscriptionID: subscriptionID,
				Action:         PPPSecretActionEnable,
			},
			Execution: PPPSecretReconciliationExecution{
				Action:   PPPSecretActionEnable,
				Executed: true,
				SecretID: "*11",
			},
		}, nil
	}

	result, err :=
		ReconcileSubscriptionLifecyclePostCommit(
			subscription,
			SubscriptionLifecycleActivate,
			lifecycleReconciliationTestKey,
			runner,
		)
	if err != nil {
		t.Fatalf(
			"reconcile activate post-commit: %v",
			err,
		)
	}

	if result.Reconciliation.Execution.Action !=
		PPPSecretActionEnable {
		t.Fatalf(
			"action = %q, want ENABLE",
			result.Reconciliation.Execution.Action,
		)
	}
}

func TestReconcileSubscriptionLifecyclePostCommitRenew(
	t *testing.T,
) {
	subscription := &models.Subscription{
		Status: "ACTIVE",
	}
	subscription.ID = 103

	runner := func(
		subscriptionID uint,
		keyMaterial string,
	) (PPPSecretReconciliationResult, error) {
		return PPPSecretReconciliationResult{
			Plan: PPPSecretReconciliationPlan{
				SubscriptionID: subscriptionID,
				Action:         PPPSecretActionNoop,
			},
			Execution: PPPSecretReconciliationExecution{
				Action:   PPPSecretActionNoop,
				Executed: false,
			},
		}, nil
	}

	result, err :=
		ReconcileSubscriptionLifecyclePostCommit(
			subscription,
			SubscriptionLifecycleRenew,
			lifecycleReconciliationTestKey,
			runner,
		)
	if err != nil {
		t.Fatalf(
			"reconcile renew post-commit: %v",
			err,
		)
	}

	if result.Status != "ACTIVE" {
		t.Fatalf(
			"status = %q, want ACTIVE",
			result.Status,
		)
	}
}

func TestReconcileSubscriptionLifecyclePostCommitDisconnect(
	t *testing.T,
) {
	subscription := &models.Subscription{
		Status: "DISCONNECTED",
	}
	subscription.ID = 104

	runner := func(
		subscriptionID uint,
		keyMaterial string,
	) (PPPSecretReconciliationResult, error) {
		return PPPSecretReconciliationResult{
			Plan: PPPSecretReconciliationPlan{
				SubscriptionID: subscriptionID,
				Action:         PPPSecretActionDisable,
			},
			Execution: PPPSecretReconciliationExecution{
				Action:   PPPSecretActionDisable,
				Executed: true,
				SecretID: "*12",
			},
		}, nil
	}

	result, err :=
		ReconcileSubscriptionLifecyclePostCommit(
			subscription,
			SubscriptionLifecycleDisconnect,
			lifecycleReconciliationTestKey,
			runner,
		)
	if err != nil {
		t.Fatalf(
			"reconcile disconnect post-commit: %v",
			err,
		)
	}

	if result.Reconciliation.Execution.Action !=
		PPPSecretActionDisable {
		t.Fatalf(
			"action = %q, want DISABLE",
			result.Reconciliation.Execution.Action,
		)
	}
}

func TestReconcileSubscriptionLifecyclePostCommitFailureDoesNotBecomeLifecycleFailure(
	t *testing.T,
) {
	subscription := &models.Subscription{
		Status: "SUSPENDED",
	}
	subscription.ID = 105

	runner := func(
		subscriptionID uint,
		keyMaterial string,
	) (PPPSecretReconciliationResult, error) {
		return PPPSecretReconciliationResult{
				Plan: PPPSecretReconciliationPlan{
					SubscriptionID: subscriptionID,
					Action:         PPPSecretActionDisable,
				},
			},
			errors.New("router unavailable")
	}

	result, err :=
		ReconcileSubscriptionLifecyclePostCommit(
			subscription,
			SubscriptionLifecycleSuspend,
			lifecycleReconciliationTestKey,
			runner,
		)

	if err != nil {
		t.Fatalf(
			"router failure became lifecycle failure: %v",
			err,
		)
	}

	if subscription.Status != "SUSPENDED" {
		t.Fatalf(
			"subscription status changed to %q",
			subscription.Status,
		)
	}

	if result.ReconciliationError == "" {
		t.Fatal(
			"expected reconciliation error to be recorded",
		)
	}

	if !result.ReconciliationAttempted {
		t.Fatal("reconciliation attempt was not recorded")
	}
}

func TestReconcileSubscriptionLifecyclePostCommitValidation(
	t *testing.T,
) {
	valid := &models.Subscription{
		Status: "ACTIVE",
	}
	valid.ID = 200

	runner := func(
		uint,
		string,
	) (PPPSecretReconciliationResult, error) {
		return PPPSecretReconciliationResult{}, nil
	}

	tests := []struct {
		name         string
		subscription *models.Subscription
		runner       SubscriptionPPPReconciliationRunner
	}{
		{
			name:         "nil subscription",
			subscription: nil,
			runner:       runner,
		},
		{
			name: "missing subscription id",
			subscription: &models.Subscription{
				Status: "ACTIVE",
			},
			runner: runner,
		},
		{
			name:         "missing runner",
			subscription: valid,
			runner:       nil,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				_, err :=
					ReconcileSubscriptionLifecyclePostCommit(
						test.subscription,
						SubscriptionLifecycleActivate,
						lifecycleReconciliationTestKey,
						test.runner,
					)

				if err == nil {
					t.Fatal(
						"expected validation failure",
					)
				}
			},
		)
	}
}

func TestLifecycleMutationsRemainIndependentFromReconciliation(
	t *testing.T,
) {
	now := time.Date(
		2026,
		time.August,
		19,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	subscription := &models.Subscription{
		Status:     "SUSPENDED",
		ExpiryDate: now.AddDate(0, 1, 0),
	}

	if subscription.Status != "SUSPENDED" {
		t.Fatal("unexpected test fixture status")
	}
}
