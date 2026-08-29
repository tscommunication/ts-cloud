package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/tscommunication/ts-cloud/internal/mikrotik"
	"github.com/tscommunication/ts-cloud/internal/models"
)

func TestReconcileSubscriptionPPPSecretCreateEndToEnd(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	reader := &fakePPPSecretReader{
		Rows: nil,
	}

	writer := &fakePPPSecretWriter{
		AddID: "*900",
	}

	result, err := ReconcileSubscriptionPPPSecret(
		subscription.ID,
		reconciliationPlanTestKey,
		reader,
		writer,
	)
	if err != nil {
		t.Fatalf(
			"reconcile CREATE: %v",
			err,
		)
	}

	if result.Plan.Action !=
		PPPSecretActionCreate {
		t.Fatalf(
			"plan action = %q",
			result.Plan.Action,
		)
	}

	if !result.Execution.Executed {
		t.Fatal(
			"CREATE execution was not performed",
		)
	}

	if result.Execution.SecretID != "*900" {
		t.Fatalf(
			"secret id = %q",
			result.Execution.SecretID,
		)
	}

	if reader.Calls != 1 {
		t.Fatalf(
			"reader calls = %d, want 1",
			reader.Calls,
		)
	}

	if writer.AddCalls != 1 {
		t.Fatalf(
			"writer add calls = %d, want 1",
			writer.AddCalls,
		)
	}

	if writer.RouterID != router.ID {
		t.Fatalf(
			"writer router id = %d, want %d",
			writer.RouterID,
			router.ID,
		)
	}

	if writer.Input.Password !=
		"subscriber-secret" {
		t.Fatal(
			"CREATE did not receive subscriber credential",
		)
	}
}

func TestReconcileSubscriptionPPPSecretNoopDoesNotMutate(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, _, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	reader := &fakePPPSecretReader{
		Rows: []mikrotik.PPPSecret{
			{
				ID:       "*100",
				Name:     "subscriber-1",
				Service:  "pppoe",
				Profile:  "Go_P25",
				Disabled: false,
			},
		},
	}

	writer := &fakePPPSecretWriter{}

	result, err := ReconcileSubscriptionPPPSecret(
		subscription.ID,
		reconciliationPlanTestKey,
		reader,
		writer,
	)
	if err != nil {
		t.Fatalf(
			"reconcile NOOP: %v",
			err,
		)
	}

	if result.Plan.Action !=
		PPPSecretActionNoop {
		t.Fatalf(
			"plan action = %q",
			result.Plan.Action,
		)
	}

	if result.Execution.Executed {
		t.Fatal(
			"NOOP unexpectedly executed mutation",
		)
	}

	if writer.AddCalls+
		writer.SetCalls+
		writer.EnableCalls+
		writer.DisableCalls != 0 {
		t.Fatal(
			"NOOP unexpectedly called writer",
		)
	}
}

func TestReconcileSubscriptionPPPSecretDisableWithoutSubscriberCredential(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, _, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"SUSPENDED",
		)

	if err := db.Exec(
		`UPDATE subscriptions
		 SET pp_po_e_password_encrypted = ''
		 WHERE id = ?`,
		subscription.ID,
	).Error; err != nil {
		t.Fatalf(
			"clear subscriber credential: %v",
			err,
		)
	}

	reader := &fakePPPSecretReader{
		Rows: []mikrotik.PPPSecret{
			{
				ID:       "*200",
				Name:     "subscriber-1",
				Service:  "pppoe",
				Profile:  "Go_P25",
				Disabled: false,
			},
		},
	}

	writer := &fakePPPSecretWriter{}

	result, err := ReconcileSubscriptionPPPSecret(
		subscription.ID,
		reconciliationPlanTestKey,
		reader,
		writer,
	)
	if err != nil {
		t.Fatalf(
			"reconcile DISABLE without subscriber credential: %v",
			err,
		)
	}

	if result.Plan.Action !=
		PPPSecretActionDisable {
		t.Fatalf(
			"plan action = %q",
			result.Plan.Action,
		)
	}

	if writer.DisableCalls != 1 {
		t.Fatalf(
			"disable calls = %d, want 1",
			writer.DisableCalls,
		)
	}

	if writer.SetCalls != 0 ||
		writer.AddCalls != 0 {
		t.Fatal(
			"DISABLE unexpectedly used password-bearing mutation",
		)
	}
}

func TestReconcileSubscriptionPPPSecretUpdateProfile(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, _, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	reader := &fakePPPSecretReader{
		Rows: []mikrotik.PPPSecret{
			{
				ID:       "*300",
				Name:     "subscriber-1",
				Service:  "pppoe",
				Profile:  "Old_Profile",
				Disabled: false,
			},
		},
	}

	writer := &fakePPPSecretWriter{}

	result, err := ReconcileSubscriptionPPPSecret(
		subscription.ID,
		reconciliationPlanTestKey,
		reader,
		writer,
	)
	if err != nil {
		t.Fatalf(
			"reconcile UPDATE: %v",
			err,
		)
	}

	if result.Plan.Action !=
		PPPSecretActionUpdate {
		t.Fatalf(
			"plan action = %q",
			result.Plan.Action,
		)
	}

	if writer.SetCalls != 1 {
		t.Fatalf(
			"set calls = %d, want 1",
			writer.SetCalls,
		)
	}

	if writer.ID != "*300" {
		t.Fatalf(
			"set secret id = %q",
			writer.ID,
		)
	}

	if writer.Input.Profile != "Go_P25" {
		t.Fatalf(
			"profile = %q",
			writer.Input.Profile,
		)
	}
}

