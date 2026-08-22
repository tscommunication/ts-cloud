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
