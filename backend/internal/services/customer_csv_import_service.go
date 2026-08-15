package services

import (
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type CustomerCSVPreview struct {
	TotalRows    int      `json:"total_rows"`
	ActiveRows   int      `json:"active_rows"`
	InactiveRows int      `json:"inactive_rows"`
	Packages     []string `json:"packages"`
	POPs         []string `json:"pops"`
	Warnings     []string `json:"warnings"`
}

func readCustomerCSV(input io.Reader) ([]map[string]string, error) {
	r := csv.NewReader(input)
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	return parseCustomerRows(rows)
}

func readCustomerXLSX(input io.Reader) ([]map[string]string, error) {
	book, err := excelize.OpenReader(input, excelize.Options{RawCellValue: false})
	if err != nil {
		return nil, fmt.Errorf("unable to open XLSX workbook: %w", err)
	}
	defer book.Close()
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("XLSX workbook contains no worksheets")
	}
	rows, err := book.GetRows(sheets[0], excelize.Options{RawCellValue: false})
	if err != nil {
		return nil, fmt.Errorf("unable to read XLSX worksheet %s: %w", sheets[0], err)
	}
	return parseCustomerRows(rows)
}

func readCustomerFile(input io.Reader, filename string) ([]map[string]string, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".csv":
		return readCustomerCSV(input)
	case ".xlsx":
		return readCustomerXLSX(input)
	default:
		return nil, errors.New("unsupported file type; upload a .csv or .xlsx file")
	}
}

func parseCustomerRows(sourceRows [][]string) ([]map[string]string, error) {
	if len(sourceRows) == 0 {
		return nil, errors.New("file contains no header row")
	}
	head := sourceRows[0]
	for i := range head {
		head[i] = strings.TrimSpace(strings.TrimPrefix(head[i], "\ufeff"))
	}
	var rows []map[string]string
	for _, values := range sourceRows[1:] {
		if len(values) > len(head) {
			return nil, errors.New("data row has more columns than the header")
		}
		row := map[string]string{}
		for i, k := range head {
			if i < len(values) {
				row[k] = strings.TrimSpace(values[i])
			} else {
				row[k] = ""
			}
		}
		if len(strings.TrimSpace(strings.Join(values, ""))) == 0 {
			continue
		}
		rows = append(rows, row)
	}
	for _, required := range []string{"ID", "Username", "Status", "Package", "POP", "Name", "Contact", "Expire", "B Cycle"} {
		if len(rows) > 0 {
			if _, ok := rows[0][required]; !ok {
				return nil, fmt.Errorf("missing required column %s", required)
			}
		}
	}
	return rows, nil
}

func PreviewCustomerFile(input io.Reader, filename string) (*CustomerCSVPreview, error) {
	rows, err := readCustomerFile(input, filename)
	if err != nil {
		return nil, err
	}
	return previewCustomerRows(rows)
}

func PreviewCustomerCSV(input io.Reader) (*CustomerCSVPreview, error) {
	rows, err := readCustomerCSV(input)
	if err != nil {
		return nil, err
	}
	return previewCustomerRows(rows)
}

func previewCustomerRows(rows []map[string]string) (*CustomerCSVPreview, error) {
	p := &CustomerCSVPreview{TotalRows: len(rows)}
	packages, pops := map[string]bool{}, map[string]bool{}
	usernames := map[string]bool{}
	for _, row := range rows {
		if strings.EqualFold(row["Status"], "active") {
			p.ActiveRows++
		} else {
			p.InactiveRows++
		}
		packages[row["Package"]] = true
		pops[row["POP"]] = true
		u := strings.ToLower(row["Username"])
		if usernames[u] {
			return nil, fmt.Errorf("duplicate username %s", row["Username"])
		}
		usernames[u] = true
	}
	for v := range packages {
		p.Packages = append(p.Packages, v)
	}
	for v := range pops {
		p.POPs = append(p.POPs, v)
	}
	sort.Strings(p.Packages)
	sort.Strings(p.POPs)
	packageCatalog := importPackageCatalogs()
	for _, sourcePackage := range p.Packages {
		packageName := strings.TrimSpace(strings.TrimPrefix(sourcePackage, "Pack:"))
		if _, ok := packageCatalog[packageName]; !ok {
			return nil, fmt.Errorf("package %s is missing from the approved package catalog", packageName)
		}
	}
	distributionCatalog, err := importAgentPOPCatalogs()
	if err != nil {
		return nil, err
	}
	approvedPOPs := map[string]bool{}
	for _, item := range distributionCatalog {
		approvedPOPs[normalizedCatalogName(item.POPName)] = true
	}
	for _, sourcePOP := range p.POPs {
		if !approvedPOPs[normalizedCatalogName(sourcePOP)] {
			return nil, fmt.Errorf("source POP %s is missing from the approved Agent/POP catalog", sourcePOP)
		}
	}
	p.Warnings = []string{"All source packages match the approved package catalog.", "All source POPs match the approved Agent/POP catalog.", "Source-active subscriptions will be imported ACTIVE; source-deactive subscriptions will be SUSPENDED."}
	return p, nil
}