func TestReconcileSubscriptionPPPSecretConflictDoesNotMutate(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, _, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	reader := &fakePPPSecretReader{
		Rows: []mikrotik.PPPSecret{
			{
				ID:   "*1",
				Name: "subscriber-1",
			},
			{
				ID:   "*2",
				Name: "subscriber-1",
			},
		},
	}

	writer := &fakePPPSecretWriter{}

	result, err := ReconcileSubscriptionPPPSecret(
		subscription.ID,
		reconciliationPlanTestKey,
		reader,
		writer,
	)
	if err == nil {
		t.Fatal(
			"expected reconciliation conflict",
		)
	}

	if result.Plan.Action !=
		PPPSecretActionConflict {
		t.Fatalf(
			"plan action = %q",
			result.Plan.Action,
		)
	}

	if writer.AddCalls+
		writer.SetCalls+
		writer.EnableCalls+
		writer.DisableCalls != 0 {
		t.Fatal(
			"conflict unexpectedly mutated router",
		)
	}

	if !strings.Contains(
		err.Error(),
		"reconciliation conflict",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestReconcileSubscriptionPPPSecretReaderFailureStopsBeforeMutation(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, _, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	reader := &fakePPPSecretReader{
		Err: errors.New(
			"RouterOS read unavailable",
		),
	}

	writer := &fakePPPSecretWriter{}

	_, err := ReconcileSubscriptionPPPSecret(
		subscription.ID,
		reconciliationPlanTestKey,
		reader,
		writer,
	)
	if err == nil {
		t.Fatal(
			"expected reader failure",
		)
	}

	if writer.AddCalls+
		writer.SetCalls+
		writer.EnableCalls+
		writer.DisableCalls != 0 {
		t.Fatal(
			"reader failure unexpectedly mutated router",
		)
	}
}

func TestReconcileSubscriptionPPPSecretValidation(
	t *testing.T,
) {
	tests := []struct {
		name   string
		id     uint
		reader PPPSecretReader
		writer PPPSecretWriter
		want   string
	}{
		{
			name:   "missing subscription id",
			reader: &fakePPPSecretReader{},
			writer: &fakePPPSecretWriter{},
			want:   "subscription id is required",
		},
		{
			name:   "missing reader",
			id:     1,
			writer: &fakePPPSecretWriter{},
			want:   "PPP secret reader is required",
		},
		{
			name:   "missing writer",
			id:     1,
			reader: &fakePPPSecretReader{},
			want:   "PPP secret writer is required",
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				_, err :=
					ReconcileSubscriptionPPPSecret(
						test.id,
						reconciliationPlanTestKey,
						test.reader,
						test.writer,
					)

				if err == nil {
					t.Fatal(
						"expected validation failure",
					)
				}

				if err.Error() != test.want {
					t.Fatalf(
						"error = %q, want %q",
						err.Error(),
						test.want,
					)
				}
			},
		)
	}
}

func TestPPPMigrationRequiresOldSecretCleanup(t *testing.T) {
	db := setupPPPReconciliationPlanDB(t)

	_, oldRouter, _ := createPPPReconciliationPlanFixture(
		t,
		db,
		"ACTIVE",
	)

	reader := &fakePPPSecretReader{
		Rows: []mikrotik.PPPSecret{
			{
				ID:       "*OLD",
				Name:     "subscriber-old",
				Service:  "pppoe",
				Disabled: false,
			},
		},
	}
	writer := &fakePPPSecretWriter{}

	err := DisableMigratedPPPSecret(
		oldRouter.ID,
		"subscriber-old",
		oldRouter.ID+1,
		"subscriber-new",
		reconciliationPlanTestKey,
		reader,
		writer,
	)
	if err != nil {
		t.Fatalf("disable migrated PPP secret: %v", err)
	}

	if writer.DisableCalls != 1 {
		t.Fatalf(
			"disable calls = %d, want 1",
			writer.DisableCalls,
		)
	}
	if writer.RouterID != oldRouter.ID {
		t.Fatalf(
			"cleanup router id = %d, want %d",
			writer.RouterID,
			oldRouter.ID,
		)
	}
	if writer.ID != "*OLD" {
		t.Fatalf(
			"cleanup secret id = %q, want *OLD",
			writer.ID,
		)
	}
}

func TestDisableMigratedPPPSecretSameIdentityNoop(t *testing.T) {
	db := setupPPPReconciliationPlanDB(t)

	_, router, _ := createPPPReconciliationPlanFixture(
		t,
		db,
		"ACTIVE",
	)

	reader := &fakePPPSecretReader{}
	writer := &fakePPPSecretWriter{}

	err := DisableMigratedPPPSecret(
		router.ID,
		"subscriber-1",
		router.ID,
		"subscriber-1",
		reconciliationPlanTestKey,
		reader,
		writer,
	)
	if err != nil {
		t.Fatalf("same identity cleanup: %v", err)
	}

	if reader.Calls != 0 {
		t.Fatalf("reader calls = %d, want 0", reader.Calls)
	}
	if writer.DisableCalls != 0 {
		t.Fatalf("disable calls = %d, want 0", writer.DisableCalls)
	}
}

func TestDisableMigratedPPPSecretAlreadyDisabledNoop(t *testing.T) {
	db := setupPPPReconciliationPlanDB(t)

	_, router, _ := createPPPReconciliationPlanFixture(
		t,
		db,
		"ACTIVE",
	)

	reader := &fakePPPSecretReader{
		Rows: []mikrotik.PPPSecret{
			{
				ID:       "*OLD",
				Name:     "subscriber-old",
				Service:  "pppoe",
				Disabled: true,
			},
		},
	}
	writer := &fakePPPSecretWriter{}

	err := DisableMigratedPPPSecret(
		router.ID,
		"subscriber-old",
		router.ID+1,
		"subscriber-new",
		reconciliationPlanTestKey,
		reader,
		writer,
	)
	if err != nil {
		t.Fatalf("already disabled cleanup: %v", err)
	}

	if writer.DisableCalls != 0 {
		t.Fatalf("disable calls = %d, want 0", writer.DisableCalls)
	}
}

func TestPPPMigrationTargetFailureLeavesOldSecretUntouched(t *testing.T) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ := createPPPReconciliationPlanFixture(
		t,
		db,
		"ACTIVE",
	)

	if err := db.Model(&models.Subscription{}).
		Where("id = ?", subscription.ID).
		Update("pp_po_e_username", "subscriber-new").Error; err != nil {
		t.Fatalf("update subscription username: %v", err)
	}

	reader := &fakePPPSecretReader{
		Rows: []mikrotik.PPPSecret{
			{
				ID:       "*OLD",
				Name:     "subscriber-1",
				Service:  "pppoe",
				Disabled: false,
			},
		},
	}

	writer := &fakePPPSecretWriter{
		Err: errors.New("target write failed"),
	}

	_, err := ReconcileSubscriptionPPPMigration(
		subscription.ID,
		router.ID,
		"subscriber-1",
		reconciliationPlanTestKey,
		reader,
		writer,
	)
	if err == nil {
		t.Fatal("expected target reconciliation failure")
	}

	if writer.AddCalls != 1 {
		t.Fatalf("target add calls = %d, want 1", writer.AddCalls)
	}
	if writer.DisableCalls != 0 {
		t.Fatalf(
			"old secret disable calls = %d, want 0",
			writer.DisableCalls,
		)
	}
}

