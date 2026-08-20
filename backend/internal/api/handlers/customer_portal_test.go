package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
)

var customerPortalTestDBCounter atomic.Uint64

func setupCustomerPortalTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			"file:customer_portal_"+
				strconv.FormatUint(
					customerPortalTestDBCounter.Add(1),
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
	); err != nil {
		t.Fatal(err)
	}

	previous := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = previous
	})

	return db
}

func TestCustomerPortalMeUsesAuthenticatedCustomerScope(
	t *testing.T,
) {
	db := setupCustomerPortalTestDB(t)

	first := models.Customer{
		CustomerCode: "CUS-PORTAL-ME-001",
		FullName:     "Authenticated Customer",
		Mobile:       "01783000001",
		Status:       "ACTIVE",
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}

	second := models.Customer{
		CustomerCode: "CUS-PORTAL-ME-002",
		FullName:     "Other Customer",
		Mobile:       "01783000002",
		Status:       "ACTIVE",
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/customer-portal/me",
		func(c *gin.Context) {
			c.Set("customer_id", first.ID)
			c.Next()
		},
		GetCustomerPortalMe,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/customer-portal/me?customer_id="+
			strconv.FormatUint(uint64(second.ID), 10),
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response models.Customer
	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}

	if response.ID != first.ID {
		t.Fatalf(
			"expected authenticated customer id %d, got %d",
			first.ID,
			response.ID,
		)
	}

	if response.ID == second.ID {
		t.Fatalf(
			"query parameter changed customer scope to %d",
			second.ID,
		)
	}
}

func TestCustomerPortalMeRejectsUnlinkedAccount(
	t *testing.T,
) {
	setupCustomerPortalTestDB(t)

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/customer-portal/me",
		GetCustomerPortalMe,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/customer-portal/me",
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected 403, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestCustomerPortalSubscriptionUsesAuthenticatedCustomerScope(
	t *testing.T,
) {
	db := setupCustomerPortalTestDB(t)

	if err := db.AutoMigrate(
		&models.Package{},
		&models.Subscription{},
	); err != nil {
		t.Fatal(err)
	}

	firstCustomer := models.Customer{
		CustomerCode: "CUS-PORTAL-SUB-001",
		FullName:     "Subscription Customer One",
		Mobile:       "01783000011",
		Status:       "ACTIVE",
	}
	if err := db.Create(&firstCustomer).Error; err != nil {
		t.Fatal(err)
	}

	secondCustomer := models.Customer{
		CustomerCode: "CUS-PORTAL-SUB-002",
		FullName:     "Subscription Customer Two",
		Mobile:       "01783000012",
		Status:       "ACTIVE",
	}
	if err := db.Create(&secondCustomer).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-PORTAL-SUB-001",
		Name:        "Portal Package",
		Price:       500,
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	firstSubscription := models.Subscription{
		SubscriptionCode: "SUB-PORTAL-001",
		CustomerID:       firstCustomer.ID,
		PackageID:        pkg.ID,
		Status:           "ACTIVE",
	}
	if err := db.Create(&firstSubscription).Error; err != nil {
		t.Fatal(err)
	}

	secondSubscription := models.Subscription{
		SubscriptionCode: "SUB-PORTAL-002",
		CustomerID:       secondCustomer.ID,
		PackageID:        pkg.ID,
		Status:           "ACTIVE",
	}
	if err := db.Create(&secondSubscription).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/customer-portal/subscription",
		func(c *gin.Context) {
			c.Set("customer_id", firstCustomer.ID)
			c.Next()
		},
		GetCustomerPortalSubscription,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/customer-portal/subscription?customer_id="+
			strconv.FormatUint(uint64(secondCustomer.ID), 10),
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response []dto.CustomerPortalSubscriptionResponse
	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}

	if len(response) != 1 {
		t.Fatalf(
			"expected 1 subscription, got %d: %s",
			len(response),
			recorder.Body.String(),
		)
	}

	if response[0].ID != firstSubscription.ID {
		t.Fatalf(
			"expected subscription %d, got %d",
			firstSubscription.ID,
			response[0].ID,
		)
	}

	if response[0].ID == secondSubscription.ID {
		t.Fatalf(
			"response leaked another customer's subscription %d",
			secondSubscription.ID,
		)
	}
}

