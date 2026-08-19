package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupAuthCustomerIdentityTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			"file:auth_customer_identity?mode=memory&cache=shared",
		),
		&gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.Customer{},
		&models.User{},
	); err != nil {
		t.Fatal(err)
	}

	previousDB := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previousDB
	})

	return db
}

func TestCustomerLoginResponseContainsCustomerID(t *testing.T) {
	db := setupAuthCustomerIdentityTestDB(t)

	oldSecret := os.Getenv("JWT_SECRET")
	oldDBType := os.Getenv("DB_TYPE")
	oldDBPath := os.Getenv("DB_PATH")

	if err := os.Setenv("JWT_SECRET", "test-customer-secret"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("DB_TYPE", "sqlite"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("DB_PATH", ":memory:"); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = os.Setenv("JWT_SECRET", oldSecret)
		_ = os.Setenv("DB_TYPE", oldDBType)
		_ = os.Setenv("DB_PATH", oldDBPath)
	})

	customer := models.Customer{
		CustomerCode: "CUS-AUTH-001",
		FullName:     "Auth Customer",
		Mobile:       "01780000001",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte("secure-pass"),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatal(err)
	}

	user := models.User{
		Name:       "Auth Customer",
		Username:   "auth-customer",
		Email:      "auth-customer@example.com",
		Password:   string(hash),
		Role:       "customer",
		Active:     true,
		CustomerID: &customer.ID,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]any{
		"username": "auth-customer",
		"password": "secure-pass",
	})
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/login", Login)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response struct {
		User struct {
			ID         uint   `json:"id"`
			Role       string `json:"role"`
			CustomerID *uint  `json:"customer_id"`
		} `json:"user"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}

	if response.User.Role != "customer" {
		t.Fatalf(
			"expected customer role, got %q",
			response.User.Role,
		)
	}

	if response.User.CustomerID == nil ||
		*response.User.CustomerID != customer.ID {
		t.Fatalf(
			"expected customer_id %d, got %+v",
			customer.ID,
			response.User.CustomerID,
		)
	}
}
