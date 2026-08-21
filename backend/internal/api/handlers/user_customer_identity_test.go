package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

var userCustomerIdentityTestDBCounter atomic.Uint64

func setupUserCustomerIdentityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		sqlite.Open("file:user_customer_identity_"+
			strconv.FormatUint(userCustomerIdentityTestDBCounter.Add(1), 10)+
			"?mode=memory&cache=shared"),
		&gorm.Config{DisableForeignKeyConstraintWhenMigrating: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Customer{}, &models.User{}); err != nil {
		t.Fatal(err)
	}
	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	return db
}

func performUserRequest(t *testing.T, method, path string, body map[string]any, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	routePath := path
	if method != http.MethodPost {
		routePath = "/users/:id"
	}
	router.Handle(method, routePath, func(c *gin.Context) {
		c.Set("role", "superadmin")
		c.Set("user_id", uint(999))
		c.Next()
	}, handler)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestCreateUserRejectsCustomerRole(t *testing.T) {
	setupUserCustomerIdentityTestDB(t)
	response := performUserRequest(t, http.MethodPost, "/users", map[string]any{
		"name": "Portal User", "username": "portal-user",
		"email": "portal@example.com", "password": "secure-pass",
		"role": "customer", "customer_id": 1,
	}, CreateUser)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "managed from the Customer module") {
		t.Fatalf("unexpected response %d: %s", response.Code, response.Body.String())
	}
}

func TestUpdateUserRejectsCustomerIdentityMutation(t *testing.T) {
	db := setupUserCustomerIdentityTestDB(t)
	staff := models.User{Name: "Staff", Username: "staff", Email: "staff@example.com", Password: "hash", Role: "admin", Active: true}
	if err := db.Create(&staff).Error; err != nil {
		t.Fatal(err)
	}
	path := "/users/" + strconv.FormatUint(uint64(staff.ID), 10)
	response := performUserRequest(t, http.MethodPut, path, map[string]any{"role": "customer", "customer_id": 7}, UpdateUser)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "managed from the Customer module") {
		t.Fatalf("unexpected response %d: %s", response.Code, response.Body.String())
	}
}

func TestDeleteUserRejectsCustomerIdentity(t *testing.T) {
	db := setupUserCustomerIdentityTestDB(t)
	customerID := uint(7)
	identity := models.User{
		Name: "Customer", Username: "CUS-000007", Email: "cus-000007@customer.invalid",
		Password: "hash", Role: "customer", Active: false, CustomerID: &customerID,
	}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatal(err)
	}
	path := "/users/" + strconv.FormatUint(uint64(identity.ID), 10)
	response := performUserRequest(t, http.MethodDelete, path, nil, DeleteUser)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "managed from the Customer module") {
		t.Fatalf("unexpected response %d: %s", response.Code, response.Body.String())
	}
}