func TestCustomerPortalSubscriptionRejectsUnlinkedAccount(
	t *testing.T,
) {
	setupCustomerPortalTestDB(t)

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/customer-portal/subscription",
		GetCustomerPortalSubscription,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/customer-portal/subscription",
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected 403, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestCustomerPortalInvoicesUseAuthenticatedCustomerScope(
	t *testing.T,
) {
	db := setupCustomerPortalTestDB(t)

	if err := db.AutoMigrate(
		&models.Package{},
		&models.Subscription{},
		&models.Invoice{},
	); err != nil {
		t.Fatal(err)
	}

	firstCustomer := models.Customer{
		CustomerCode: "CUS-PORTAL-INV-001",
		FullName:     "Invoice Customer One",
		Mobile:       "01783000021",
		Status:       "ACTIVE",
	}
	if err := db.Create(&firstCustomer).Error; err != nil {
		t.Fatal(err)
	}

	secondCustomer := models.Customer{
		CustomerCode: "CUS-PORTAL-INV-002",
		FullName:     "Invoice Customer Two",
		Mobile:       "01783000022",
		Status:       "ACTIVE",
	}
	if err := db.Create(&secondCustomer).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-PORTAL-INV-001",
		Name:        "Portal Invoice Package",
		Price:       500,
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	firstSubscription := models.Subscription{
		SubscriptionCode: "SUB-PORTAL-INV-001",
		CustomerID:       firstCustomer.ID,
		PackageID:        pkg.ID,
		Status:           "ACTIVE",
	}
	if err := db.Create(&firstSubscription).Error; err != nil {
		t.Fatal(err)
	}

	secondSubscription := models.Subscription{
		SubscriptionCode: "SUB-PORTAL-INV-002",
		CustomerID:       secondCustomer.ID,
		PackageID:        pkg.ID,
		Status:           "ACTIVE",
	}
	if err := db.Create(&secondSubscription).Error; err != nil {
		t.Fatal(err)
	}

	firstInvoice := models.Invoice{
		InvoiceNo:      "INV-PORTAL-001",
		SubscriptionID: firstSubscription.ID,
		CustomerID:     firstCustomer.ID,
		PackageID:      pkg.ID,
		BillMonth:      8,
		BillYear:       2026,
		TotalAmount:    500,
		DueAmount:      500,
		Status:         "UNPAID",
	}
	if err := db.Create(&firstInvoice).Error; err != nil {
		t.Fatal(err)
	}

	secondInvoice := models.Invoice{
		InvoiceNo:      "INV-PORTAL-002",
		SubscriptionID: secondSubscription.ID,
		CustomerID:     secondCustomer.ID,
		PackageID:      pkg.ID,
		BillMonth:      8,
		BillYear:       2026,
		TotalAmount:    500,
		DueAmount:      500,
		Status:         "UNPAID",
	}
	if err := db.Create(&secondInvoice).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/customer-portal/invoices",
		func(c *gin.Context) {
			c.Set("customer_id", firstCustomer.ID)
			c.Next()
		},
		GetCustomerPortalInvoices,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/customer-portal/invoices?customer_id="+
			strconv.FormatUint(uint64(secondCustomer.ID), 10),
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response []dto.CustomerPortalInvoiceResponse
	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}

	if len(response) != 1 {
		t.Fatalf(
			"expected 1 invoice, got %d: %s",
			len(response),
			recorder.Body.String(),
		)
	}

	if response[0].ID != firstInvoice.ID {
		t.Fatalf(
			"expected invoice %d, got %d",
			firstInvoice.ID,
			response[0].ID,
		)
	}

	if response[0].ID == secondInvoice.ID {
		t.Fatalf(
			"response leaked another customer's invoice %d",
			secondInvoice.ID,
		)
	}
}

