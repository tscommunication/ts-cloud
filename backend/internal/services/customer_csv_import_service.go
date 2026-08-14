package services

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
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
	head, err := r.Read()
	if err != nil {
		return nil, err
	}
	for i := range head {
		head[i] = strings.TrimSpace(strings.TrimPrefix(head[i], "\ufeff"))
	}
	var rows []map[string]string
	for {
		values, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(values) != len(head) {
			return nil, errors.New("CSV row has an invalid column count")
		}
		row := map[string]string{}
		for i, k := range head {
			row[k] = strings.TrimSpace(values[i])
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

func PreviewCustomerCSV(input io.Reader) (*CustomerCSVPreview, error) {
	rows, err := readCustomerCSV(input)
	if err != nil {
		return nil, err
	}
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
	p.Warnings = []string{"Imported packages will be INACTIVE until price and speed are reviewed.", "Imported subscriptions will be SUSPENDED until their packages are activated."}
	return p, nil
}

var speedPattern = regexp.MustCompile(`(?i)_P(\d+)`)

func ImportCustomerCSV(input io.Reader, filename string, routerID uint) (*models.CustomerImportBatch, error) {
	rows, err := readCustomerCSV(input)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("CSV contains no customer rows")
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
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(batch).Error; err != nil {
			return err
		}
		pkgMap := map[string]uint{}
		popMap := map[string]uint{}
		for _, row := range rows {
			var existing int64
			if err := tx.Model(&models.Subscription{}).Where("LOWER(pp_po_e_username) = LOWER(?)", row["Username"]).Count(&existing).Error; err != nil {
				return err
			}
			if existing > 0 {
				return fmt.Errorf("PPPoE username %s already exists in TS-Cloud", row["Username"])
			}
			pkgName, popName := row["Package"], row["POP"]
			if pkgMap[pkgName] == 0 {
				speed := 0
				if m := speedPattern.FindStringSubmatch(pkgName); len(m) > 1 {
					speed, _ = strconv.Atoi(m[1])
				}
				pkg := models.Package{PackageCode: fmt.Sprintf("IMP-%d-P%02d", batch.ID, len(pkgMap)+1), Name: pkgName, DownloadSpeed: speed, UploadSpeed: speed, ValidityDays: 30, Status: "INACTIVE", Description: "CSV import; review price, speed and MikroTik profile before activation"}
				if err := tx.Create(&pkg).Error; err != nil {
					return err
				}
				pkgMap[pkgName] = pkg.ID
			}
			if popMap[popName] == 0 {
				pop := models.POP{Code: fmt.Sprintf("IMP-%d-O%02d", batch.ID, len(popMap)+1), Name: popName, Status: "ACTIVE"}
				if err := tx.Create(&pop).Error; err != nil {
					return err
				}
				popMap[popName] = pop.ID
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
			customer := models.Customer{CustomerCode: fmt.Sprintf("IMP-%d-%s", batch.ID, row["ID"]), FullName: row["Name"], Mobile: mobile, FatherName: row["Father Name"], MotherName: row["Mother Name"], NID: row["NID"], Email: row["Email"], Address: importAddress(row), PopID: ptrUint(popMap[popName]), Status: status, BillingDay: billing, ActivationDate: &activation}
			if err := tx.Create(&customer).Error; err != nil {
				return fmt.Errorf("row %s customer: %w", row["ID"], err)
			}
			balance, _ := strconv.ParseFloat(strings.Trim(row["Balance"], "'"), 64)
			sub := models.Subscription{SubscriptionCode: fmt.Sprintf("IMP-%d-%s", batch.ID, row["ID"]), CustomerID: customer.ID, PackageID: pkgMap[pkgName], ActivationDate: activation, BillingDay: billing, NextBillingDate: expiry, ExpiryDate: expiry, Status: "SUSPENDED", RouterID: routerID, PPPoEUsername: row["Username"], DueAmount: balance, Remarks: fmt.Sprintf("CSV source status=%s; source POP=%s; IP=%s; MAC=%s; %s", row["Status"], popName, row["IP Address"], row["Mac"], row["Remarks"])}
			if err := tx.Create(&sub).Error; err != nil {
				return fmt.Errorf("row %s subscription: %w", row["ID"], err)
			}
			if err := tx.Create(&models.CustomerImportItem{BatchID: batch.ID, SourceID: row["ID"], Username: row["Username"], CustomerID: customer.ID, SubscriptionID: sub.ID}).Error; err != nil {
				return err
			}
			batch.ImportedRows++
		}
		batch.CreatedPackages = len(pkgMap)
		batch.CreatedPOPs = len(popMap)
		batch.Status = "COMPLETED"
		return tx.Save(batch).Error
	})
	if err != nil {
		return nil, err
	}
	return batch, nil
}
func parseImportDate(v string) time.Time {
	t, _ := time.Parse("2006-01-02", strings.TrimSpace(v))
	return t
}
func ptrUint(v uint) *uint { return &v }
func importAddress(r map[string]string) string {
	parts := []string{r["Area"], r["Block"], r["Road Name"], r["Road No"], r["Building Name"], r["Building No"], r["Flat"]}
	out := []string{}
	for _, v := range parts {
		if v != "" {
			out = append(out, v)
		}
	}
	return strings.Join(out, ", ")
}
