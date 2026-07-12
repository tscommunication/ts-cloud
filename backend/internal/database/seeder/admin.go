package seeder

import (
	"log"

	"golang.org/x/crypto/bcrypt"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func SeedAdmin() {

	var count int64

	database.DB.Model(&models.User{}).Where("username = ?", "tsadmin").Count(&count)

	if count > 0 {
		log.Println("Admin already exists")
		return
	}

	password, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)

	admin := models.User{
		Name:     "Tariqul Islam",
		Username: "tsadmin",
		Email:    "ts.communicationmagura@gmail.com",
		Password: string(password),
		Role:     "superadmin",
		Active:   true,
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		log.Fatal(err)
	}

	log.Println("Super Admin created")
}
