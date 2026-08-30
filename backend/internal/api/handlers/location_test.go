package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tscommunication/ts-cloud/internal/database"
	"github.com/tscommunication/ts-cloud/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var locationHandlerTestDBCounter atomic.Uint64

func setupLocationHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(
			"file:location_handler_"+
				strconv.FormatUint(uint64(locationHandlerTestDBCounter.Add(1)), 10)+
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
		&models.Division{},
		&models.District{},
		&models.Upazila{},
		&models.PostOffice{},
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

func seedLocationHandlerHierarchy(t *testing.T, db *gorm.DB) (
	models.Division,
	models.District,
	models.Upazila,
	models.PostOffice,
) {
	t.Helper()

	division := models.Division{
		Name: "Dhaka",
	}
	if err := db.Create(&division).Error; err != nil {
		t.Fatal(err)
	}

	district := models.District{
		DivisionID: division.ID,
		Name:       "Dhaka",
	}
	if err := db.Create(&district).Error; err != nil {
		t.Fatal(err)
	}

	upazila := models.Upazila{
		DistrictID: district.ID,
		Name:       "Dhamrai",
	}
	if err := db.Create(&upazila).Error; err != nil {
		t.Fatal(err)
	}

	postOffice := models.PostOffice{
		UpazilaID:  upazila.ID,
		Name:       "Dhamrai",
		PostalCode: "1350",
	}
	if err := db.Create(&postOffice).Error; err != nil {
		t.Fatal(err)
	}

	return division, district, upazila, postOffice
}

func performLocationHandlerRequest(
	handler gin.HandlerFunc,
	path string,
	paramValue string,
) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	if paramValue == "" {
		router.GET(path, handler)
	} else {
		router.GET(path, handler)
	}

	requestPath := path
	if paramValue != "" {
		requestPath = "/test/" + paramValue
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		requestPath,
		nil,
	)

	router.ServeHTTP(recorder, request)

	return recorder
}

func TestGetDivisions(t *testing.T) {
	db := setupLocationHandlerTestDB(t)
	division, _, _, _ := seedLocationHandlerHierarchy(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", GetDivisions)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var rows []models.Division
	if err := json.Unmarshal(recorder.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}

	if len(rows) != 1 || rows[0].ID != division.ID {
		t.Fatalf("unexpected divisions: %+v", rows)
	}
}

func TestGetDistrictsByDivision(t *testing.T) {
	db := setupLocationHandlerTestDB(t)
	division, district, _, _ := seedLocationHandlerHierarchy(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test/:id", GetDistrictsByDivision)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/test/"+uintString(division.ID),
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var rows []models.District
	if err := json.Unmarshal(recorder.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}

	if len(rows) != 1 ||
		rows[0].ID != district.ID ||
		rows[0].DivisionID != division.ID {
		t.Fatalf("unexpected districts: %+v", rows)
	}
}

func TestGetUpazilasByDistrict(t *testing.T) {
	db := setupLocationHandlerTestDB(t)
	_, district, upazila, _ := seedLocationHandlerHierarchy(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test/:id", GetUpazilasByDistrict)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/test/"+uintString(district.ID),
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var rows []models.Upazila
	if err := json.Unmarshal(recorder.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}

	if len(rows) != 1 ||
		rows[0].ID != upazila.ID ||
		rows[0].DistrictID != district.ID {
		t.Fatalf("unexpected upazilas: %+v", rows)
	}
}

func TestGetPostOfficesByDistrict(t *testing.T) {
	db := setupLocationHandlerTestDB(t)
	_, district, upazila, postOffice := seedLocationHandlerHierarchy(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test/:id", GetPostOfficesByDistrict)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/test/"+uintString(district.ID),
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var rows []models.PostOffice
	if err := json.Unmarshal(recorder.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}

	if len(rows) != 1 ||
		rows[0].ID != postOffice.ID ||
		rows[0].UpazilaID != upazila.ID ||
		rows[0].PostalCode != "1350" {
		t.Fatalf("unexpected district post offices: %+v", rows)
	}
}

func TestGetPostOfficesByUpazila(t *testing.T) {
	db := setupLocationHandlerTestDB(t)
	_, _, upazila, postOffice := seedLocationHandlerHierarchy(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test/:id", GetPostOfficesByUpazila)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/test/"+uintString(upazila.ID),
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var rows []models.PostOffice
	if err := json.Unmarshal(recorder.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}

	if len(rows) != 1 ||
		rows[0].ID != postOffice.ID ||
		rows[0].UpazilaID != upazila.ID ||
		rows[0].PostalCode != "1350" {
		t.Fatalf("unexpected post offices: %+v", rows)
	}
}

func TestLocationHandlerRejectsInvalidParentID(t *testing.T) {
	setupLocationHandlerTestDB(t)

	tests := []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{
			name:    "districts",
			handler: GetDistrictsByDivision,
		},
		{
			name:    "upazilas",
			handler: GetUpazilasByDistrict,
		},
		{
			name:    "post offices",
			handler: GetPostOfficesByUpazila,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/test/:id", test.handler)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodGet,
				"/test/not-a-number",
				nil,
			)

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status %d, got %d: %s",
					http.StatusBadRequest,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func uintString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
