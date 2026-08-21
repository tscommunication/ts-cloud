package services

import (
	"testing"

	"gorm.io/gorm"

	"github.com/tscommunication/ts-cloud/internal/mikrotik"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func reconciliationSubscription(
	status string,
) *models.Subscription {
	return &models.Subscription{
		RouterID:               7,
		PPPoEUsername:          "subscriber-1",
		PPPoEPasswordEncrypted: "encrypted-secret",
		Status:                 status,
	}
}

func reconciliationPackage() *models.Package {
	return &models.Package{
		MikroTikProfile: "Go_P25",
	}
}

func TestBuildSubscriptionPPPSecretDesiredStateActive(
	t *testing.T,
) {
	state, err := BuildSubscriptionPPPSecretDesiredState(
		reconciliationSubscription("ACTIVE"),
		reconciliationPackage(),
	)
	if err != nil {
		t.Fatalf("build desired state: %v", err)
	}

	if state.Username != "subscriber-1" {
		t.Fatalf(
			"username = %q",
			state.Username,
		)
	}

	if state.Service != "pppoe" {
		t.Fatalf(
			"service = %q",
			state.Service,
		)
	}

	if state.Profile != "Go_P25" {
		t.Fatalf(
			"profile = %q",
			state.Profile,
		)
	}

	if state.Disabled {
		t.Fatal(
			"active subscription should be enabled",
		)
	}
}

func TestBuildSubscriptionPPPSecretDesiredStateUsesCustomerInternetAccount(
	t *testing.T,
) {
	subscription := &models.Subscription{
		RouterID:               1,
		PPPoEUsername:          "legacy-user",
		PPPoEPasswordEncrypted: "legacy-secret",
		Status:                 "ACTIVE",
		InternetAccount: &models.CustomerInternetAccount{
			Model:                  gorm.Model{ID: 9},
			RouterID:               7,
			PPPoEUsername:          "customer-owned-user",
			PPPoEPasswordEncrypted: "customer-owned-secret",
			MACAddress:              "C0:A4:76:F7:F7:DD",
			StaticIPAddress:         "10.9.0.220",
		},
	}
	pkg := &models.Package{MikroTikProfile: "internet-profile"}

	desired, err := BuildSubscriptionPPPSecretDesiredState(subscription, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if desired.Username != "customer-owned-user" || subscription.RouterID != 7 ||
		subscription.PPPoEPasswordEncrypted != "customer-owned-secret" ||
		desired.CallerID != "C0:A4:76:F7:F7:DD" ||
		desired.RemoteAddress != "10.9.0.220" {
		t.Fatalf("internet account was not used: %+v", desired)
	}
}

func TestDecideSubscriptionPPPSecretUpdateBindings(t *testing.T) {
	subscription := reconciliationSubscription("ACTIVE")
	subscription.InternetAccount = &models.CustomerInternetAccount{
		Model: gorm.Model{ID: 3}, RouterID: 7,
		PPPoEUsername: "subscriber-1", PPPoEPasswordEncrypted: "encrypted-secret",
		MACAddress: "C0:A4:76:F7:F7:DD", StaticIPAddress: "10.9.0.220",
	}
	decision, err := DecideSubscriptionPPPSecretReconciliation(
		subscription,
		reconciliationPackage(),
		[]mikrotik.PPPSecret{{ID: "*1", Name: "subscriber-1", Service: "pppoe", Profile: "Go_P25"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != PPPSecretActionUpdate {
		t.Fatalf("action = %q, want UPDATE", decision.Action)
	}
}

func TestBuildSubscriptionPPPSecretDesiredStateInactiveStatusesDisable(
	t *testing.T,
) {
	statuses := []string{
		"SUSPENDED",
		"EXPIRED",
		"DISCONNECTED",
	}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			state, err :=
				BuildSubscriptionPPPSecretDesiredState(
					reconciliationSubscription(status),
					reconciliationPackage(),
				)
			if err != nil {
				t.Fatalf(
					"build desired state: %v",
					err,
				)
			}

			if !state.Disabled {
				t.Fatalf(
					"%s subscription should be disabled",
					status,
				)
			}
		})
	}
}

func TestDecideSubscriptionPPPSecretCreate(
	t *testing.T,
) {
	decision, err :=
		DecideSubscriptionPPPSecretReconciliation(
			reconciliationSubscription("ACTIVE"),
			reconciliationPackage(),
			nil,
		)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}

	if decision.Action != PPPSecretActionCreate {
		t.Fatalf(
			"action = %q, want CREATE",
			decision.Action,
		)
	}
}

func TestDecideSubscriptionPPPSecretNoop(
	t *testing.T,
) {
	decision, err :=
		DecideSubscriptionPPPSecretReconciliation(
			reconciliationSubscription("ACTIVE"),
			reconciliationPackage(),
			[]mikrotik.PPPSecret{
				{
					ID:       "*1",
					Name:     "subscriber-1",
					Service:  "pppoe",
					Profile:  "Go_P25",
					Disabled: false,
				},
			},
		)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}

	if decision.Action != PPPSecretActionNoop {
		t.Fatalf(
			"action = %q, want NOOP",
			decision.Action,
		)
	}
}

