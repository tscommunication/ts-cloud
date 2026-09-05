package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/tscommunication/ts-cloud/internal/mikrotik"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type fakePPPSecretWriter struct {
	AddCalls     int
	SetCalls     int
	EnableCalls  int
	DisableCalls int

	RouterID uint
	ID       string
	Input    mikrotik.PPPSecretInput
	Key      string

	AddID string
	Err   error
}

type disconnectingPPPSecretWriter struct {
	fakePPPSecretWriter
	DisconnectCalls  int
	DisconnectedUser string
	DisconnectErr    error
}

func (writer *disconnectingPPPSecretWriter) DisconnectPPPActiveSessions(
	router *models.NetworkRouter,
	username string,
	keyMaterial string,
) error {
	writer.DisconnectCalls++
	writer.DisconnectedUser = username
	if router != nil {
		writer.RouterID = router.ID
	}
	writer.Key = keyMaterial
	return writer.DisconnectErr
}

func (writer *fakePPPSecretWriter) AddPPPSecret(
	router *models.NetworkRouter,
	input mikrotik.PPPSecretInput,
	keyMaterial string,
) (string, error) {
	writer.AddCalls++
	writer.Input = input
	writer.Key = keyMaterial

	if router != nil {
		writer.RouterID = router.ID
	}

	if writer.Err != nil {
		return "", writer.Err
	}

	if writer.AddID == "" {
		return "*NEW", nil
	}

	return writer.AddID, nil
}

func (writer *fakePPPSecretWriter) SetPPPSecret(
	router *models.NetworkRouter,
	id string,
	input mikrotik.PPPSecretInput,
	keyMaterial string,
) error {
	writer.SetCalls++
	writer.ID = id
	writer.Input = input
	writer.Key = keyMaterial

	if router != nil {
		writer.RouterID = router.ID
	}

	return writer.Err
}

func (writer *fakePPPSecretWriter) EnablePPPSecret(
	router *models.NetworkRouter,
	id string,
	keyMaterial string,
) error {
	writer.EnableCalls++
	writer.ID = id
	writer.Key = keyMaterial

	if router != nil {
		writer.RouterID = router.ID
	}

	return writer.Err
}

func (writer *fakePPPSecretWriter) DisablePPPSecret(
	router *models.NetworkRouter,
	id string,
	keyMaterial string,
) error {
	writer.DisableCalls++
	writer.ID = id
	writer.Key = keyMaterial

	if router != nil {
		writer.RouterID = router.ID
	}

	return writer.Err
}

func reconciliationExecutionPlan(
	subscription models.Subscription,
	router models.NetworkRouter,
	action PPPSecretReconciliationAction,
) PPPSecretReconciliationPlan {
	return PPPSecretReconciliationPlan{
		SubscriptionID: subscription.ID,
		RouterID:       router.ID,
		RouterCode:     router.Code,
		Username:       subscription.PPPoEUsername,
		Profile:        "Go_P25",
		Action:         action,
		Reason:         "test decision",
	}
}

func TestExecuteSubscriptionPPPSecretReconciliationPlanCreate(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	writer := &fakePPPSecretWriter{
		AddID: "*123",
	}

	plan := reconciliationExecutionPlan(
		subscription,
		router,
		PPPSecretActionCreate,
	)

	result, err :=
		ExecuteSubscriptionPPPSecretReconciliationPlan(
			plan,
			reconciliationPlanTestKey,
			writer,
		)
	if err != nil {
		t.Fatalf(
			"execute CREATE: %v",
			err,
		)
	}

	if !result.Executed {
		t.Fatal("CREATE was not executed")
	}

	if result.SecretID != "*123" {
		t.Fatalf(
			"secret id = %q",
			result.SecretID,
		)
	}

	if writer.AddCalls != 1 {
		t.Fatalf(
			"add calls = %d, want 1",
			writer.AddCalls,
		)
	}

	if writer.Input.Password !=
		"subscriber-secret" {
		t.Fatal(
			"CREATE did not receive decrypted subscriber credential",
		)
	}

	if writer.Input.Name !=
		"subscriber-1" {
		t.Fatalf(
			"name = %q",
			writer.Input.Name,
		)
	}

	if writer.Input.Profile != "Go_P25" {
		t.Fatalf(
			"profile = %q",
			writer.Input.Profile,
		)
	}
}

func TestExecuteSubscriptionPPPSecretReconciliationPlanUpdate(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	writer := &fakePPPSecretWriter{}

	plan := reconciliationExecutionPlan(
		subscription,
		router,
		PPPSecretActionUpdate,
	)

	plan.CurrentSecret = &mikrotik.PPPSecret{
		ID:       "*55",
		Name:     "subscriber-1",
		Service:  "pppoe",
		Profile:  "Old_Profile",
		Disabled: false,
	}

	result, err :=
		ExecuteSubscriptionPPPSecretReconciliationPlan(
			plan,
			reconciliationPlanTestKey,
			writer,
		)
	if err != nil {
		t.Fatalf(
			"execute UPDATE: %v",
			err,
		)
	}

	if !result.Executed {
		t.Fatal("UPDATE was not executed")
	}

	if writer.SetCalls != 1 {
		t.Fatalf(
			"set calls = %d, want 1",
			writer.SetCalls,
		)
	}

	if writer.ID != "*55" {
		t.Fatalf(
			"set id = %q",
			writer.ID,
		)
	}

	if writer.Input.Password !=
		"subscriber-secret" {
		t.Fatal(
			"UPDATE did not receive decrypted subscriber credential",
		)
	}
}