var speedPattern = regexp.MustCompile(`(?i)(?:_P|_)(\d+)(?:MB)?(?:$|[-_])`)

//go:embed package_catalog.csv
var packageCatalogCSV string

//go:embed agent_pop_catalog.csv
var agentPOPCatalogCSV string

type importPackageCatalog struct {
	SourceID         int
	Rate, Commission float64
	Profile          string
}

type importAgentPOPCatalog struct {
	ManagerID, ManagerName, ResellerType, ManagerAddress, ManagerContact string
	OpeningBalance                                                       float64
	POPID, POPName, POPLocation, POPContact                              string
}

func importAgentPOPCatalogs() ([]importAgentPOPCatalog, error) {
	rows, err := csv.NewReader(strings.NewReader(agentPOPCatalogCSV)).ReadAll()
	if err != nil {
		return nil, err
	}
	result := make([]importAgentPOPCatalog, 0, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) != 10 {
			return nil, fmt.Errorf("Agent/POP catalog row has %d columns; expected 10", len(row))
		}
		balance, err := strconv.ParseFloat(row[5], 64)
		if err != nil {
			return nil, fmt.Errorf("Agent/POP catalog manager %s opening balance: %w", row[0], err)
		}
		result = append(result, importAgentPOPCatalog{ManagerID: row[0], ManagerName: row[1], ResellerType: row[2], ManagerAddress: row[3], ManagerContact: row[4], OpeningBalance: balance, POPID: row[6], POPName: row[7], POPLocation: row[8], POPContact: row[9]})
	}
	return result, nil
}

