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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

var userCustomerIdentityTestDBCounter atomic.Uint64

func setupUserCustomerIdentityTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			"file:user_customer_identity_"+
				strconv.FormatUint(
					userCustomerIdentityTestDBCounter.Add(1),
					10,
				)+
				"?mode=memory&cache=shared",
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

func performCreateUserRequest(
	t *testing.T,
	body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.POST(
		"/users",
		func(c *gin.Context) {
			c.Set("role", "superadmin")
			c.Set("user_id", uint(1))
			c.Next()
		},
		CreateUser,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/users",
		bytes.NewReader(payload),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	return recorder
}

func TestCreateCustomerUserRequiresCustomerID(t *testing.T) {
	setupUserCustomerIdentityTestDB(t)

	response := performCreateUserRequest(
		t,
		map[string]any{
			"name":     "Portal User",
			"username": "portal-user",
			"email":    "portal@example.com",
			"password": "secure-pass",
			"role":     "customer",
		},
	)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected 400, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}

	if !strings.Contains(
		response.Body.String(),
		"Customer is required for customer role",
	) {
		t.Fatalf(
			"unexpected response: %s",
			response.Body.String(),
		)
	}
}

func TestCreateCustomerUserRejectsUnknownCustomer(t *testing.T) {
	setupUserCustomerIdentityTestDB(t)

	response := performCreateUserRequest(
		t,
		map[string]any{
			"name":        "Portal User",
			"username":    "portal-user",
			"email":       "portal@example.com",
			"password":    "secure-pass",
			"role":        "customer",
			"customer_id": 999,
		},
	)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected 400, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}

	if !strings.Contains(
		response.Body.String(),
		"Customer not found",
	) {
		t.Fatalf(
			"unexpected response: %s",
			response.Body.String(),
		)
	}
}

