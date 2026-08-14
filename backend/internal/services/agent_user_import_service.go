package services

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AgentUserImportRow struct {
	RowNumber int    `json:"row_number"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	Active    bool   `json:"active"`
	AgentID   *uint  `json:"agent_id,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
	Status    string `json:"status"`
}

type AgentUserImportPreview struct {
	TotalRows   int                  `json:"total_rows"`
	ReadyRows   int                  `json:"ready_rows"`
	CreateRows  int                  `json:"create_rows"`
	UpdateRows  int                  `json:"update_rows"`
	SkippedRows int                  `json:"skipped_rows"`
	Rows        []AgentUserImportRow `json:"rows"`
	Warnings    []string             `json:"warnings"`
}

type AgentUserImportResult struct {
	TotalRows   int `json:"total_rows"`
	CreatedRows int `json:"created_rows"`
	UpdatedRows int `json:"updated_rows"`
	SkippedRows int `json:"skipped_rows"`
}

type agentUserSourceRow struct {
	RowNumber int
	Name      string
	Username  string
	Password  string
	Role      string
	Active    bool
}

func readAgentUserWorkbook(input io.Reader, filename string) ([]agentUserSourceRow, error) {
	if strings.ToLower(filepath.Ext(filename)) != ".xlsx" {
		return nil, errors.New("agent login import requires an .xlsx file")
	}
	book, err := excelize.OpenReader(input, excelize.Options{RawCellValue: false})
	if err != nil {
		return nil, fmt.Errorf("unable to open XLSX workbook: %w", err)
	}
	defer book.Close()
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("XLSX workbook contains no worksheets")
	}
	values, err := book.GetRows(sheets[0], excelize.Options{RawCellValue: false})
	if err != nil {
		return nil, fmt.Errorf("unable to read XLSX worksheet: %w", err)
	}
	if len(values) < 2 {
		return nil, errors.New("agent login workbook contains no data rows")
	}
	headers := map[string]int{}
	for index, value := range values[0] {
		headers[normalizeHeader(value)] = index
	}
	required := []string{"agentsname", "username", "password", "role", "activestate"}
	for _, name := range required {
		if _, ok := headers[name]; !ok {
			return nil, fmt.Errorf("missing required column %s", name)
		}
	}
	cell := func(row []string, key string) string {
		index := headers[key]
		if index >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[index])
	}
	seen := map[string]bool{}
	rows := make([]agentUserSourceRow, 0, len(values)-1)
	for index, row := range values[1:] {
		name := cell(row, "agentsname")
		username := strings.ToLower(cell(row, "username"))
		password := cell(row, "password")
		role := strings.ToLower(cell(row, "role"))
		if name == "" && username == "" && password == "" {
			continue
		}
		if name == "" || username == "" || password == "" {
			return nil, fmt.Errorf("row %d requires agent name, username and password", index+2)
		}
		if len(password) < 8 {
			return nil, fmt.Errorf("row %d password must contain at least 8 characters", index+2)
		}
		if role != "agent" && role != "superadmin" {
			return nil, fmt.Errorf("row %d role must be Agent or superadmin", index+2)
		}
		if seen[username] {
			return nil, fmt.Errorf("duplicate username %s", username)
		}
		seen[username] = true
		rows = append(rows, agentUserSourceRow{RowNumber: index + 2, Name: name, Username: username, Password: password, Role: role, Active: strings.EqualFold(cell(row, "activestate"), "active")})
	}
	return rows, nil
}

func normalizeHeader(value string) string {
	return normalizePersonName(strings.TrimPrefix(value, "\ufeff"))
}

func normalizePersonName(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}

func matchImportAgent(name string, agents []models.Agent) *models.Agent {
	wanted := normalizePersonName(name)
	var prefixMatches []models.Agent
	for _, agent := range agents {
		candidate := normalizePersonName(agent.Name)
		if candidate == wanted {
			matched := agent
			return &matched
		}
		if len(wanted) >= 8 && (strings.HasPrefix(candidate, wanted) || strings.HasPrefix(wanted, candidate)) {
			prefixMatches = append(prefixMatches, agent)
		}
	}
	if len(prefixMatches) == 1 {
		return &prefixMatches[0]
	}
	return nil
}