type sequentialPPPSecretReader struct {
	RowsByCall [][]mikrotik.PPPSecret
	Calls      int
}

func (reader *sequentialPPPSecretReader) ListPPPSecrets(
	router *models.NetworkRouter,
	name string,
	keyMaterial string,
) ([]mikrotik.PPPSecret, error) {
	reader.Calls++

	index := reader.Calls - 1
	if index < 0 || index >= len(reader.RowsByCall) {
		return nil, nil
	}

	return reader.RowsByCall[index], nil
}

func TestPPPMigrationSuccessDisablesOldSecret(t *testing.T) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ := createPPPReconciliationPlanFixture(
		t,
		db,
		"ACTIVE",
	)

	if err := db.Model(&models.Subscription{}).
		Where("id = ?", subscription.ID).
		Update("pp_po_e_username", "subscriber-new").Error; err != nil {
		t.Fatalf("update subscription username: %v", err)
	}

	reader := &sequentialPPPSecretReader{
		RowsByCall: [][]mikrotik.PPPSecret{
			nil,
			{
				{
					ID:       "*OLD",
					Name:     "subscriber-1",
					Service:  "pppoe",
					Disabled: false,
				},
			},
		},
	}
	writer := &fakePPPSecretWriter{
		AddID: "*NEW",
	}

	result, err := ReconcileSubscriptionPPPMigration(
		subscription.ID,
		router.ID,
		"subscriber-1",
		reconciliationPlanTestKey,
		reader,
		writer,
	)
	if err != nil {
		t.Fatalf("reconcile migration: %v", err)
	}

	if result.Execution.SecretID != "*NEW" {
		t.Fatalf(
			"target secret id = %q, want *NEW",
			result.Execution.SecretID,
		)
	}

	if writer.AddCalls != 1 {
		t.Fatalf("target add calls = %d, want 1", writer.AddCalls)
	}

	if writer.DisableCalls != 1 {
		t.Fatalf(
			"old secret disable calls = %d, want 1",
			writer.DisableCalls,
		)
	}
}
