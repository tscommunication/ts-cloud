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
	"github.com/tscommunication/ts-cloud/internal/middleware"
	"github.com/tscommunication/ts-cloud/internal/models"
)

var codeManagementTestDBCounter atomic.Uint64

func setupCodeManagementTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			"file:code_management_"+
				strconv.FormatUint(codeManagementTestDBCounter.Add(1), 10)+
				"?mode=memory&cache=shared",
		),
		&gorm.Config{DisableForeignKeyConstraintWhenMigrating: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(
		&models.POP{},
		&models.Agent{},
		&models.Package{},
		&models.CodeChangeAudit{},
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

func performCodeManagementRequest(
	t *testing.T,
	role string,
	userID uint,
	entity string,
	id uint,
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
		"/code-management/:entity/:id",
		func(c *gin.Context) {
			c.Set("role", role)
			c.Set("user_id", userID)
			c.Next()
		},
		middleware.RequireRoles("superadmin"),
		UpdateManagedCode,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/code-management/"+entity+"/"+strconv.FormatUint(uint64(id), 10),
		bytes.NewReader(payload),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)
	return recorder
}

func createCodeManagementPOP(t *testing.T, db *gorm.DB, code string) models.POP {
	t.Helper()

	row := models.POP{
		Code:   code,
		Name:   "Test POP " + code,
		Status: "ACTIVE",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	return row
}

func createCodeManagementAgent(
	t *testing.T,
	db *gorm.DB,
	code string,
	popID uint,
) models.Agent {
	t.Helper()

	row := models.Agent{
		Code:   code,
		Name:   "Test Agent " + code,
		POPID:  popID,
		Status: "ACTIVE",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	return row
}

func createCodeManagementPackage(
	t *testing.T,
	db *gorm.DB,
	code string,
) models.Package {
	t.Helper()

	row := models.Package{
		PackageCode:  code,
		Name:         "Test Package " + code,
		Price:        500,
		ValidityDays: 30,
		Status:       "ACTIVE",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	return row
}

func TestUpdateManagedCodeAgentCreatesAudit(t *testing.T) {
	db := setupCodeManagementTestDB(t)

	pop := createCodeManagementPOP(t, db, "POP-CM-01")
	agent := createCodeManagementAgent(t, db, "AGENT-CM-01", pop.ID)

	response := performCodeManagementRequest(
		t,
		"superadmin",
		999,
		"agent",
		agent.ID,
		map[string]any{
			"code":   " agent-new-01 ",
			"reason": " Correct agent code ",
		},
	)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s",
			response.Code, response.Body.String())
	}

	var stored models.Agent
	if err := db.First(&stored, agent.ID).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Code != "AGENT-NEW-01" {
		t.Fatalf("expected AGENT-NEW-01, got %q", stored.Code)
	}

	var audit models.CodeChangeAudit
	if err := db.First(&audit).Error; err != nil {
		t.Fatal(err)
	}

	if audit.EntityType != "AGENT" ||
		audit.EntityID != agent.ID ||
		audit.OldCode != "AGENT-CM-01" ||
		audit.NewCode != "AGENT-NEW-01" ||
		audit.ChangedBy != 999 ||
		audit.Reason != "Correct agent code" {
		t.Fatalf("unexpected audit: %+v", audit)
	}

	if audit.ChangedAt.IsZero() {
		t.Fatal("expected ChangedAt")
	}
}

func TestUpdateManagedCodePOPAndPackage(t *testing.T) {
	t.Run("POP", func(t *testing.T) {
		db := setupCodeManagementTestDB(t)

		row := createCodeManagementPOP(t, db, "POP-CM-02")

		response := performCodeManagementRequest(
			t, "superadmin", 777, "pop", row.ID,
			map[string]any{
				"code":   "POP-NEW-02",
				"reason": "POP regression test",
			},
		)

		if response.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s",
				response.Code, response.Body.String())
		}

		var stored models.POP
		if err := db.First(&stored, row.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.Code != "POP-NEW-02" {
			t.Fatalf("unexpected POP code %q", stored.Code)
		}
	})

	t.Run("PACKAGE", func(t *testing.T) {
		db := setupCodeManagementTestDB(t)

		row := createCodeManagementPackage(t, db, "PKG-CM-02")

		response := performCodeManagementRequest(
			t, "superadmin", 777, "package", row.ID,
			map[string]any{
				"code":   "PKG-NEW-02",
				"reason": "Package regression test",
			},
		)

		if response.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s",
				response.Code, response.Body.String())
		}

		var stored models.Package
		if err := db.First(&stored, row.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.PackageCode != "PKG-NEW-02" {
			t.Fatalf("unexpected package code %q", stored.PackageCode)
		}
	})
}

