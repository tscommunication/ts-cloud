package security

import "testing"

func TestSecretRoundTrip(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef"
	encrypted, err := EncryptSecret("router-password", key)
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "router-password" {
		t.Fatal("secret was stored as plaintext")
	}
	decrypted, err := DecryptSecret(encrypted, key)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "router-password" {
		t.Fatalf("unexpected decrypted value %q", decrypted)
	}
}

func TestSecretRejectsWrongKey(t *testing.T) {
	encrypted, err := EncryptSecret("router-password", "0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptSecret(encrypted, "abcdef0123456789abcdef0123456789"); err == nil {
		t.Fatal("expected wrong key to fail")
	}
}
