package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/auth"
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func setupAuthMiddlewareCustomerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open("file:auth-middleware-customer?mode=memory&cache=shared"),
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

	previous := database.DB
	database.DB = db

	oldSecret := os.Getenv("JWT_SECRET")
	oldDBType := os.Getenv("DB_TYPE")
	oldDBPath := os.Getenv("DB_PATH")

	if err := os.Setenv("JWT_SECRET", "middleware-test-secret"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("DB_TYPE", "sqlite"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("DB_PATH", ":memory:"); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		database.DB = previous
		_ = os.Setenv("JWT_SECRET", oldSecret)
		_ = os.Setenv("DB_TYPE", oldDBType)
		_ = os.Setenv("DB_PATH", oldDBPath)
	})

	return db
}

func TestAuthMiddlewareSetsCustomerIDFromCurrentDBUser(t *testing.T) {
	db := setupAuthMiddlewareCustomerTestDB(t)

	customer := models.Customer{
		CustomerCode: "CUS-MW-001",
		FullName:     "Middleware Customer",
		Mobile:       "01781000001",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	user := models.User{
		Name:       "Middleware Customer",
		Username:   "middleware-customer",
		Email:      "middleware@example.com",
		Password:   "hash",
		Role:       "customer",
		Active:     true,
		CustomerID: &customer.ID,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	token, err := auth.GenerateToken(&user)
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/protected",
		AuthMiddleware(),
		func(c *gin.Context) {
			customerID, exists := c.Get("customer_id")
			if !exists {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{"error": "customer_id missing"},
				)
				return
			}

			c.JSON(
				http.StatusOK,
				gin.H{"customer_id": customerID},
			)
		},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)
	request.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestAuthMiddlewareUsesUpdatedCustomerMapping(
	t *testing.T,
) {
	db := setupAuthMiddlewareCustomerTestDB(t)

	firstCustomer := models.Customer{
		CustomerCode: "CUS-MW-002",
		FullName:     "First Customer",
		Mobile:       "01781000002",
		Status:       "ACTIVE",
	}
	if err := db.Create(&firstCustomer).Error; err != nil {
		t.Fatal(err)
	}

	secondCustomer := models.Customer{
		CustomerCode: "CUS-MW-003",
		FullName:     "Second Customer",
		Mobile:       "01781000003",
		Status:       "ACTIVE",
	}
	if err := db.Create(&secondCustomer).Error; err != nil {
		t.Fatal(err)
	}

	user := models.User{
		Name:       "Mapped Customer",
		Username:   "mapped-customer",
		Email:      "mapped@example.com",
		Password:   "hash",
		Role:       "customer",
		Active:     true,
		CustomerID: &firstCustomer.ID,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	token, err := auth.GenerateToken(&user)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("customer_id", secondCustomer.ID).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/protected",
		AuthMiddleware(),
		func(c *gin.Context) {
			customerID := c.GetUint("customer_id")

			if customerID != secondCustomer.ID {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"error": "stale customer mapping",
					},
				)
				return
			}

			c.Status(http.StatusOK)
		},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)
	request.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected current DB mapping, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestAuthMiddlewareRejectsDisabledCustomerWithOldToken(
	t *testing.T,
) {
	db := setupAuthMiddlewareCustomerTestDB(t)

	customer := models.Customer{
		CustomerCode: "CUS-MW-004",
		FullName:     "Disabled Customer",
		Mobile:       "01781000004",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	user := models.User{
		Name:       "Disabled Customer",
		Username:   "disabled-customer",
		Email:      "disabled@example.com",
		Password:   "hash",
		Role:       "customer",
		Active:     true,
		CustomerID: &customer.ID,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	token, err := auth.GenerateToken(&user)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("active", false).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/protected",
		AuthMiddleware(),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)
	request.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected 401, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}