func TestExecuteSubscriptionPPPSecretReconciliationPlanDisableDoesNotNeedSubscriberCredential(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"SUSPENDED",
		)

	if err := db.Model(
		&models.Subscription{},
	).Where(
		"id = ?",
		subscription.ID,
	).Update(
		"pp_po_e_password_encrypted",
		"",
	).Error; err != nil {
		t.Fatalf(
			"clear encrypted credential: %v",
			err,
		)
	}

	writer := &fakePPPSecretWriter{}

	plan := reconciliationExecutionPlan(
		subscription,
		router,
		PPPSecretActionDisable,
	)

	plan.CurrentSecret = &mikrotik.PPPSecret{
		ID:       "*77",
		Name:     "subscriber-1",
		Disabled: false,
	}

	result, err :=
		ExecuteSubscriptionPPPSecretReconciliationPlan(
			plan,
			reconciliationPlanTestKey,
			writer,
		)
	if err != nil {
		t.Fatalf(
			"execute DISABLE without subscriber credential: %v",
			err,
		)
	}

	if !result.Executed {
		t.Fatal("DISABLE was not executed")
	}

	if writer.DisableCalls != 1 {
		t.Fatalf(
			"disable calls = %d, want 1",
			writer.DisableCalls,
		)
	}

	if writer.AddCalls != 0 ||
		writer.SetCalls != 0 {
		t.Fatal(
			"DISABLE unexpectedly used password-bearing mutation",
		)
	}
}

func TestExecuteSubscriptionPPPSecretReconciliationPlanDisableDisconnectsLiveSession(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)
	subscription, router, _ := createPPPReconciliationPlanFixture(t, db, "SUSPENDED")

	writer := &disconnectingPPPSecretWriter{}
	plan := reconciliationExecutionPlan(subscription, router, PPPSecretActionDisable)
	plan.CurrentSecret = &mikrotik.PPPSecret{ID: "*77", Name: "subscriber-1"}

	result, err := ExecuteSubscriptionPPPSecretReconciliationPlan(plan, reconciliationPlanTestKey, writer)
	if err != nil {
		t.Fatalf("execute DISABLE: %v", err)
	}
	if !result.Executed || writer.DisableCalls != 1 {
		t.Fatalf("disable execution = %#v, calls = %d", result, writer.DisableCalls)
	}
	if writer.DisconnectCalls != 1 || writer.DisconnectedUser != "subscriber-1" {
		t.Fatalf("disconnect calls/user = %d/%q, want 1/subscriber-1", writer.DisconnectCalls, writer.DisconnectedUser)
	}
}

func TestExecuteSubscriptionPPPSecretReconciliationPlanEnableDoesNotNeedSubscriberCredential(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	if err := db.Model(
		&models.Subscription{},
	).Where(
		"id = ?",
		subscription.ID,
	).Update(
		"pp_po_e_password_encrypted",
		"",
	).Error; err != nil {
		t.Fatalf(
			"clear encrypted credential: %v",
			err,
		)
	}

	writer := &fakePPPSecretWriter{}

	plan := reconciliationExecutionPlan(
		subscription,
		router,
		PPPSecretActionEnable,
	)

	plan.CurrentSecret = &mikrotik.PPPSecret{
		ID:       "*88",
		Name:     "subscriber-1",
		Disabled: true,
	}

	result, err :=
		ExecuteSubscriptionPPPSecretReconciliationPlan(
			plan,
			reconciliationPlanTestKey,
			writer,
		)
	if err != nil {
		t.Fatalf(
			"execute ENABLE without subscriber credential: %v",
			err,
		)
	}

	if !result.Executed {
		t.Fatal("ENABLE was not executed")
	}

	if writer.EnableCalls != 1 {
		t.Fatalf(
			"enable calls = %d, want 1",
			writer.EnableCalls,
		)
	}
}

