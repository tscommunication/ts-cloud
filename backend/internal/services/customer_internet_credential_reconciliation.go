package services

import (
	"errors"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type CustomerInternetCredentialPPPReconciliationRunner func(
	subscriptionID uint,
	keyMaterial string,
) (PPPSecretReconciliationResult, error)

type CustomerInternetCredentialPPPMigrationRunner func(
	subscriptionID uint,
	oldRouterID uint,
	oldUsername string,
	keyMaterial string,
) (PPPSecretReconciliationResult, error)

type CustomerInternetCredentialFTPReconciliationRunner func(
	subscription *models.Subscription,
	keyMaterial string,
) (ManagedFTPReconciliationResult, error)

type CustomerInternetCredentialReconciliationResult struct {
	PPPAttempted bool
	PPP          PPPSecretReconciliationResult
	PPPError     string

	FTPAttempted bool
	FTP          ManagedFTPReconciliationResult
	FTPError     string
}

// ReconcileCustomerInternetCredentialPostSave reconciles external service
// projections only after the canonical customer internet credential has
// already been persisted.
//
// PPP/MikroTik and FTP are deliberately independent. Failure of either
// projection must not prevent the other from being attempted and must not
// roll back the saved canonical credential.
func ReconcileCustomerInternetCredentialPostSave(
	subscription *models.Subscription,
	keyMaterial string,
	identityChanged bool,
	oldRouterID uint,
	oldUsername string,
	pppCredentialRunner CustomerInternetCredentialPPPReconciliationRunner,
	pppMigrationRunner CustomerInternetCredentialPPPMigrationRunner,
	ftpRunner CustomerInternetCredentialFTPReconciliationRunner,
) (CustomerInternetCredentialReconciliationResult, error) {
	var result CustomerInternetCredentialReconciliationResult

	if subscription == nil || subscription.ID == 0 {
		return result, errors.New("subscription is required")
	}

	if pppCredentialRunner == nil {
		return result, errors.New(
			"PPP credential reconciliation runner is required",
		)
	}

	if pppMigrationRunner == nil {
		return result, errors.New(
			"PPP migration reconciliation runner is required",
		)
	}

	if ftpRunner == nil {
		return result, errors.New(
			"FTP reconciliation runner is required",
		)
	}

	result.PPPAttempted = true

	var pppErr error

	if identityChanged {
		result.PPP, pppErr = pppMigrationRunner(
			subscription.ID,
			oldRouterID,
			oldUsername,
			keyMaterial,
		)
	} else {
		result.PPP, pppErr = pppCredentialRunner(
			subscription.ID,
			keyMaterial,
		)
	}

	if pppErr != nil {
		result.PPPError = pppErr.Error()
	}

	// FTP must still run when MikroTik is unavailable. The canonical
	// credential is already committed and FTP reconciliation is an
	// independent projection.
	result.FTPAttempted = true

	ftpResult, ftpErr := ftpRunner(
		subscription,
		keyMaterial,
	)
	result.FTP = ftpResult

	if ftpErr != nil {
		result.FTPError = ftpErr.Error()
	}

	return result, nil
}

func ReconcileCustomerInternetCredentialWithManagedServicesPostSave(
	subscription *models.Subscription,
	keyMaterial string,
	identityChanged bool,
	oldRouterID uint,
	oldUsername string,
) (CustomerInternetCredentialReconciliationResult, error) {
	return ReconcileCustomerInternetCredentialPostSave(
		subscription,
		keyMaterial,
		identityChanged,
		oldRouterID,
		oldUsername,
		ReconcileSubscriptionPPPSecretCredentialWithMikroTik,
		ReconcileSubscriptionPPPMigrationWithMikroTik,
		ReconcileManagedFTPForSubscription,
	)
}
