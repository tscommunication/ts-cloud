package dto

import (
	"time"

	"github.com/tscommunication/ts-cloud/internal/models"
)

type CustomerReferenceRequest struct {
	Name     string `json:"name" binding:"required"`
	Mobile   string `json:"mobile"`
	Address  string `json:"address"`
	Relation string `json:"relation"`
	Note     string `json:"note"`
}

type CustomerReferenceResponse struct {
	ID         uint      `json:"id"`
	CustomerID uint      `json:"customer_id"`
	Name       string    `json:"name"`
	Mobile     string    `json:"mobile"`
	Address    string    `json:"address"`
	Relation   string    `json:"relation"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func ToCustomerReferenceResponse(
	row models.CustomerReference,
) CustomerReferenceResponse {
	return CustomerReferenceResponse{
		ID:         row.ID,
		CustomerID: row.CustomerID,
		Name:       row.Name,
		Mobile:     row.Mobile,
		Address:    row.Address,
		Relation:   row.Relation,
		Note:       row.Note,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}