func buildAgentUserPreview(db *gorm.DB, source []agentUserSourceRow) (*AgentUserImportPreview, error) {
	var agents []models.Agent
	if err := db.Order("id ASC").Find(&agents).Error; err != nil {
		return nil, err
	}
	preview := &AgentUserImportPreview{TotalRows: len(source), Rows: make([]AgentUserImportRow, 0, len(source))}
	for _, sourceRow := range source {
		row := AgentUserImportRow{RowNumber: sourceRow.RowNumber, Name: sourceRow.Name, Username: sourceRow.Username, Role: sourceRow.Role, Active: sourceRow.Active}
		if sourceRow.Role == "agent" {
			agent := matchImportAgent(sourceRow.Name, agents)
			if agent == nil {
				row.Status = "SKIPPED_UNMATCHED_AGENT"
				preview.SkippedRows++
				preview.Rows = append(preview.Rows, row)
				continue
			}
			row.AgentID = &agent.ID
			row.AgentName = agent.Name
		}
		var count int64
		if err := db.Model(&models.User{}).Where("LOWER(username) = ?", sourceRow.Username).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			row.Status = "READY_UPDATE"
			preview.UpdateRows++
		} else {
			row.Status = "READY_CREATE"
			preview.CreateRows++
		}
		preview.ReadyRows++
		preview.Rows = append(preview.Rows, row)
	}
	preview.Warnings = []string{
		"Passwords are never returned by the preview and will be stored only as bcrypt hashes.",
		"Importing an existing username resets that account password to the workbook value.",
		"Agent rows without a unique Agent catalog match are skipped.",
	}
	return preview, nil
}

func PreviewAgentUserWorkbook(input io.Reader, filename string) (*AgentUserImportPreview, error) {
	rows, err := readAgentUserWorkbook(input, filename)
	if err != nil {
		return nil, err
	}
	return buildAgentUserPreview(database.DB, rows)
}

func ImportAgentUserWorkbook(input io.Reader, filename string) (*AgentUserImportResult, error) {
	rows, err := readAgentUserWorkbook(input, filename)
	if err != nil {
		return nil, err
	}
	result := &AgentUserImportResult{TotalRows: len(rows)}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		preview, previewErr := buildAgentUserPreview(tx, rows)
		if previewErr != nil {
			return previewErr
		}
		previewByRow := map[int]AgentUserImportRow{}
		for _, row := range preview.Rows {
			previewByRow[row.RowNumber] = row
		}
		for _, sourceRow := range rows {
			previewRow := previewByRow[sourceRow.RowNumber]
			if previewRow.Status == "SKIPPED_UNMATCHED_AGENT" {
				result.SkippedRows++
				continue
			}
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(sourceRow.Password), bcrypt.DefaultCost)
			if hashErr != nil {
				return fmt.Errorf("unable to secure password for row %d: %w", sourceRow.RowNumber, hashErr)
			}
			var user models.User
			lookupErr := tx.Where("LOWER(username) = ?", sourceRow.Username).First(&user).Error
			if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
				return lookupErr
			}
			if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
				user = models.User{Name: sourceRow.Name, Username: sourceRow.Username, Email: sourceRow.Username, Password: string(hash), Role: sourceRow.Role, Active: sourceRow.Active, AgentID: previewRow.AgentID}
				if err := tx.Create(&user).Error; err != nil {
					return fmt.Errorf("unable to create account %s: %w", sourceRow.Username, err)
				}
				result.CreatedRows++
				continue
			}
			user.Name = sourceRow.Name
			user.Username = sourceRow.Username
			if strings.TrimSpace(user.Email) == "" {
				user.Email = sourceRow.Username
			}
			user.Password = string(hash)
			user.Role = sourceRow.Role
			user.Active = sourceRow.Active
			user.AgentID = previewRow.AgentID
			if err := tx.Save(&user).Error; err != nil {
				return fmt.Errorf("unable to update account %s: %w", sourceRow.Username, err)
			}
			result.UpdatedRows++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
