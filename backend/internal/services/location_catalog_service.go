package services

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"io"
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

//go:embed location_catalog.csv
var approvedLocationCatalogCSV string

var approvedLocationCatalogHeader = []string{
	"division",
	"district",
	"upazila",
	"post_office",
	"postal_code",
}

// Postal codes are catalog hints only. Customer postal_code remains an
// editable snapshot and is not required to match the location master.
func parseApprovedLocationCatalog(
	input io.Reader,
) ([]approvedLocationCatalogEntry, error) {
	reader := csv.NewReader(input)

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read approved location catalog: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("approved location catalog is empty")
	}

	header := rows[0]
	if len(header) != len(approvedLocationCatalogHeader) {
		return nil, fmt.Errorf(
			"approved location catalog header has %d columns; expected %d",
			len(header),
			len(approvedLocationCatalogHeader),
		)
	}

	for i, expected := range approvedLocationCatalogHeader {
		if strings.TrimSpace(header[i]) != expected {
			return nil, fmt.Errorf(
				"approved location catalog header column %d is %q; expected %q",
				i+1,
				header[i],
				expected,
			)
		}
	}

	entries := make([]approvedLocationCatalogEntry, 0, len(rows)-1)

	for rowIndex, row := range rows[1:] {
		if len(row) != len(approvedLocationCatalogHeader) {
			return nil, fmt.Errorf(
				"approved location catalog row %d has %d columns; expected %d",
				rowIndex+2,
				len(row),
				len(approvedLocationCatalogHeader),
			)
		}

		entry := approvedLocationCatalogEntry{
			Division:   normalizeLocationCatalogValue(row[0]),
			District:   normalizeLocationCatalogValue(row[1]),
			Upazila:    normalizeLocationCatalogValue(row[2]),
			PostOffice: normalizeLocationCatalogValue(row[3]),
			PostalCode: strings.TrimSpace(row[4]),
		}

		if entry.Division == "" ||
			entry.District == "" ||
			entry.Upazila == "" {
			return nil, fmt.Errorf(
				"approved location catalog row %d requires division, district and upazila",
				rowIndex+2,
			)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func approvedLocationCatalog() ([]approvedLocationCatalogEntry, error) {
	return parseApprovedLocationCatalog(
		strings.NewReader(approvedLocationCatalogCSV),
	)
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

	entries, err := approvedLocationCatalog()
	if err != nil {
		return nil, err
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = syncApprovedLocationCatalog(
			tx,
			entries,
		)
		return err
	})

	return result, err
}
