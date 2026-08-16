package services

import (
	"fmt"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/gorm"
)

type approvedLocationCatalogEntry struct {
	Division   string
	District   string
	Upazila    string
	PostOffice string
	PostalCode string
}

type LocationCatalogSyncResult struct {
	CreatedDivisions   int
	CreatedDistricts   int
	CreatedUpazilas    int
	CreatedPostOffices int
}

// approvedLocationCatalog is intentionally empty until an approved,
// repository-tracked Bangladesh location source is added.
//
// Postal codes are catalog hints only. Customer postal_code remains an
// editable snapshot and is not required to match the location master.
func approvedLocationCatalog() []approvedLocationCatalogEntry {
	return nil
}

func normalizeLocationCatalogValue(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func syncApprovedLocationCatalog(
	tx *gorm.DB,
	entries []approvedLocationCatalogEntry,
) (*LocationCatalogSyncResult, error) {
	result := &LocationCatalogSyncResult{}

	for _, raw := range entries {
		entry := approvedLocationCatalogEntry{
			Division:   normalizeLocationCatalogValue(raw.Division),
			District:   normalizeLocationCatalogValue(raw.District),
			Upazila:    normalizeLocationCatalogValue(raw.Upazila),
			PostOffice: normalizeLocationCatalogValue(raw.PostOffice),
			PostalCode: strings.TrimSpace(raw.PostalCode),
		}

		if entry.Division == "" ||
			entry.District == "" ||
			entry.Upazila == "" {
			return nil, fmt.Errorf(
				"location catalog requires division, district and upazila",
			)
		}

		var division models.Division
		resultDB := tx.
			Where("LOWER(name) = LOWER(?)", entry.Division).
			First(&division)

		if resultDB.Error != nil {
			if resultDB.Error != gorm.ErrRecordNotFound {
				return nil, resultDB.Error
			}

			division = models.Division{Name: entry.Division}
			if err := tx.Create(&division).Error; err != nil {
				return nil, err
			}
			result.CreatedDivisions++
		}

		var district models.District
		resultDB = tx.
			Where(
				"division_id = ? AND LOWER(name) = LOWER(?)",
				division.ID,
				entry.District,
			).
			First(&district)

		if resultDB.Error != nil {
			if resultDB.Error != gorm.ErrRecordNotFound {
				return nil, resultDB.Error
			}

			district = models.District{
				DivisionID: division.ID,
				Name:       entry.District,
			}
			if err := tx.Create(&district).Error; err != nil {
				return nil, err
			}
			result.CreatedDistricts++
		}

		var upazila models.Upazila
		resultDB = tx.
			Where(
				"district_id = ? AND LOWER(name) = LOWER(?)",
				district.ID,
				entry.Upazila,
			).
			First(&upazila)

		if resultDB.Error != nil {
			if resultDB.Error != gorm.ErrRecordNotFound {
				return nil, resultDB.Error
			}

			upazila = models.Upazila{
				DistrictID: district.ID,
				Name:       entry.Upazila,
			}
			if err := tx.Create(&upazila).Error; err != nil {
				return nil, err
			}
			result.CreatedUpazilas++
		}

		if entry.PostOffice == "" {
			continue
		}

		var postOffice models.PostOffice
		resultDB = tx.
			Where(
				"upazila_id = ? AND LOWER(name) = LOWER(?)",
				upazila.ID,
				entry.PostOffice,
			).
			First(&postOffice)

		if resultDB.Error != nil {
			if resultDB.Error != gorm.ErrRecordNotFound {
				return nil, resultDB.Error
			}

			postOffice = models.PostOffice{
				UpazilaID:  upazila.ID,
				Name:       entry.PostOffice,
				PostalCode: entry.PostalCode,
			}
			if err := tx.Create(&postOffice).Error; err != nil {
				return nil, err
			}
			result.CreatedPostOffices++
			continue
		}

		// Never replace a known master postal code with an empty catalog value.
		// A non-empty approved catalog value may refresh the master hint.
		if entry.PostalCode != "" &&
			postOffice.PostalCode != entry.PostalCode {
			postOffice.PostalCode = entry.PostalCode
			if err := tx.Save(&postOffice).Error; err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func SyncApprovedLocationCatalog() (*LocationCatalogSyncResult, error) {
	var result *LocationCatalogSyncResult

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = syncApprovedLocationCatalog(
			tx,
			approvedLocationCatalog(),
		)
		return err
	})

	return result, err
}