func TestUpdateManagedCodeValidation(t *testing.T) {
	db := setupCodeManagementTestDB(t)
	pop := createCodeManagementPOP(t, db, "POP-CM-03")

	tests := []struct {
		name string
		body map[string]any
		want string
	}{
		{
			name: "invalid code",
			body: map[string]any{
				"code":   "bad code!",
				"reason": "Validation test",
			},
			want: "Code must be 2-30 characters",
		},
		{
			name: "blank reason",
			body: map[string]any{
				"code":   "POP-NEW-03",
				"reason": "   ",
			},
			want: "Reason is required for audit",
		},
		{
			name: "unchanged",
			body: map[string]any{
				"code":   " pop-cm-03 ",
				"reason": "Unchanged test",
			},
			want: "new code is unchanged",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := performCodeManagementRequest(
				t,
				"superadmin",
				999,
				"pop",
				pop.ID,
				tc.body,
			)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s",
					response.Code, response.Body.String())
			}

			if !strings.Contains(response.Body.String(), tc.want) {
				t.Fatalf("expected %q, got %s",
					tc.want, response.Body.String())
			}
		})
	}

	var count int64
	if err := db.Model(&models.CodeChangeAudit{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("validation failures created %d audit rows", count)
	}
}

func TestUpdateManagedCodeUnsupportedEntity(t *testing.T) {
	setupCodeManagementTestDB(t)

	response := performCodeManagementRequest(
		t,
		"superadmin",
		999,
		"customer",
		1,
		map[string]any{
			"code":   "CUS-NEW-01",
			"reason": "Unsupported entity test",
		},
	)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s",
			response.Code, response.Body.String())
	}

	if !strings.Contains(response.Body.String(), "unsupported entity type") {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}

func TestUpdateManagedCodeNotFound(t *testing.T) {
	setupCodeManagementTestDB(t)

	response := performCodeManagementRequest(
		t,
		"superadmin",
		999,
		"pop",
		999999,
		map[string]any{
			"code":   "POP-MISSING-01",
			"reason": "Missing record test",
		},
	)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s",
			response.Code, response.Body.String())
	}
}

func TestUpdateManagedCodeDuplicateRollsBack(t *testing.T) {
	db := setupCodeManagementTestDB(t)

	first := createCodeManagementPOP(t, db, "POP-DUP-01")
	second := createCodeManagementPOP(t, db, "POP-DUP-02")

	response := performCodeManagementRequest(
		t,
		"superadmin",
		999,
		"pop",
		second.ID,
		map[string]any{
			"code":   first.Code,
			"reason": "Duplicate test",
		},
	)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s",
			response.Code, response.Body.String())
	}

	if !strings.Contains(
		response.Body.String(),
		"code already exists or cannot be updated",
	) {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}

	var stored models.POP
	if err := db.First(&stored, second.ID).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Code != "POP-DUP-02" {
		t.Fatalf("failed update changed code to %q", stored.Code)
	}

	var count int64
	if err := db.Model(&models.CodeChangeAudit{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatalf("failed transaction created %d audit rows", count)
	}
}

func TestUpdateManagedCodeRejectsAdmin(t *testing.T) {
	db := setupCodeManagementTestDB(t)
	pop := createCodeManagementPOP(t, db, "POP-AUTH-01")

	response := performCodeManagementRequest(
		t,
		"admin",
		123,
		"pop",
		pop.ID,
		map[string]any{
			"code":   "POP-AUTH-02",
			"reason": "Authorization test",
		},
	)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s",
			response.Code, response.Body.String())
	}

	if !strings.Contains(response.Body.String(), "Permission denied") {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}

	var stored models.POP
	if err := db.First(&stored, pop.ID).Error; err != nil {
		t.Fatal(err)
	}

	if stored.Code != "POP-AUTH-01" {
		t.Fatalf("forbidden request changed code to %q", stored.Code)
	}

	var count int64
	if err := db.Model(&models.CodeChangeAudit{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatalf("forbidden request created %d audit rows", count)
	}
}
