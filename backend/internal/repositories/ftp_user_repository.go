package repositories

import (
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func CreateFTPUser(user *models.FTPUser) error {
	return database.DB.Create(user).Error
}

func GetFTPUsers() ([]models.FTPUser, error) {
	var users []models.FTPUser

	err := database.DB.Preload("Customer").
		Preload("Subscription").
		Preload("FTPServer").
		Find(&users).Error

	return users, err
}

func GetFTPUserByID(id uint) (*models.FTPUser, error) {
	var user models.FTPUser

	err := database.DB.Preload("Customer").
		Preload("Subscription").
		Preload("FTPServer").
		First(&user, id).Error

	return &user, err
}

func GetFTPUserByServiceEntitlementID(serviceEntitlementID uint) (*models.FTPUser, error) {
	var user models.FTPUser

	err := database.DB.
		Preload("Customer").
		Preload("Subscription").
		Preload("FTPServer").
		Where("service_entitlement_id = ?", serviceEntitlementID).
		First(&user).Error

	return &user, err
}

func GetFTPUsersByCustomer(customerID uint) ([]models.FTPUser, error) {
	var users []models.FTPUser
	err := database.DB.
		Preload("FTPServer").
		Where("customer_id = ?", customerID).
		Order("id ASC").
		Find(&users).Error
	return users, err
}

func UpdateFTPUser(user *models.FTPUser) error {
	return database.DB.Save(user).Error
}
func DeleteFTPUser(id uint) error {
	return database.DB.
		Unscoped().
		Delete(&models.FTPUser{}, id).Error

}

// UpdateFTPUserStatus updates FTP user status.
func UpdateFTPUserStatus(id uint, status string) error {

	return database.DB.Model(&models.FTPUser{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func GetFTPUserByUsername(
	username string,
) (*models.FTPUser, error) {

	var user models.FTPUser

	err := database.DB.
		Where("username = ?", username).
		First(&user).Error

	return &user, err
}
