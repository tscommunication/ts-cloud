package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/xuri/excelize/v2"
)

type DataExport struct {
	Filename    string
	ContentType string
	Data        []byte
}

var customerExportHeaders = []string{"ID", "Username", "Code", "Status", "Expire", "B Cycle", "Package", "OTC", "POP", "Name", "Contact", "Father Name", "Mother Name", "NID", "Area", "Block", "Road Name", "Road No", "Building Name", "Building No", "Flat", "Box", "OLT/PON", "Latitude", "Longitude", "Cable Type", "C Date", "J Date", "Remarks", "Balance", "Email", "Mac", "IP Address"}

func ExportData(kind, format string) (*DataExport, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "csv" && format != "xlsx" {
		return nil, errors.New("export format must be csv or xlsx")
	}
	var headers []string
	var rows [][]string
	var err error
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "customers":
		headers = customerExportHeaders
		rows, err = customerExportRows()
	case "agent-users":
		headers = []string{"ID", "Name", "Username", "Email", "Role", "Active State", "Agent Code", "Agent Name", "POPs"}
		rows, err = agentUserExportRows()
	default:
		return nil, errors.New("export type must be customers or agent-users")
	}
	if err != nil {
		return nil, err
	}
	stamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("ts-cloud-%s-%s.%s", kind, stamp, format)
	if format == "csv" {
		data, exportErr := exportCSV(headers, rows)
		return &DataExport{Filename: filename, ContentType: "text/csv; charset=utf-8", Data: data}, exportErr
	}
	data, err := exportXLSX(headers, rows)
	return &DataExport{Filename: filename, ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Data: data}, err
}

func customerExportRows() ([][]string, error) {
	var subscriptions []models.Subscription
	err := database.DB.Preload("Customer.POP").Preload("Package").Order("subscriptions.id ASC").Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}
	rows := make([][]string, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		customer := subscription.Customer
		popName := ""
		if customer.POP != nil {
			popName = customer.POP.Name
		}
		status := "deactive"
		if strings.EqualFold(subscription.Status, "ACTIVE") && strings.EqualFold(customer.Status, "ACTIVE") {
			status = "active"
		}
		activation := subscription.ActivationDate.Format("2006-01-02")
		area := strings.TrimSpace(customer.RoadOrArea)
		if area == "" {
			area = strings.TrimSpace(customer.Address)
		}
		buildingName := strings.TrimSpace(customer.VillageOrHolding)
		rows = append(rows, []string{
			strconv.FormatUint(uint64(customer.ID), 10), subscription.PPPoEUsername, customer.CustomerCode, status,
			subscription.ExpiryDate.Format("2006-01-02"), strconv.Itoa(subscription.BillingDay), "Pack:" + subscription.Package.Name,
			"0", popName, customer.FullName, customer.Mobile, customer.FatherName, customer.MotherName, customer.NID,
			area, "", "", "", buildingName, "", "", "", "", "", "", "", customer.CreatedAt.Format("2006-01-02"),
			activation, subscription.Remarks, strconv.FormatFloat(subscription.DueAmount, 'f', 2, 64), customer.Email, "", "",
		})
	}
	return rows, nil
}

func agentUserExportRows() ([][]string, error) {
	var users []models.User
	if err := database.DB.Preload("Agent.AgentPOPs.POP").Preload("Agent.POP").Order("users.id ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	rows := make([][]string, 0, len(users))
	for _, user := range users {
		agentCode, agentName := "", ""
		popNames := []string{}
		if user.Agent != nil {
			agentCode, agentName = user.Agent.Code, user.Agent.Name
			for _, membership := range user.Agent.AgentPOPs {
				if membership.POP.Name != "" {
					popNames = append(popNames, membership.POP.Name)
				}
			}
			if len(popNames) == 0 && user.Agent.POP.Name != "" {
				popNames = append(popNames, user.Agent.POP.Name)
			}
			sort.Strings(popNames)
		}
		active := "Disabled"
		if user.Active {
			active = "Active"
		}
		rows = append(rows, []string{strconv.FormatUint(uint64(user.ID), 10), user.Name, user.Username, user.Email, user.Role, active, agentCode, agentName, strings.Join(popNames, ", ")})
	}
	return rows, nil
}

func exportCSV(headers []string, rows [][]string) ([]byte, error) {
	var output bytes.Buffer
	output.WriteString("\ufeff")
	writer := csv.NewWriter(&output)
	if err := writer.Write(headers); err != nil {
		return nil, err
	}
	if err := writer.WriteAll(rows); err != nil {
		return nil, err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func exportXLSX(headers []string, rows [][]string) ([]byte, error) {
	book := excelize.NewFile()
	defer book.Close()
	sheet := book.GetSheetName(0)
	headerValues := make([]interface{}, len(headers))
	for index := range headers {
		headerValues[index] = headers[index]
	}
	if err := book.SetSheetRow(sheet, "A1", &headerValues); err != nil {
		return nil, err
	}
	for rowIndex, row := range rows {
		values := make([]interface{}, len(row))
		for columnIndex := range row {
			values[columnIndex] = row[columnIndex]
		}
		cell, _ := excelize.CoordinatesToCellName(1, rowIndex+2)
		if err := book.SetSheetRow(sheet, cell, &values); err != nil {
			return nil, err
		}
	}
	style, err := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"1F4E78"}, Pattern: 1}})
	if err != nil {
		return nil, err
	}
	lastColumn, _ := excelize.ColumnNumberToName(len(headers))
	if err := book.SetCellStyle(sheet, "A1", lastColumn+"1", style); err != nil {
		return nil, err
	}
	if err := book.SetPanes(sheet, &excelize.Panes{Freeze: true, Split: false, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return nil, err
	}
	if err := book.SetColWidth(sheet, "A", lastColumn, 18); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := book.Write(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