func TestCreateCustomerUserLinksCustomerAndClearsAgent(
	t *testing.T,
) {
	db := setupUserCustomerIdentityTestDB(t)

	customer := models.Customer{
		CustomerCode: "CUS-PORTAL-100",
		FullName:     "Portal Customer",
		Mobile:       "01770000100",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	response := performCreateUserRequest(
		t,
		map[string]any{
			"name":        "Portal Customer",
			"username":    "portal-customer",
			"email":       "portal-customer@example.com",
			"password":    "secure-pass",
			"role":        "customer",
			"customer_id": customer.ID,
			"agent_id":    123,
		},
	)

	if response.Code != http.StatusCreated {
		t.Fatalf(
			"expected 201, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}

	var user models.User
	if err := db.
		Where("username = ?", "portal-customer").
		First(&user).Error; err != nil {
		t.Fatal(err)
	}

	if user.CustomerID == nil ||
		*user.CustomerID != customer.ID {
		t.Fatalf(
			"expected customer_id %d, got %+v",
			customer.ID,
			user.CustomerID,
		)
	}

	if user.AgentID != nil {
		t.Fatalf(
			"expected agent_id cleared for customer role, got %v",
			*user.AgentID,
		)
	}
}

func TestCreateCustomerUserRejectsDuplicateCustomerLogin(
	t *testing.T,
) {
	db := setupUserCustomerIdentityTestDB(t)

	customer := models.Customer{
		CustomerCode: "CUS-PORTAL-101",
		FullName:     "Portal Customer",
		Mobile:       "01770000101",
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

	existing := models.User{
		Name:       "Existing Portal User",
		Username:   "existing-portal",
		Email:      "existing@example.com",
		Password:   string(hash),
		Role:       "customer",
		Active:     true,
		CustomerID: &customer.ID,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}

	response := performCreateUserRequest(
		t,
		map[string]any{
			"name":        "Second Portal User",
			"username":    "second-portal",
			"email":       "second@example.com",
			"password":    "secure-pass",
			"role":        "customer",
			"customer_id": customer.ID,
		},
	)

	if response.Code != http.StatusConflict {
		t.Fatalf(
			"expected 409, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}

	if !strings.Contains(
		response.Body.String(),
		"Customer already has a login account",
	) {
		t.Fatalf(
			"unexpected response: %s",
			response.Body.String(),
		)
	}
}

func TestCreateStaffUserClearsCustomerID(t *testing.T) {
	db := setupUserCustomerIdentityTestDB(t)

	customer := models.Customer{
		CustomerCode: "CUS-PORTAL-102",
		FullName:     "Portal Customer",
		Mobile:       "01770000102",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	response := performCreateUserRequest(
		t,
		map[string]any{
			"name":        "Staff User",
			"username":    "staff-user",
			"email":       "staff@example.com",
			"password":    "secure-pass",
			"role":        "admin",
			"customer_id": customer.ID,
		},
	)

	if response.Code != http.StatusCreated {
		t.Fatalf(
			"expected 201, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}

	var user models.User
	if err := db.
		Where("username = ?", "staff-user").
		First(&user).Error; err != nil {
		t.Fatal(err)
	}

	if user.CustomerID != nil {
		t.Fatalf(
			"expected customer_id cleared for staff role, got %v",
			*user.CustomerID,
		)
	}
}

func performUpdateUserRequest(
	t *testing.T,
	userID uint,
	actorID uint,
	actorRole string,
	body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.PUT(
		"/users/:id",
		func(c *gin.Context) {
			c.Set("user_id", actorID)
			c.Set("role", actorRole)
			c.Next()
		},
		UpdateUser,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/users/"+strconv.FormatUint(uint64(userID), 10),
		bytes.NewReader(payload),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	return recorder
}

func TestUpdateStaffToCustomerRequiresCustomerID(t *testing.T) {
	db := setupUserCustomerIdentityTestDB(t)

	user := models.User{
		Name:     "Staff User",
		Username: "staff-to-customer",
		Email:    "staff-to-customer@example.com",
		Password: "hash",
		Role:     "admin",
		Active:   true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	response := performUpdateUserRequest(
		t,
		user.ID,
		999,
		"superadmin",
		map[string]any{
			"role": "customer",
		},
	)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected 400, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}

	if !strings.Contains(
		response.Body.String(),
		"Customer is required for customer role",
	) {
		t.Fatalf(
			"unexpected response: %s",
			response.Body.String(),
		)
	}
}

func TestUpdateCustomerMappingRejectsDuplicateCustomer(
	t *testing.T,
) {
	db := setupUserCustomerIdentityTestDB(t)

	customer := models.Customer{
		CustomerCode: "CUS-UPD-001",
		FullName:     "Existing Portal Customer",
		Mobile:       "01772000001",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	existing := models.User{
		Name:       "Existing Customer User",
		Username:   "existing-customer-user",
		Email:      "existing-customer-user@example.com",
		Password:   "hash",
		Role:       "customer",
		Active:     true,
		CustomerID: &customer.ID,
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}

	target := models.User{
		Name:     "Target Staff",
		Username: "target-staff",
		Email:    "target-staff@example.com",
		Password: "hash",
		Role:     "admin",
		Active:   true,
	}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}

	response := performUpdateUserRequest(
		t,
		target.ID,
		999,
		"superadmin",
		map[string]any{
			"role":        "customer",
			"customer_id": customer.ID,
		},
	)

	if response.Code != http.StatusConflict {
		t.Fatalf(
			"expected 409, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}

	if !strings.Contains(
		response.Body.String(),
		"Customer already has a login account",
	) {
		t.Fatalf(
			"unexpected response: %s",
			response.Body.String(),
		)
	}
}

func TestUpdateCustomerToStaffClearsCustomerID(t *testing.T) {
	db := setupUserCustomerIdentityTestDB(t)

	customer := models.Customer{
		CustomerCode: "CUS-UPD-002",
		FullName:     "Customer To Staff",
		Mobile:       "01772000002",
		Status:       "ACTIVE",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	user := models.User{
		Name:       "Customer To Staff",
		Username:   "customer-to-staff",
		Email:      "customer-to-staff@example.com",
		Password:   "hash",
		Role:       "customer",
		Active:     true,
		CustomerID: &customer.ID,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	response := performUpdateUserRequest(
		t,
		user.ID,
		999,
		"superadmin",
		map[string]any{
			"role": "admin",
		},
	)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"expected 200, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}

	var updated models.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatal(err)
	}

	if updated.Role != "admin" {
		t.Fatalf(
			"expected admin role, got %q",
			updated.Role,
		)
	}

	if updated.CustomerID != nil {
		t.Fatalf(
			"expected customer_id cleared, got %v",
			*updated.CustomerID,
		)
	}

	if updated.AgentID != nil {
		t.Fatalf(
			"expected agent_id cleared, got %v",
			*updated.AgentID,
		)
	}
}

func TestUpdateCustomerMappingRequiresSuperadmin(t *testing.T) {
	db := setupUserCustomerIdentityTestDB(t)

	firstCustomer := models.Customer{
		CustomerCode: "CUS-UPD-003",
		FullName:     "First Customer",
		Mobile:       "01772000003",
		Status:       "ACTIVE",
	}
	if err := db.Create(&firstCustomer).Error; err != nil {
		t.Fatal(err)
	}

	secondCustomer := models.Customer{
		CustomerCode: "CUS-UPD-004",
		FullName:     "Second Customer",
		Mobile:       "01772000004",
		Status:       "ACTIVE",
	}
	if err := db.Create(&secondCustomer).Error; err != nil {
		t.Fatal(err)
	}

	user := models.User{
		Name:       "Mapped Customer",
		Username:   "mapped-customer-update",
		Email:      "mapped-customer-update@example.com",
		Password:   "hash",
		Role:       "customer",
		Active:     true,
		CustomerID: &firstCustomer.ID,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	response := performUpdateUserRequest(
		t,
		user.ID,
		999,
		"admin",
		map[string]any{
			"customer_id": secondCustomer.ID,
		},
	)

	if response.Code != http.StatusForbidden {
		t.Fatalf(
			"expected 403, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}

	var unchanged models.User
	if err := db.First(&unchanged, user.ID).Error; err != nil {
		t.Fatal(err)
	}

	if unchanged.CustomerID == nil ||
		*unchanged.CustomerID != firstCustomer.ID {
		t.Fatal("customer mapping changed without superadmin permission")
	}
}

func TestUpdateCustomerMappingAllowsSuperadmin(t *testing.T) {
	db := setupUserCustomerIdentityTestDB(t)

	firstCustomer := models.Customer{
		CustomerCode: "CUS-UPD-005",
		FullName:     "First Mapping",
		Mobile:       "01772000005",
		Status:       "ACTIVE",
	}
	if err := db.Create(&firstCustomer).Error; err != nil {
		t.Fatal(err)
	}

	secondCustomer := models.Customer{
		CustomerCode: "CUS-UPD-006",
		FullName:     "Second Mapping",
		Mobile:       "01772000006",
		Status:       "ACTIVE",
	}
	if err := db.Create(&secondCustomer).Error; err != nil {
		t.Fatal(err)
	}

	user := models.User{
		Name:       "Customer Remap",
		Username:   "customer-remap",
		Email:      "customer-remap@example.com",
		Password:   "hash",
		Role:       "customer",
		Active:     true,
		CustomerID: &firstCustomer.ID,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	response := performUpdateUserRequest(
		t,
		user.ID,
		999,
		"superadmin",
		map[string]any{
			"customer_id": secondCustomer.ID,
		},
	)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"expected 200, got %d: %s",
			response.Code,
			response.Body.String(),
		)
	}

	var updated models.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatal(err)
	}

	if updated.CustomerID == nil ||
		*updated.CustomerID != secondCustomer.ID {
		t.Fatalf(
			"expected customer_id %d, got %+v",
			secondCustomer.ID,
			updated.CustomerID,
		)
	}
}