func normalizedCatalogName(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func importPackageCatalogs() map[string]importPackageCatalog {
	rows, _ := csv.NewReader(strings.NewReader(packageCatalogCSV)).ReadAll()
	result := map[string]importPackageCatalog{}
	for _, row := range rows[1:] {
		sourceID, _ := strconv.Atoi(row[0])
		rate, _ := strconv.ParseFloat(row[2], 64)
		commission, _ := strconv.ParseFloat(row[4], 64)
		result[row[1]] = importPackageCatalog{SourceID: sourceID, Rate: rate, Profile: row[3], Commission: commission}
	}
	return result
}

func catalogPackageSpeed(name string) int {
	if match := speedPattern.FindStringSubmatch(name); len(match) > 1 {
		speed, _ := strconv.Atoi(match[1])
		return speed
	}
	return 0
}

func ImportCustomerCSV(input io.Reader, filename string, routerID uint) (*models.CustomerImportBatch, error) {
	rows, err := readCustomerCSV(input)
	if err != nil {
		return nil, err
	}
	return importCustomerRows(rows, filename, routerID)
}

func ImportCustomerFile(input io.Reader, filename string, routerID uint) (*models.CustomerImportBatch, error) {
	rows, err := readCustomerFile(input, filename)
	if err != nil {
		return nil, err
	}
	return importCustomerRows(rows, filename, routerID)
}

func importCustomerRows(rows []map[string]string, filename string, routerID uint) (*models.CustomerImportBatch, error) {
	if len(rows) == 0 {
		return nil, errors.New("file contains no customer rows")
	}
	seen := map[string]bool{}
	for _, row := range rows {
		username := strings.ToLower(strings.TrimSpace(row["Username"]))
		if username == "" || row["Name"] == "" || strings.Trim(row["Contact"], "'") == "" {
			return nil, fmt.Errorf("row %s is missing username, name or contact", row["ID"])
		}
		if seen[username] {
			return nil, fmt.Errorf("duplicate username %s", row["Username"])
		}
		seen[username] = true
	}
	if err := ValidateSubscriptionRouter(routerID); err != nil {
		return nil, err
	}
	batch := &models.CustomerImportBatch{Filename: filename, Status: "RUNNING", RouterID: routerID, TotalRows: len(rows), CreatedAt: time.Now()}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(batch).Error; err != nil {
			return err
		}
		pkgMap := map[string]uint{}
		createdPackages := 0
		catalog := importPackageCatalogs()
		distribution, err := syncApprovedDistributionCatalog(tx)
		if err != nil {
			return err
		}
		for _, row := range rows {
			var existing int64
			if err := tx.Model(&models.Subscription{}).Where("LOWER(pp_po_e_username) = LOWER(?)", row["Username"]).Count(&existing).Error; err != nil {
				return err
			}
			if existing > 0 {
				return fmt.Errorf("PPPoE username %s already exists in TS-Cloud", row["Username"])
			}
			pkgName, popName := strings.TrimPrefix(row["Package"], "Pack:"), row["POP"]
			if pkgMap[pkgName] == 0 {
				catalogRow, ok := catalog[pkgName]
				if !ok {
					return fmt.Errorf("package %s is missing from the approved package catalog", pkgName)
				}
				var pkg models.Package
				err := tx.Where("LOWER(name) = LOWER(?)", pkgName).First(&pkg).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					speed := catalogPackageSpeed(pkgName)
					pkg = models.Package{PackageCode: fmt.Sprintf("CAT-%06d", catalogRow.SourceID), Name: pkgName, Price: catalogRow.Rate, Commission: catalogRow.Commission, DownloadSpeed: speed, UploadSpeed: speed, ValidityDays: 30, MikroTikProfile: catalogRow.Profile, Status: "ACTIVE", Description: fmt.Sprintf("Approved Packages List.xlsx catalog; source ID=%d", catalogRow.SourceID)}
					if err := tx.Create(&pkg).Error; err != nil {
						return err
					}
					createdPackages++
				} else if err != nil {
					return err
				}
				pkgMap[pkgName] = pkg.ID
			}
			popKey := normalizedCatalogName(popName)
			popID := distribution.POPIDs[popKey]
			if popID == 0 {
				return fmt.Errorf("source POP %s is missing from the approved Agent/POP catalog", popName)
			}
			agentID := distribution.POPAgentIDs[popKey]
			if agentID == 0 {
				return fmt.Errorf("POP %s has no approved agent", popName)
			}
			mobile := strings.Trim(strings.TrimSpace(row["Contact"]), "'")
			billing, _ := strconv.Atoi(row["B Cycle"])
			if billing < 1 || billing > 31 {
				billing = 1
			}
			activation := parseImportDate(row["J Date"])
			if activation.IsZero() {
				activation = parseImportDate(row["C Date"])
			}
			if activation.IsZero() {
				activation = time.Now()
			}
			expiry := parseImportDate(row["Expire"])
			if expiry.IsZero() {
				expiry = activation.AddDate(0, 1, 0)
			}
			status := "ACTIVE"
			if !strings.EqualFold(row["Status"], "active") {
				status = "INACTIVE"
			}
			legacyAddress := importAddress(row)
			roadOrArea := importAddressParts(row, "Area", "Block", "Road Name", "Road No")
			villageOrHolding := importAddressParts(row, "Building Name", "Building No", "Flat")
			customer := models.Customer{CustomerCode: fmt.Sprintf("IMP-%d-%s", batch.ID, row["ID"]), FullName: row["Name"], Mobile: mobile, FatherName: row["Father Name"], MotherName: row["Mother Name"], NID: row["NID"], Email: row["Email"], Country: "Bangladesh", RoadOrArea: roadOrArea, VillageOrHolding: villageOrHolding, Address: legacyAddress, PopID: ptrUint(popID), AgentID: ptrUint(agentID), Status: status, BillingDay: billing, ActivationDate: &activation}
			if err := tx.Create(&customer).Error; err != nil {
				return fmt.Errorf("row %s customer: %w", row["ID"], err)
			}
			balance, _ := strconv.ParseFloat(strings.Trim(row["Balance"], "'"), 64)
			subStatus := "ACTIVE"
			if status != "ACTIVE" {
				subStatus = "SUSPENDED"
			}
			sub := models.Subscription{SubscriptionCode: fmt.Sprintf("IMP-%d-%s", batch.ID, row["ID"]), CustomerID: customer.ID, PackageID: pkgMap[pkgName], ActivationDate: activation, BillingDay: billing, NextBillingDate: expiry, ExpiryDate: expiry, Status: subStatus, RouterID: routerID, PPPoEUsername: row["Username"], DueAmount: balance, Remarks: fmt.Sprintf("Import source status=%s; source POP=%s; IP=%s; MAC=%s; %s", row["Status"], popName, row["IP Address"], row["Mac"], row["Remarks"])}
			if err := tx.Create(&sub).Error; err != nil {
				return fmt.Errorf("row %s subscription: %w", row["ID"], err)
			}
			if err := tx.Create(&models.CustomerImportItem{BatchID: batch.ID, SourceID: row["ID"], Username: row["Username"], CustomerID: customer.ID, SubscriptionID: sub.ID}).Error; err != nil {
				return err
			}
			batch.ImportedRows++
		}
		batch.CreatedPackages = createdPackages
		batch.CreatedPOPs = distribution.CreatedPOPs
		batch.CreatedAgents = distribution.CreatedAgents
		batch.Status = "COMPLETED"
		return tx.Save(batch).Error
	})
	if err != nil {
		return nil, err
	}
	return batch, nil
}
func parseImportDate(v string) time.Time {
	value := strings.TrimSpace(v)
	for _, layout := range []string{"2006-01-02", "1/2/2006", "01/02/2006", "1/2/06", "02-Jan-2006", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
func ptrUint(v uint) *uint { return &v }
func importAddressParts(r map[string]string, keys ...string) string {
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if value := strings.TrimSpace(r[key]); value != "" {
			out = append(out, value)
		}
	}
	return strings.Join(out, ", ")
}

func importAddress(r map[string]string) string {
	return importAddressParts(
		r,
		"Area",
		"Block",
		"Road Name",
		"Road No",
		"Building Name",
		"Building No",
		"Flat",
	)
}
