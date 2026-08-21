package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestCustomerCannotChangePortalPasswordIndependently(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:customer_portal_password_invariant?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })
	hash, err := bcrypt.GenerateFromPassword([]byte("pppoe-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	customerID := uint(1)
	user := models.User{Name: "Customer", Username: "CUS-000001", Email: "customer@invalid.test", Password: string(hash), Role: "customer", Active: true, CustomerID: &customerID}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/me/password", func(c *gin.Context) { c.Set("user_id", user.ID); c.Next() }, ChangeMyPassword)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/me/password", bytes.NewBufferString(`{"current_password":"pppoe-password","new_password":"different-password"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
