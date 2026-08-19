package services

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestCustomerReferenceCRUDAndValidation(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:customer_reference_service?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	database.DB = db

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.CustomerReference{},
	); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}

	customer := models.Customer{
		CustomerCode: "CUS-REF-001",
		FullName:     "Reference Test",
		Mobile:       "01712345670",
		NID:          "1234567890125",
	}

	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}

	if _, err := CreateCustomerReference(
		customer.ID,
		CustomerReferenceInput{
			Name:   "",
			Mobile: "01712345671",
		},
	); err == nil {
		t.Fatal("expected blank reference name to fail")
	}

	if _, err := CreateCustomerReference(
		customer.ID,
		CustomerReferenceInput{
			Name:   "Invalid Mobile",
			Mobile: "12345",
		},
	); err == nil {
		t.Fatal("expected invalid reference mobile to fail")
	}

	first, err := CreateCustomerReference(
		customer.ID,
		CustomerReferenceInput{
			Name:     "Reference One",
			Mobile:   "01712345671",
			Address:  "Dhaka",
			Relation: "Brother",
			Note:     "Primary reference",
		},
	)
	if err != nil {
		t.Fatalf("create first reference: %v", err)
	}

	second, err := CreateCustomerReference(
		customer.ID,
		CustomerReferenceInput{
			Name:   "Reference Two",
			Mobile: "",
		},
	)
	if err != nil {
		t.Fatalf("create second reference: %v", err)
	}

	rows, err := ListCustomerReferences(customer.ID)
	if err != nil {
		t.Fatalf("list references: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("reference count = %d, want 2", len(rows))
	}

	got, err := GetCustomerReference(customer.ID, first.ID)
	if err != nil {
		t.Fatalf("get reference: %v", err)
	}

	if got.Name != "Reference One" {
		t.Fatalf("reference name = %q", got.Name)
	}

	if err := UpdateCustomerReference(
		got,
		CustomerReferenceInput{
			Name:     "Reference Updated",
			Mobile:   "01812345672",
			Address:  "Khulna",
			Relation: "Friend",
			Note:     "Updated note",
		},
	); err != nil {
		t.Fatalf("update reference: %v", err)
	}

	updated, err := GetCustomerReference(customer.ID, first.ID)
	if err != nil {
		t.Fatalf("reload updated reference: %v", err)
	}
	if updated.Name != "Reference Updated" ||
		updated.Mobile != "01812345672" {
		t.Fatalf("reference update not persisted: %+v", updated)
	}

	if err := DeleteCustomerReference(second); err != nil {
		t.Fatalf("delete reference: %v", err)
	}

	rows, err = ListCustomerReferences(customer.ID)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("reference count after delete = %d, want 1", len(rows))
	}
}
