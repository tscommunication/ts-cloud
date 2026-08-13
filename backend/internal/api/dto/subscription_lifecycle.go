package dto

type RenewSubscriptionRequest struct {
	Months int `json:"months" binding:"required,min=1,max=12"`
}
