package routes

import (
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/config"
)

func TestRegisterIncludesCodeManagementRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	cfg := &config.Config{}

	Register(router, cfg)

	const wantMethod = "PUT"
	const wantPath = "/api/v1/code-management/:entity/:id"
	const wantHandler = "github.com/tscommunication/ts-cloud/internal/api/handlers.UpdateManagedCode"

	for _, route := range router.Routes() {
		if route.Method != wantMethod || route.Path != wantPath {
			continue
		}

		if route.Handler != wantHandler {
			t.Fatalf(
				"code management route handler = %q, want %q",
				route.Handler,
				wantHandler,
			)
		}

		return
	}

	t.Fatalf(
		"missing route %s %s",
		wantMethod,
		wantPath,
	)
}

func TestRegisterIncludesLocationHierarchyRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	cfg := &config.Config{}

	Register(router, cfg)

	expected := map[string]string{
		"/api/v1/divisions/:id/districts":   "github.com/tscommunication/ts-cloud/internal/api/handlers.GetDistrictsByDivision",
		"/api/v1/districts/:id/upazilas":    "github.com/tscommunication/ts-cloud/internal/api/handlers.GetUpazilasByDistrict",
		"/api/v1/upazilas/:id/post-offices": "github.com/tscommunication/ts-cloud/internal/api/handlers.GetPostOfficesByUpazila",
	}

	found := make(map[string]bool, len(expected))

	for _, route := range router.Routes() {
		if route.Method != "GET" {
			continue
		}

		wantHandler, ok := expected[route.Path]
		if !ok {
			continue
		}

		if route.Handler != wantHandler {
			t.Fatalf(
				"location route %s handler = %q, want %q",
				route.Path,
				route.Handler,
				wantHandler,
			)
		}

		found[route.Path] = true
	}

	for path := range expected {
		if !found[path] {
			t.Fatalf("missing GET location route %s", path)
		}
	}
}