func TestDecideSubscriptionPPPSecretUpdateProfile(
	t *testing.T,
) {
	decision, err :=
		DecideSubscriptionPPPSecretReconciliation(
			reconciliationSubscription("ACTIVE"),
			reconciliationPackage(),
			[]mikrotik.PPPSecret{
				{
					ID:       "*1",
					Name:     "subscriber-1",
					Service:  "pppoe",
					Profile:  "Old_Profile",
					Disabled: false,
				},
			},
		)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}

	if decision.Action != PPPSecretActionUpdate {
		t.Fatalf(
			"action = %q, want UPDATE",
			decision.Action,
		)
	}
}

func TestDecideSubscriptionPPPSecretDisable(
	t *testing.T,
) {
	decision, err :=
		DecideSubscriptionPPPSecretReconciliation(
			reconciliationSubscription("SUSPENDED"),
			reconciliationPackage(),
			[]mikrotik.PPPSecret{
				{
					ID:       "*1",
					Name:     "subscriber-1",
					Service:  "pppoe",
					Profile:  "Go_P25",
					Disabled: false,
				},
			},
		)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}

	if decision.Action != PPPSecretActionDisable {
		t.Fatalf(
			"action = %q, want DISABLE",
			decision.Action,
		)
	}
}

func TestDecideSubscriptionPPPSecretEnable(
	t *testing.T,
) {
	decision, err :=
		DecideSubscriptionPPPSecretReconciliation(
			reconciliationSubscription("ACTIVE"),
			reconciliationPackage(),
			[]mikrotik.PPPSecret{
				{
					ID:       "*1",
					Name:     "subscriber-1",
					Service:  "pppoe",
					Profile:  "Go_P25",
					Disabled: true,
				},
			},
		)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}

	if decision.Action != PPPSecretActionEnable {
		t.Fatalf(
			"action = %q, want ENABLE",
			decision.Action,
		)
	}
}

func TestDecideSubscriptionPPPSecretConflictDuplicateUsername(
	t *testing.T,
) {
	decision, err :=
		DecideSubscriptionPPPSecretReconciliation(
			reconciliationSubscription("ACTIVE"),
			reconciliationPackage(),
			[]mikrotik.PPPSecret{
				{
					ID:   "*1",
					Name: "subscriber-1",
				},
				{
					ID:   "*2",
					Name: "SUBSCRIBER-1",
				},
			},
		)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}

	if decision.Action != PPPSecretActionConflict {
		t.Fatalf(
			"action = %q, want CONFLICT",
			decision.Action,
		)
	}
}

func TestDecideSubscriptionPPPSecretConflictMissingInternalID(
	t *testing.T,
) {
	decision, err :=
		DecideSubscriptionPPPSecretReconciliation(
			reconciliationSubscription("ACTIVE"),
			reconciliationPackage(),
			[]mikrotik.PPPSecret{
				{
					Name:     "subscriber-1",
					Service:  "pppoe",
					Profile:  "Go_P25",
					Disabled: false,
				},
			},
		)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}

	if decision.Action != PPPSecretActionConflict {
		t.Fatalf(
			"action = %q, want CONFLICT",
			decision.Action,
		)
	}
}

func TestDecideSubscriptionPPPSecretCreateRequiresCredential(
	t *testing.T,
) {
	subscription :=
		reconciliationSubscription("ACTIVE")
	subscription.PPPoEPasswordEncrypted = ""

	_, err :=
		DecideSubscriptionPPPSecretReconciliation(
			subscription,
			reconciliationPackage(),
			nil,
		)
	if err == nil {
		t.Fatal(
			"expected missing credential to fail",
		)
	}
}

func TestBuildSubscriptionPPPSecretDesiredStateValidation(
	t *testing.T,
) {
	tests := []struct {
		name         string
		subscription *models.Subscription
		pkg          *models.Package
	}{
		{
			name:         "nil subscription",
			subscription: nil,
			pkg:          reconciliationPackage(),
		},
		{
			name:         "nil package",
			subscription: reconciliationSubscription("ACTIVE"),
			pkg:          nil,
		},
		{
			name: "missing username",
			subscription: &models.Subscription{
				RouterID: 7,
				Status:   "ACTIVE",
			},
			pkg: reconciliationPackage(),
		},
		{
			name: "missing router",
			subscription: &models.Subscription{
				PPPoEUsername: "subscriber-1",
				Status:        "ACTIVE",
			},
			pkg: reconciliationPackage(),
		},
		{
			name:         "missing profile",
			subscription: reconciliationSubscription("ACTIVE"),
			pkg:          &models.Package{},
		},
		{
			name:         "unsupported status",
			subscription: reconciliationSubscription("UNKNOWN"),
			pkg:          reconciliationPackage(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err :=
				BuildSubscriptionPPPSecretDesiredState(
					test.subscription,
					test.pkg,
				)
			if err == nil {
				t.Fatal(
					"expected validation error",
				)
			}
		})
	}
}
