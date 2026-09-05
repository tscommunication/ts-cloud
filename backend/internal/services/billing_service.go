package services

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

var billingRunMu sync.Mutex

type CustomerLedgerEntry struct {
	Date        time.Time `json:"date"`
	Type        string    `json:"type"`
	Reference   string    `json:"reference"`
	Description string    `json:"description"`
	Debit       float64   `json:"debit"`
	Credit      float64   `json:"credit"`
}

func RunDueBilling(now time.Time, triggeredBy uint) (*models.BillingRun, error) {
	billingRunMu.Lock()
	defer billingRunMu.Unlock()

	subscriptions, err := repositories.GetSubscriptionsDueForBilling(now)
	if err != nil {
		return nil, err
	}
	run := &models.BillingRun{RunDate: now, TriggeredBy: triggeredBy, Status: "RUNNING", Total: len(subscriptions)}
	if err := repositories.CreateBillingRun(run); err != nil {
		return nil, err
	}
	for index := range subscriptions {
		subscription := &subscriptions[index]
		billingDate := subscription.NextBillingDate
		exists, checkErr := repositories.InvoiceExistsForPeriod(subscription.ID, int(billingDate.Month()), billingDate.Year(), 0)
		item := &models.BillingRunItem{BillingRunID: run.ID, SubscriptionID: subscription.ID}
		if checkErr != nil {
			item.Status, item.ErrorMessage = "FAILED", checkErr.Error()
			run.FailedCount++
		} else if exists {
			item.Status = "SKIPPED"
			run.SkippedCount++
			subscription.NextBillingDate = nextBillingDate(billingDate)
			if err := repositories.UpdateSubscription(subscription); err != nil {
				item.Status, item.ErrorMessage = "FAILED", err.Error()
				run.SkippedCount--
				run.FailedCount++
			}
		} else {
			invoice := &models.Invoice{SubscriptionID: subscription.ID, BillMonth: int(billingDate.Month()), BillYear: billingDate.Year(), IssueDate: now, DueDate: now.AddDate(0, 0, 7), PackagePrice: subscription.Package.Price}
			if err := CreateInvoice(invoice); err != nil {
				item.Status, item.ErrorMessage = "FAILED", err.Error()
				run.FailedCount++
			} else {
				item.Status, item.InvoiceID = "CREATED", &invoice.ID
				run.CreatedCount++
				subscription.NextBillingDate = nextBillingDate(billingDate)
				if err := repositories.UpdateSubscription(subscription); err != nil {
					item.ErrorMessage = fmt.Sprintf("invoice created; next billing date update failed: %v", err)
				}
			}
		}
		if err := repositories.CreateBillingRunItem(item); err != nil {
			log.Printf("Billing run item: %v", err)
		}
	}
	switch {
	case run.FailedCount == 0:
		run.Status = "COMPLETED"
	case run.CreatedCount+run.SkippedCount > 0:
		run.Status = "PARTIAL"
	default:
		run.Status = "FAILED"
	}
	if err := repositories.SaveBillingRun(run); err != nil {
		return nil, err
	}
	return run, nil
}

// nextBillingDate advances a billing cycle without skipping short months.
// For example, a subscription billed on the 31st advances from 31 January to
// 28 February (or 29 February in a leap year), rather than rolling into March.
func nextBillingDate(current time.Time) time.Time {
	year, month, _ := current.Date()
	nextMonth := month + 1
	if nextMonth > time.December {
		nextMonth = time.January
		year++
	}

	day := current.Day()
	lastDay := time.Date(year, nextMonth+1, 0, 0, 0, 0, 0, current.Location()).Day()
	if day > lastDay {
		day = lastDay
	}

	return time.Date(year, nextMonth, day, current.Hour(), current.Minute(), current.Second(), current.Nanosecond(), current.Location())
}

func GetRecentBillingRuns() ([]models.BillingRun, error) {
	return repositories.GetRecentBillingRuns(10)
}
func GetBillingSummary(now time.Time) (*repositories.BillingSummary, error) {
	return repositories.GetBillingSummary(now)
}

func GetCustomerLedger(customerID uint) ([]CustomerLedgerEntry, error) {
	invoices, err := repositories.GetInvoicesByCustomer(customerID)
	if err != nil {
		return nil, err
	}
	payments, err := repositories.GetPaymentsByCustomer(customerID)
	if err != nil {
		return nil, err
	}
	entries := make([]CustomerLedgerEntry, 0, len(invoices)+len(payments))
	for _, invoice := range invoices {
		if invoice.Status != "CANCELLED" {
			entries = append(entries, CustomerLedgerEntry{Date: invoice.IssueDate, Type: "INVOICE", Reference: invoice.InvoiceNo, Description: fmt.Sprintf("Billing %02d/%d", invoice.BillMonth, invoice.BillYear), Debit: invoice.TotalAmount})
		}
	}
	for _, payment := range payments {
		if payment.Status == "SUCCESS" {
			entries = append(entries, CustomerLedgerEntry{Date: payment.PaymentDate, Type: "PAYMENT", Reference: payment.ReceiptNo, Description: payment.Method, Credit: payment.Amount})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Date.After(entries[j].Date) })
	return entries, nil
}

func StartBillingWorker() {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := RunDueBilling(time.Now(), 0); err != nil {
				log.Printf("Billing worker: %v", err)
			}
		}
	}()
}