func TestExecuteSubscriptionPPPSecretReconciliationPlanNoopDoesNotNeedSubscriberCredential(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	if err := db.Model(
		&models.Subscription{},
	).Where(
		"id = ?",
		subscription.ID,
	).Update(
		"pp_po_e_password_encrypted",
		"",
	).Error; err != nil {
		t.Fatalf(
			"clear credential: %v",
			err,
		)
	}

	writer := &fakePPPSecretWriter{}

	plan := reconciliationExecutionPlan(
		subscription,
		router,
		PPPSecretActionNoop,
	)

	result, err :=
		ExecuteSubscriptionPPPSecretReconciliationPlan(
			plan,
			reconciliationPlanTestKey,
			writer,
		)
	if err != nil {
		t.Fatalf("execute NOOP: %v", err)
	}

	if result.Executed {
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

func TestExecuteSubscriptionPPPSecretReconciliationPlanNoopDisabledSecretDisconnectsLiveSession(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)
	subscription, router, _ := createPPPReconciliationPlanFixture(t, db, "SUSPENDED")

	writer := &disconnectingPPPSecretWriter{}
	plan := reconciliationExecutionPlan(subscription, router, PPPSecretActionNoop)
	plan.CurrentSecret = &mikrotik.PPPSecret{
		ID:       "*77",
		Name:     "subscriber-1",
		Disabled: true,
	}

	result, err := ExecuteSubscriptionPPPSecretReconciliationPlan(
		plan,
		reconciliationPlanTestKey,
		writer,
	)
	if err != nil {
		t.Fatalf("execute disabled NOOP: %v", err)
	}
	if !result.Executed || result.SecretID != "*77" {
		t.Fatalf("disabled NOOP execution = %#v", result)
	}
	if writer.DisconnectCalls != 1 || writer.DisconnectedUser != "subscriber-1" {
		t.Fatalf("disconnect calls/user = %d/%q, want 1/subscriber-1", writer.DisconnectCalls, writer.DisconnectedUser)
	}
	if writer.AddCalls+writer.SetCalls+writer.EnableCalls+writer.DisableCalls != 0 {
		t.Fatal("disabled NOOP unexpectedly mutated PPP secret")
	}
}

func TestExecuteSubscriptionPPPSecretReconciliationPlanConflictDoesNotMutate(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	writer := &fakePPPSecretWriter{}

	plan := reconciliationExecutionPlan(
		subscription,
		router,
		PPPSecretActionConflict,
	)

	plan.Reason = "duplicate username"

	_, err :=
		ExecuteSubscriptionPPPSecretReconciliationPlan(
			plan,
			reconciliationPlanTestKey,
			writer,
		)
	if err == nil {
		t.Fatal(
			"expected conflict to fail",
		)
	}

	if !strings.Contains(
		err.Error(),
		"reconciliation conflict",
	) {
		t.Fatalf(
			"unexpected conflict error: %v",
			err,
		)
	}

	if writer.AddCalls+
		writer.SetCalls+
		writer.EnableCalls+
		writer.DisableCalls != 0 {
		t.Fatal(
			"CONFLICT unexpectedly mutated router",
		)
	}
}

func TestExecuteSubscriptionPPPSecretReconciliationPlanCreateRequiresCredential(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	if err := db.Model(
		&models.Subscription{},
	).Where(
		"id = ?",
		subscription.ID,
	).Update(
		"pp_po_e_password_encrypted",
		"",
	).Error; err != nil {
		t.Fatalf(
			"clear credential: %v",
			err,
		)
	}

	writer := &fakePPPSecretWriter{}

	plan := reconciliationExecutionPlan(
		subscription,
		router,
		PPPSecretActionCreate,
	)

	_, err :=
		ExecuteSubscriptionPPPSecretReconciliationPlan(
			plan,
			reconciliationPlanTestKey,
			writer,
		)
	if err == nil {
		t.Fatal(
			"expected CREATE without credential to fail",
		)
	}

	if writer.AddCalls != 0 {
		t.Fatal(
			"CREATE writer called before credential validation",
		)
	}
}

func TestExecuteSubscriptionPPPSecretReconciliationPlanRejectsStaleRouter(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	writer := &fakePPPSecretWriter{}

	plan := reconciliationExecutionPlan(
		subscription,
		router,
		PPPSecretActionCreate,
	)

	plan.RouterID = router.ID + 100

	_, err :=
		ExecuteSubscriptionPPPSecretReconciliationPlan(
			plan,
			reconciliationPlanTestKey,
			writer,
		)
	if err == nil {
		t.Fatal(
			"expected stale router plan to fail",
		)
	}

	if writer.AddCalls != 0 {
		t.Fatal(
			"stale reconciliation plan mutated router",
		)
	}
}

func TestExecuteSubscriptionPPPSecretReconciliationPlanPropagatesWriterFailure(
	t *testing.T,
) {
	db := setupPPPReconciliationPlanDB(t)

	subscription, router, _ :=
		createPPPReconciliationPlanFixture(
			t,
			db,
			"ACTIVE",
		)

	writer := &fakePPPSecretWriter{
		Err: errors.New("router mutation failed"),
	}

	plan := reconciliationExecutionPlan(
		subscription,
		router,
		PPPSecretActionCreate,
	)

	_, err :=
		ExecuteSubscriptionPPPSecretReconciliationPlan(
			plan,
			reconciliationPlanTestKey,
			writer,
		)
	if err == nil {
		t.Fatal(
			"expected writer failure",
		)
	}

	if !strings.Contains(
		err.Error(),
		"create RouterOS PPP secret",
	) {
		t.Fatalf(
			"unexpected writer error: %v",
			err,
		)
	}
}
