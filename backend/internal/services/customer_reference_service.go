package services

import (
	"fmt"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
)

type CustomerReferenceInput struct {
	Name     string
	Mobile   string
	Address  string
	Relation string
	Note     string
}

func ValidateCustomerReference(input CustomerReferenceInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("reference name is required")
	}

	mobile := strings.TrimSpace(input.Mobile)
	if mobile != "" && !bangladeshMobileRegex.MatchString(mobile) {
		return fmt.Errorf(
			"reference mobile must be a valid 11-digit Bangladesh mobile number starting with 013-019",
		)
	}

	return nil
}

func ListCustomerReferences(
	customerID uint,
) ([]models.CustomerReference, error) {
	return repositories.ListCustomerReferences(customerID)
}

func GetCustomerReference(
	customerID uint,
	referenceID uint,
) (*models.CustomerReference, error) {
	return repositories.GetCustomerReference(
		customerID,
		referenceID,
	)
}

func CreateCustomerReference(
	customerID uint,
	input CustomerReferenceInput,
) (*models.CustomerReference, error) {
	if err := ValidateCustomerReference(input); err != nil {
		return nil, err
	}

	reference := &models.CustomerReference{
		CustomerID: customerID,
		Name:       strings.TrimSpace(input.Name),
		Mobile:     strings.TrimSpace(input.Mobile),
		Address:    strings.TrimSpace(input.Address),
		Relation:   strings.TrimSpace(input.Relation),
		Note:       strings.TrimSpace(input.Note),
	}

	if err := repositories.CreateCustomerReference(reference); err != nil {
		return nil, err
	}

	return reference, nil
}

func UpdateCustomerReference(
	reference *models.CustomerReference,
	input CustomerReferenceInput,
) error {
	if err := ValidateCustomerReference(input); err != nil {
		return err
	}

	reference.Name = strings.TrimSpace(input.Name)
	reference.Mobile = strings.TrimSpace(input.Mobile)
	reference.Address = strings.TrimSpace(input.Address)
	reference.Relation = strings.TrimSpace(input.Relation)
	reference.Note = strings.TrimSpace(input.Note)

	return repositories.UpdateCustomerReference(reference)
}

func DeleteCustomerReference(
	reference *models.CustomerReference,
) error {
	return repositories.DeleteCustomerReference(reference)
}
