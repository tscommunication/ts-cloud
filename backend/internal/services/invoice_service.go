package services

import (
	"errors"
	"fmt"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

func CreateInvoice(invoice *models.Invoice) error {

	// Load Subscription
	subscription, err := repositories.GetSubscriptionByID(invoice.SubscriptionID)
	if err != nil {
		return errors.New("subscription not found")
	}

	// Debug
	fmt.Println("========== CREATE INVOICE DEBUG ==========")
	fmt.Printf("SubscriptionID : %d\n", subscription.ID)
	fmt.Printf("CustomerID     : %d\n", subscription.CustomerID)
	fmt.Printf("PackageID      : %d\n", subscription.PackageID)

	// Copy IDs
	invoice.CustomerID = subscription.CustomerID
	invoice.PackageID = subscription.PackageID

	fmt.Println("---------- Before Save ----------")
	fmt.Printf("Invoice.CustomerID : %d\n", invoice.CustomerID)
	fmt.Printf("Invoice.PackageID  : %d\n", invoice.PackageID)
	fmt.Println("=================================")

	// Save
	return repositories.CreateInvoice(invoice)
}

func GetInvoices() ([]models.Invoice, error) {
	return repositories.GetInvoices()
}

func GetInvoiceByID(id uint) (*models.Invoice, error) {
	return repositories.GetInvoiceByID(id)
}

func UpdateInvoice(invoice *models.Invoice) error {
	return repositories.UpdateInvoice(invoice)
}

func DeleteInvoice(id uint) error {
	return repositories.DeleteInvoice(id)
}
