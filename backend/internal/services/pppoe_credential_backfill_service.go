package services

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/security"
)

type PPPoECredentialBackfillResult struct {
	SubscriptionsUpdated     int
	ProvisionRequestsUpdated int
}

func BackfillLegacyPPPoECredentials(
	keyMaterial string,
) (PPPoECredentialBackfillResult, error) {
	if strings.TrimSpace(keyMaterial) == "" {
		return PPPoECredentialBackfillResult{},
			fmt.Errorf("credential encryption key is required")
	}

	result := PPPoECredentialBackfillResult{}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var subscriptions []models.Subscription

		if err := tx.
			Where(
				"TRIM(COALESCE(pp_po_e_password, '')) <> '' AND " +
					"TRIM(COALESCE(pp_po_e_password_encrypted, '')) = ''",
			).
			Find(&subscriptions).Error; err != nil {
			return fmt.Errorf(
				"load legacy subscription PPPoE credentials: %w",
				err,
			)
		}

		for i := range subscriptions {
			row := &subscriptions[i]

			plaintext := strings.TrimSpace(
				row.PPPoEPassword,
			)
			if plaintext == "" {
				continue
			}

			encrypted, err := security.EncryptSecret(
				plaintext,
				keyMaterial,
			)
			if err != nil {
				return fmt.Errorf(
					"encrypt subscription PPPoE credential %d: %w",
					row.ID,
					err,
				)
			}

			update := tx.
				Model(&models.Subscription{}).
				Where(
					"id = ? AND "+
						"TRIM(COALESCE(pp_po_e_password_encrypted, '')) = ''",
					row.ID,
				).
				Updates(map[string]interface{}{
					"pp_po_e_password_encrypted": encrypted,
					"pp_po_e_password":           "",
				})

			if update.Error != nil {
				return fmt.Errorf(
					"backfill subscription PPPoE credential %d: %w",
					row.ID,
					update.Error,
				)
			}

			if update.RowsAffected > 0 {
				result.SubscriptionsUpdated++
			}
		}

		var requests []models.CustomerProvisionRequest

		if err := tx.
			Where(
				"TRIM(COALESCE(pp_po_e_password, '')) <> '' AND " +
					"TRIM(COALESCE(pp_po_e_password_encrypted, '')) = ''",
			).
			Find(&requests).Error; err != nil {
			return fmt.Errorf(
				"load legacy provision PPPoE credentials: %w",
				err,
			)
		}

		for i := range requests {
			row := &requests[i]

			plaintext := strings.TrimSpace(
				row.PPPoEPassword,
			)
			if plaintext == "" {
				continue
			}

			encrypted, err := security.EncryptSecret(
				plaintext,
				keyMaterial,
			)
			if err != nil {
				return fmt.Errorf(
					"encrypt provision PPPoE credential %d: %w",
					row.ID,
					err,
				)
			}

			update := tx.
				Model(&models.CustomerProvisionRequest{}).
				Where(
					"id = ? AND "+
						"TRIM(COALESCE(pp_po_e_password_encrypted, '')) = ''",
					row.ID,
				).
				Updates(map[string]interface{}{
					"pp_po_e_password_encrypted": encrypted,
					"pp_po_e_password":           "",
				})

			if update.Error != nil {
				return fmt.Errorf(
					"backfill provision PPPoE credential %d: %w",
					row.ID,
					update.Error,
				)
			}

			if update.RowsAffected > 0 {
				result.ProvisionRequestsUpdated++
			}
		}

		return nil
	})

	if err != nil {
		return PPPoECredentialBackfillResult{}, err
	}

	return result, nil
}
