package config

import (
	"os"
	"testing"
)

func TestLoadCredentialKeyPrefersGenericKey(t *testing.T) {
	t.Setenv("APP_NAME", "TS Cloud Test")
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "8080")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("DB_TYPE", "sqlite")
	t.Setenv("DB_PATH", ":memory:")

	t.Setenv(
		"CREDENTIAL_KEY",
		"generic-credential-key-012345678901234567890123456789",
	)
	t.Setenv(
		"ROUTER_CREDENTIAL_KEY",
		"router-fallback-key-012345678901234567890123456789",
	)

	cfg := Load()

	want := os.Getenv("CREDENTIAL_KEY")

	if cfg.CredentialKey != want {
		t.Fatalf(
			"CredentialKey = %q, want generic CREDENTIAL_KEY",
			cfg.CredentialKey,
		)
	}
}

func TestLoadCredentialKeyFallsBackToRouterCredentialKey(t *testing.T) {
	t.Setenv("APP_NAME", "TS Cloud Test")
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "8080")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("DB_TYPE", "sqlite")
	t.Setenv("DB_PATH", ":memory:")

	t.Setenv("CREDENTIAL_KEY", "")
	t.Setenv(
		"ROUTER_CREDENTIAL_KEY",
		"router-fallback-key-012345678901234567890123456789",
	)

	cfg := Load()

	want := os.Getenv("ROUTER_CREDENTIAL_KEY")

	if cfg.CredentialKey != want {
		t.Fatalf(
			"CredentialKey = %q, want ROUTER_CREDENTIAL_KEY fallback",
			cfg.CredentialKey,
		)
	}
}

func TestLoadKeepsRouterCredentialKeyAvailable(t *testing.T) {
	t.Setenv("APP_NAME", "TS Cloud Test")
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "8080")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("DB_TYPE", "sqlite")
	t.Setenv("DB_PATH", ":memory:")

	t.Setenv(
		"ROUTER_CREDENTIAL_KEY",
		"legacy-router-key-0123456789012345678901234567890123",
	)

	cfg := Load()

	if cfg.RouterCredentialKey != os.Getenv("ROUTER_CREDENTIAL_KEY") {
		t.Fatalf(
			"RouterCredentialKey = %q, want legacy value preserved",
			cfg.RouterCredentialKey,
		)
	}
}