func TestCustomerPortalPaymentsUseAuthenticatedCustomerScope(
	t *testing.T,
) {
	db := setupCustomerPortalTestDB(t)

	if err := db.AutoMigrate(
		&models.Package{},
		&models.Subscription{},
		&models.Invoice{},
		&models.Payment{},
	); err != nil {
		t.Fatal(err)
	}

	firstCustomer := models.Customer{
		CustomerCode: "CUS-PORTAL-PAY-001",
		FullName:     "Payment Customer One",
		Mobile:       "01783000031",
		Status:       "ACTIVE",
	}
	if err := db.Create(&firstCustomer).Error; err != nil {
		t.Fatal(err)
	}

	secondCustomer := models.Customer{
		CustomerCode: "CUS-PORTAL-PAY-002",
		FullName:     "Payment Customer Two",
		Mobile:       "01783000032",
		Status:       "ACTIVE",
	}
	if err := db.Create(&secondCustomer).Error; err != nil {
		t.Fatal(err)
	}

	pkg := models.Package{
		PackageCode: "PKG-PORTAL-PAY-001",
		Name:        "Portal Payment Package",
		Price:       500,
		Status:      "ACTIVE",
	}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}

	firstSubscription := models.Subscription{
		SubscriptionCode: "SUB-PORTAL-PAY-001",
		CustomerID:       firstCustomer.ID,
		PackageID:        pkg.ID,
		Status:           "ACTIVE",
	}
	if err := db.Create(&firstSubscription).Error; err != nil {
		t.Fatal(err)
	}

	secondSubscription := models.Subscription{
		SubscriptionCode: "SUB-PORTAL-PAY-002",
		CustomerID:       secondCustomer.ID,
		PackageID:        pkg.ID,
		Status:           "ACTIVE",
	}
	if err := db.Create(&secondSubscription).Error; err != nil {
		t.Fatal(err)
	}

	firstInvoice := models.Invoice{
		InvoiceNo:      "INV-PORTAL-PAY-001",
		SubscriptionID: firstSubscription.ID,
		CustomerID:     firstCustomer.ID,
		PackageID:      pkg.ID,
		BillMonth:      8,
		BillYear:       2026,
		TotalAmount:    500,
		DueAmount:      0,
		PaidAmount:     500,
		Status:         "PAID",
	}
	if err := db.Create(&firstInvoice).Error; err != nil {
		t.Fatal(err)
	}

	secondInvoice := models.Invoice{
		InvoiceNo:      "INV-PORTAL-PAY-002",
		SubscriptionID: secondSubscription.ID,
		CustomerID:     secondCustomer.ID,
		PackageID:      pkg.ID,
		BillMonth:      8,
		BillYear:       2026,
		TotalAmount:    500,
		DueAmount:      0,
		PaidAmount:     500,
		Status:         "PAID",
	}
	if err := db.Create(&secondInvoice).Error; err != nil {
		t.Fatal(err)
	}

	firstPayment := models.Payment{
		ReceiptNo:      "RCP-PORTAL-001",
		InvoiceID:      firstInvoice.ID,
		SubscriptionID: firstSubscription.ID,
		CustomerID:     firstCustomer.ID,
		Amount:         500,
		Method:         "CASH",
		Status:         "SUCCESS",
	}
	if err := db.Create(&firstPayment).Error; err != nil {
		t.Fatal(err)
	}

	secondPayment := models.Payment{
		ReceiptNo:      "RCP-PORTAL-002",
		InvoiceID:      secondInvoice.ID,
		SubscriptionID: secondSubscription.ID,
		CustomerID:     secondCustomer.ID,
		Amount:         500,
		Method:         "CASH",
		Status:         "SUCCESS",
	}
	if err := db.Create(&secondPayment).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/customer-portal/payments",
		func(c *gin.Context) {
			c.Set("customer_id", firstCustomer.ID)
			c.Next()
		},
		GetCustomerPortalPayments,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/customer-portal/payments?customer_id="+
			strconv.FormatUint(uint64(secondCustomer.ID), 10),
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var response []dto.CustomerPortalPaymentResponse
	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}

	if len(response) != 1 {
		t.Fatalf(
			"expected 1 payment, got %d: %s",
			len(response),
			recorder.Body.String(),
		)
	}

	if response[0].ID != firstPayment.ID {
		t.Fatalf(
			"expected payment %d, got %d",
			firstPayment.ID,
			response[0].ID,
		)
	}

	if response[0].ID == secondPayment.ID {
		t.Fatalf(
			"response leaked another customer's payment %d",
			secondPayment.ID,
		)
	}
}

func TestCustomerPortalInvoicesRejectUnlinkedAccount(
	t *testing.T,
) {
	setupCustomerPortalTestDB(t)

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/customer-portal/invoices",
		GetCustomerPortalInvoices,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/customer-portal/invoices",
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected 403, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestCustomerPortalPaymentsRejectUnlinkedAccount(
	t *testing.T,
) {
	setupCustomerPortalTestDB(t)

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/customer-portal/payments",
		GetCustomerPortalPayments,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/customer-portal/payments",
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected 403, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}
