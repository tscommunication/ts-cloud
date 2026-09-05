package services

import (
	"errors"
	"testing"

	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestCustomerInternetCredentialPostSavePasswordChangeRunsPPPAndFTP(
	t *testing.T,
) {
	subscription := &models.Subscription{
		Status: "ACTIVE",
	}
	subscription.ID = 401

	pppCalls := 0
	migrationCalls := 0
	ftpCalls := 0

	result, err := ReconcileCustomerInternetCredentialPostSave(
		subscription,
		"test-key",
		false,
		0,
		"",
		func(
			subscriptionID uint,
			keyMaterial string,
		) (PPPSecretReconciliationResult, error) {
			pppCalls++

			if subscriptionID != subscription.ID {
				t.Fatalf(
					"PPP subscription id = %d",
					subscriptionID,
				)
			}

			return PPPSecretReconciliationResult{}, nil
		},
		func(
			uint,
			uint,
			string,
			string,
		) (PPPSecretReconciliationResult, error) {
			migrationCalls++
			return PPPSecretReconciliationResult{}, nil
		},
		func(
			row *models.Subscription,
			keyMaterial string,
		) (ManagedFTPReconciliationResult, error) {
			ftpCalls++

			return ManagedFTPReconciliationResult{
				SubscriptionID: row.ID,
				Status:         "ACTIVE",
				Action:         "PASSWORD_SYNC_AND_UNLOCK",
				Executed:       true,
			}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if pppCalls != 1 {
		t.Fatalf("PPP calls = %d, want 1", pppCalls)
	}
	if migrationCalls != 0 {
		t.Fatalf(
			"migration calls = %d, want 0",
			migrationCalls,
		)
	}
	if ftpCalls != 1 {
		t.Fatalf("FTP calls = %d, want 1", ftpCalls)
	}

	if !result.PPPAttempted || !result.FTPAttempted {
		t.Fatalf("unexpected result: %+v", result)
	}

	if result.FTP.Action != "PASSWORD_SYNC_AND_UNLOCK" {
		t.Fatalf("FTP action = %q", result.FTP.Action)
	}
}

func TestCustomerInternetCredentialPostSaveIdentityChangeRunsMigrationAndFTP(
	t *testing.T,
) {
	subscription := &models.Subscription{
		Status: "ACTIVE",
	}
	subscription.ID = 402

	credentialCalls := 0
	migrationCalls := 0
	ftpCalls := 0

	_, err := ReconcileCustomerInternetCredentialPostSave(
		subscription,
		"test-key",
		true,
		77,
		"old-pppoe-user",
		func(
			uint,
			string,
		) (PPPSecretReconciliationResult, error) {
			credentialCalls++
			return PPPSecretReconciliationResult{}, nil
		},
		func(
			subscriptionID uint,
			oldRouterID uint,
			oldUsername string,
			keyMaterial string,
		) (PPPSecretReconciliationResult, error) {
			migrationCalls++

			if oldRouterID != 77 {
				t.Fatalf(
					"old router = %d, want 77",
					oldRouterID,
				)
			}

			if oldUsername != "old-pppoe-user" {
				t.Fatalf(
					"old username = %q",
					oldUsername,
				)
			}

			return PPPSecretReconciliationResult{}, nil
		},
		func(
			row *models.Subscription,
			keyMaterial string,
		) (ManagedFTPReconciliationResult, error) {
			ftpCalls++

			return ManagedFTPReconciliationResult{
				SubscriptionID: row.ID,
				Action:         "PASSWORD_SYNC_AND_UNLOCK",
			}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if credentialCalls != 0 {
		t.Fatalf(
			"credential calls = %d, want 0",
			credentialCalls,
		)
	}
	if migrationCalls != 1 {
		t.Fatalf(
			"migration calls = %d, want 1",
			migrationCalls,
		)
	}
	if ftpCalls != 1 {
		t.Fatalf("FTP calls = %d, want 1", ftpCalls)
	}
}

func TestCustomerInternetCredentialPostSavePPPFailureStillRunsFTP(
	t *testing.T,
) {
	subscription := &models.Subscription{}
	subscription.ID = 403

	ftpCalled := false

	result, err := ReconcileCustomerInternetCredentialPostSave(
		subscription,
		"test-key",
		false,
		0,
		"",
		func(
			uint,
			string,
		) (PPPSecretReconciliationResult, error) {
			return PPPSecretReconciliationResult{},
				errors.New("router unavailable")
		},
		func(
			uint,
			uint,
			string,
			string,
		) (PPPSecretReconciliationResult, error) {
			return PPPSecretReconciliationResult{}, nil
		},
		func(
			row *models.Subscription,
			keyMaterial string,
		) (ManagedFTPReconciliationResult, error) {
			ftpCalled = true

			return ManagedFTPReconciliationResult{
				SubscriptionID: row.ID,
				Action:         "PASSWORD_SYNC_AND_UNLOCK",
			}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.PPPError != "router unavailable" {
		t.Fatalf("PPP error = %q", result.PPPError)
	}

	if !ftpCalled {
		t.Fatal("FTP did not run after PPP failure")
	}

	if result.FTPError != "" {
		t.Fatalf("unexpected FTP error: %q", result.FTPError)
	}
}

func TestCustomerInternetCredentialPostSaveFTPFailurePreservesPPPSuccess(
	t *testing.T,
) {
	subscription := &models.Subscription{}
	subscription.ID = 404

	result, err := ReconcileCustomerInternetCredentialPostSave(
		subscription,
		"test-key",
		false,
		0,
		"",
		func(
			uint,
			string,
		) (PPPSecretReconciliationResult, error) {
			return PPPSecretReconciliationResult{}, nil
		},
		func(
			uint,
			uint,
			string,
			string,
		) (PPPSecretReconciliationResult, error) {
			return PPPSecretReconciliationResult{}, nil
		},
		func(
			row *models.Subscription,
			keyMaterial string,
		) (ManagedFTPReconciliationResult, error) {
			return ManagedFTPReconciliationResult{
					SubscriptionID: row.ID,
				},
				errors.New("FTP server unavailable")
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.PPPError != "" {
		t.Fatalf("unexpected PPP error: %q", result.PPPError)
	}

	if result.FTPError != "FTP server unavailable" {
		t.Fatalf("FTP error = %q", result.FTPError)
	}
}
