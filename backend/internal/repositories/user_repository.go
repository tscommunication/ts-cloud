package repositories

import (
	"strings"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func GetUserByUsername(username string) (*models.User, error) {

	var user models.User

	err := database.DB.
		Table("users").
		Select("users.*").
		Joins("LEFT JOIN customer_internet_accounts cia ON cia.customer_id = users.customer_id AND cia.deleted_at IS NULL").
		Where("LOWER(users.username) = LOWER(?) OR LOWER(cia.pp_po_e_username) = LOWER(?)", username, username).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByEmail(email string) (*models.User, error) {

	var user models.User

	err := database.DB.
		Where("email = ?", email).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByID(id uint) (*models.User, error) {

	var user models.User

	err := database.DB.
		First(&user, id).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByCustomerID(customerID uint) (*models.User, error) {
	var user models.User

	err := database.DB.
		Where("customer_id = ?", customerID).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUsers(page, limit int, search, sort, order string) ([]models.User, int64, error) {

	var users []models.User
	var total int64

	query := database.DB.Model(&models.User{}).
		Where("role <> ?", "customer")

	if search != "" {
		search = "%" + strings.ToLower(search) + "%"

		query = query.Where(
			"LOWER(name) LIKE ? OR LOWER(username) LIKE ? OR LOWER(email) LIKE ?",
			search,
			search,
			search,
		)
	}

	query.Count(&total)

	allowedSort := map[string]bool{
		"id":       true,
		"name":     true,
		"username": true,
		"email":    true,
		"role":     true,
	}

	if !allowedSort[sort] {
		sort = "id"
	}

	order = strings.ToUpper(order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	err := query.
		Order(sort + " " + order).
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&users).Error

	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func CreateUser(user *models.User) error {
	return database.DB.Create(user).Error
}

func UpdateUser(user *models.User) error {
	return database.DB.Save(user).Error
}

func DeleteUser(id uint) error {
	return database.DB.Delete(&models.User{}, id).Error
}

func CountActiveSuperadmins() (int64, error) {
	var count int64
	err := database.DB.Model(&models.User{}).Where("role = ? AND active = ?", "superadmin", true).Count(&count).Error
	return count, err
}
