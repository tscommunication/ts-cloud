package services

import (
	"strings"
	"testing"

	"github.com/tscommunication/ts-cloud/internal/models"
)

const subscriptionSecretTestKey = "0123456789abcdef0123456789abcdef"

func TestSubscriptionPPPoEPasswordRoundTrip(
	t *testing.T,
) {
	subscription := &models.Subscription{}

	err := SetSubscriptionPPPoEPassword(
		subscription,
		"subscriber-secret",
		subscriptionSecretTestKey,
	)
	if err != nil {
		t.Fatalf(
			"set subscription PPPoE password: %v",
			err,
		)
	}

	if subscription.PPPoEPassword != "" {
		t.Fatalf(
			"expected plaintext password to be cleared, got length %d",
			len(subscription.PPPoEPassword),
		)
	}

	if strings.TrimSpace(
		subscription.PPPoEPasswordEncrypted,
	) == "" {
		t.Fatal(
			"expected encrypted PPPoE password",
		)
	}

	password, err := GetSubscriptionPPPoEPassword(
		subscription,
		subscriptionSecretTestKey,
	)
	if err != nil {
		t.Fatalf(
			"get subscription PPPoE password: %v",
			err,
		)
	}

	if password != "subscriber-secret" {
		t.Fatalf(
			"unexpected decrypted password %q",
			password,
		)
	}
}

func TestGetSubscriptionPPPoEPasswordRejectsWrongKey(
	t *testing.T,
) {
	subscription := &models.Subscription{}

	if err := SetSubscriptionPPPoEPassword(
		subscription,
		"subscriber-secret",
		subscriptionSecretTestKey,
	); err != nil {
		t.Fatalf(
			"prepare encrypted PPPoE password: %v",
			err,
		)
	}

	_, err := GetSubscriptionPPPoEPassword(
		subscription,
		"abcdef0123456789abcdef0123456789",
	)
	if err == nil {
		t.Fatal(
			"expected wrong credential key to fail",
		)
	}

	if !strings.Contains(
		err.Error(),
		"decrypt subscription PPPoE credential",
	) {
		t.Fatalf(
			"unexpected wrong-key error: %v",
			err,
		)
	}
}

func TestGetSubscriptionPPPoEPasswordRejectsMissingCredential(
	t *testing.T,
) {
	subscription := &models.Subscription{}

	_, err := GetSubscriptionPPPoEPassword(
		subscription,
		subscriptionSecretTestKey,
	)
	if err == nil {
		t.Fatal(
			"expected missing PPPoE credential to fail",
		)
	}

	expected :=
		"subscription PPPoE credential is not configured"

	if err.Error() != expected {
		t.Fatalf(
			"unexpected error %q, want %q",
			err.Error(),
			expected,
		)
	}
}

func TestGetSubscriptionPPPoEPasswordRejectsNilSubscription(
	t *testing.T,
) {
	_, err := GetSubscriptionPPPoEPassword(
		nil,
		subscriptionSecretTestKey,
	)
	if err == nil {
		t.Fatal(
			"expected nil subscription to fail",
		)
	}

	if err.Error() != "subscription is required" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestCreateSubscriptionRejectsPlaintextPPPoEPassword(
	t *testing.T,
) {
	subscription := &models.Subscription{
		PPPoEPassword: "plaintext-secret",
	}

	err := CreateSubscription(subscription)
	if err == nil {
		t.Fatal(
			"expected plaintext PPPoE password to be rejected",
		)
	}

	expected :=
		"plaintext PPPoE password must be encrypted before saving subscription"

	if err.Error() != expected {
		t.Fatalf(
			"unexpected error %q, want %q",
			err.Error(),
			expected,
		)
	}
}

func TestUpdateSubscriptionRejectsPlaintextPPPoEPassword(
	t *testing.T,
) {
	subscription := &models.Subscription{
		PPPoEPassword: "plaintext-secret",
	}

	err := UpdateSubscription(subscription)
	if err == nil {
		t.Fatal(
			"expected plaintext PPPoE password to be rejected",
		)
	}

	expected :=
		"plaintext PPPoE password must be encrypted before saving subscription"

	if err.Error() != expected {
		t.Fatalf(
			"unexpected error %q, want %q",
			err.Error(),
			expected,
		)
	}
}

func TestSubscriptionSecretHelpersRejectNilSubscription(
	t *testing.T,
) {
	if err := SetSubscriptionPPPoEPassword(
		nil,
		"secret",
		subscriptionSecretTestKey,
	); err == nil {
		t.Fatal(
			"expected setter to reject nil subscription",
		)
	}

	if err := CreateSubscription(nil); err == nil {
		t.Fatal(
			"expected create to reject nil subscription",
		)
	}

	if err := UpdateSubscription(nil); err == nil {
		t.Fatal(
			"expected update to reject nil subscription",
		)
	}
}

func TestSetSubscriptionPPPoEPasswordBlankPreservesCredential(
	t *testing.T,
) {
	subscription := &models.Subscription{}

	if err := SetSubscriptionPPPoEPassword(
		subscription,
		"existing-secret",
		subscriptionSecretTestKey,
	); err != nil {
		t.Fatalf(
			"prepare encrypted credential: %v",
			err,
		)
	}

	originalCiphertext :=
		subscription.PPPoEPasswordEncrypted

	if err := SetSubscriptionPPPoEPassword(
		subscription,
		"   ",
		subscriptionSecretTestKey,
	); err != nil {
		t.Fatalf(
			"blank password update: %v",
			err,
		)
	}

	if subscription.PPPoEPasswordEncrypted !=
		originalCiphertext {
		t.Fatal(
			"blank password unexpectedly replaced encrypted credential",
		)
	}
}
