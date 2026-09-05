package services

import (
	"fmt"
	"strings"

	"github.com/tscommunication/ts-cloud/internal/mikrotik"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/security"
)

type PPPSecretReconciliationAction string

const (
	PPPSecretActionCreate   PPPSecretReconciliationAction = "CREATE"
	PPPSecretActionUpdate   PPPSecretReconciliationAction = "UPDATE"
	PPPSecretActionEnable   PPPSecretReconciliationAction = "ENABLE"
	PPPSecretActionDisable  PPPSecretReconciliationAction = "DISABLE"
	PPPSecretActionNoop     PPPSecretReconciliationAction = "NOOP"
	PPPSecretActionConflict PPPSecretReconciliationAction = "CONFLICT"
)

type PPPSecretDesiredState struct {
	Username      string
	Service       string
	Profile       string
	CallerID      string
	RemoteAddress string
	Disabled      bool
}

type PPPSecretReconciliationDecision struct {
	Action  PPPSecretReconciliationAction
	Reason  string
	Desired PPPSecretDesiredState
	Current *mikrotik.PPPSecret
}

func applyCustomerInternetAccount(
	subscription *models.Subscription,
) {
	if subscription == nil || subscription.InternetAccount == nil ||
		subscription.InternetAccount.ID == 0 {
		return
	}

	account := subscription.InternetAccount
	subscription.RouterID = account.RouterID
	subscription.PPPoEUsername = account.PPPoEUsername
	subscription.PPPoEPasswordEncrypted = account.PPPoEPasswordEncrypted
	if account.PackageID != 0 {
		subscription.PackageID = account.PackageID
		if account.Package.ID != 0 {
			subscription.Package = account.Package
		}
	}
	if account.ActivationDate != nil {
		subscription.ActivationDate = *account.ActivationDate
	}
	if account.BillingDay > 0 {
		subscription.BillingDay = account.BillingDay
	}
	if account.NextBillingDate != nil {
		subscription.NextBillingDate = *account.NextBillingDate
	}
	if account.ExpiryDate != nil {
		subscription.ExpiryDate = *account.ExpiryDate
	}
	if strings.TrimSpace(account.Status) != "" {
		subscription.Status = account.Status
	}
}

func BuildSubscriptionPPPSecretDesiredState(
	subscription *models.Subscription,
	pkg *models.Package,
) (PPPSecretDesiredState, error) {
	if subscription == nil {
		return PPPSecretDesiredState{},
			fmt.Errorf("subscription is required")
	}

	if pkg == nil {
		return PPPSecretDesiredState{},
			fmt.Errorf("package is required")
	}

	applyCustomerInternetAccount(subscription)

	username := strings.TrimSpace(
		subscription.PPPoEUsername,
	)
	if username == "" {
		return PPPSecretDesiredState{},
			fmt.Errorf("subscription PPPoE username is required")
	}

	if subscription.RouterID == 0 {
		return PPPSecretDesiredState{},
			fmt.Errorf("subscription router is required")
	}

	profile := strings.TrimSpace(
		pkg.MikroTikProfile,
	)
	if profile == "" {
		return PPPSecretDesiredState{},
			fmt.Errorf(
				"package MikroTik profile is required",
			)
	}

	status := strings.ToUpper(
		strings.TrimSpace(subscription.Status),
	)

	var disabled bool

	switch status {
	case "ACTIVE", TemporaryInternetStatusActive:
		disabled = false
	case "SUSPENDED", "EXPIRED", "DISCONNECTED", "INACTIVE":
		disabled = true
	default:
		return PPPSecretDesiredState{},
			fmt.Errorf(
				"unsupported subscription status %q",
				subscription.Status,
			)
	}
	if subscription.Customer.ID != 0 && strings.ToUpper(strings.TrimSpace(subscription.Customer.Status)) != "ACTIVE" {
		disabled = true
	}

	callerID, remoteAddress := "", ""
	if subscription.InternetAccount != nil {
		callerID = strings.TrimSpace(subscription.InternetAccount.MACAddress)
		remoteAddress = strings.TrimSpace(subscription.InternetAccount.StaticIPAddress)
	}

	return PPPSecretDesiredState{
		Username:      username,
		Service:       "pppoe",
		Profile:       profile,
		CallerID:      callerID,
		RemoteAddress: remoteAddress,
		Disabled:      disabled,
	}, nil
}

func DecideSubscriptionPPPSecretReconciliation(
	subscription *models.Subscription,
	pkg *models.Package,
	secrets []mikrotik.PPPSecret,
) (PPPSecretReconciliationDecision, error) {
	desired, err :=
		BuildSubscriptionPPPSecretDesiredState(
			subscription,
			pkg,
		)
	if err != nil {
		return PPPSecretReconciliationDecision{},
			err
	}

	matches := make(
		[]mikrotik.PPPSecret,
		0,
		len(secrets),
	)

	for _, secret := range secrets {
		if strings.EqualFold(
			strings.TrimSpace(secret.Name),
			desired.Username,
		) {
			matches = append(matches, secret)
		}
	}

	if len(matches) > 1 {
		return PPPSecretReconciliationDecision{
			Action:  PPPSecretActionConflict,
			Reason:  "multiple RouterOS PPP secrets match subscription username",
			Desired: desired,
		}, nil
	}

	if len(matches) == 0 {
		if strings.TrimSpace(
			subscription.PPPoEPasswordEncrypted,
		) == "" {
			return PPPSecretReconciliationDecision{},
				fmt.Errorf(
					"subscription PPPoE credential is not configured",
				)
		}

		return PPPSecretReconciliationDecision{
			Action:  PPPSecretActionCreate,
			Reason:  "RouterOS PPP secret does not exist",
			Desired: desired,
		}, nil
	}

	current := matches[0]

	if strings.TrimSpace(current.ID) == "" {
		return PPPSecretReconciliationDecision{
			Action:  PPPSecretActionConflict,
			Reason:  "RouterOS PPP secret is missing internal id",
			Desired: desired,
			Current: &current,
		}, nil
	}

	currentService := strings.ToLower(
		strings.TrimSpace(current.Service),
	)
	if currentService == "" {
		currentService = "pppoe"
	}

	if currentService != desired.Service ||
		strings.TrimSpace(current.Profile) !=
			desired.Profile ||
		strings.TrimSpace(current.CallerID) != desired.CallerID ||
		strings.TrimSpace(current.RemoteAddress) != desired.RemoteAddress {
		return PPPSecretReconciliationDecision{
			Action:  PPPSecretActionUpdate,
			Reason:  "RouterOS PPP secret service, profile, MAC or static IP binding differs from customer",
			Desired: desired,
			Current: &current,
		}, nil
	}

	if current.Disabled != desired.Disabled {
		action := PPPSecretActionEnable
		reason := "subscription requires RouterOS PPP secret to be enabled"

		if desired.Disabled {
			action = PPPSecretActionDisable
			reason =
				"subscription requires RouterOS PPP secret to be disabled"
		}

		return PPPSecretReconciliationDecision{
			Action:  action,
			Reason:  reason,
			Desired: desired,
			Current: &current,
		}, nil
	}

	return PPPSecretReconciliationDecision{
		Action:  PPPSecretActionNoop,
		Reason:  "RouterOS PPP secret already matches subscription state",
		Desired: desired,
		Current: &current,
	}, nil
}

type PPPSecretReconciliationPlan struct {
	SubscriptionID uint
	RouterID       uint
	RouterCode     string
	Username       string
	Profile        string
	CallerID       string
	RemoteAddress  string
	Action         PPPSecretReconciliationAction
	Reason         string
	CurrentSecret  *mikrotik.PPPSecret
}

type PPPSecretReader interface {
	ListPPPSecrets(
		router *models.NetworkRouter,
		name string,
		keyMaterial string,
	) ([]mikrotik.PPPSecret, error)
}

type MikroTikPPPSecretReader struct{}

func (MikroTikPPPSecretReader) ListPPPSecrets(
	router *models.NetworkRouter,
	name string,
	keyMaterial string,
) ([]mikrotik.PPPSecret, error) {
	if router == nil {
		return nil, fmt.Errorf("router is required")
	}

	if strings.TrimSpace(
		router.APIPasswordEncrypted,
	) == "" {
		return nil, fmt.Errorf(
			"router API credentials are not configured",
		)
	}

	password, err := security.DecryptSecret(
		router.APIPasswordEncrypted,
		keyMaterial,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"decrypt router API credential: %w",
			err,
		)
	}

	return mikrotik.ListPPPSecrets(
		router.Host,
		router.APIPort,
		router.UseTLS,
		router.APIUsername,
		password,
		name,
	)
}

func BuildSubscriptionPPPSecretReconciliationPlan(
	subscriptionID uint,
	keyMaterial string,
	reader PPPSecretReader,
) (PPPSecretReconciliationPlan, error) {
	if subscriptionID == 0 {
		return PPPSecretReconciliationPlan{},
			fmt.Errorf("subscription id is required")
	}

	if reader == nil {
		return PPPSecretReconciliationPlan{},
			fmt.Errorf("PPP secret reader is required")
	}

	subscription, err :=
		repositories.GetSubscriptionByID(
			subscriptionID,
		)
	if err != nil {
		return PPPSecretReconciliationPlan{},
			fmt.Errorf("subscription not found")
	}
	applyCustomerInternetAccount(subscription)

	pkg := &subscription.Package

	if pkg.ID == 0 {
		pkg, err = repositories.GetPackageByID(
			subscription.PackageID,
		)
		if err != nil {
			return PPPSecretReconciliationPlan{},
				fmt.Errorf("package not found")
		}
	}

	desired, err :=
		BuildSubscriptionPPPSecretDesiredState(
			subscription,
			pkg,
		)
	if err != nil {
		return PPPSecretReconciliationPlan{},
			err
	}

	router, err := repositories.GetNetworkRouter(
		subscription.RouterID,
	)
	if err != nil {
		return PPPSecretReconciliationPlan{},
			fmt.Errorf("router not found")
	}

	if strings.ToUpper(
		strings.TrimSpace(router.Status),
	) != "ACTIVE" {
		return PPPSecretReconciliationPlan{},
			fmt.Errorf(
				"subscription requires an active router",
			)
	}

	secrets, err := reader.ListPPPSecrets(
		router,
		desired.Username,
		keyMaterial,
	)
	if err != nil {
		return PPPSecretReconciliationPlan{},
			fmt.Errorf(
				"read RouterOS PPP secrets: %w",
				err,
			)
	}

	decision, err :=
		DecideSubscriptionPPPSecretReconciliation(
			subscription,
			pkg,
			secrets,
		)
	if err != nil {
		return PPPSecretReconciliationPlan{},
			err
	}

	return PPPSecretReconciliationPlan{
		SubscriptionID: subscription.ID,
		RouterID:       router.ID,
		RouterCode:     router.Code,
		Username:       desired.Username,
		Profile:        desired.Profile,
		CallerID:       desired.CallerID,
		RemoteAddress:  desired.RemoteAddress,
		Action:         decision.Action,
		Reason:         decision.Reason,
		CurrentSecret:  decision.Current,
	}, nil
}

type PPPSecretWriter interface {
	AddPPPSecret(
		router *models.NetworkRouter,
		input mikrotik.PPPSecretInput,
		keyMaterial string,
	) (string, error)

	SetPPPSecret(
		router *models.NetworkRouter,
		id string,
		input mikrotik.PPPSecretInput,
		keyMaterial string,
	) error

	EnablePPPSecret(
		router *models.NetworkRouter,
		id string,
		keyMaterial string,
	) error

	DisablePPPSecret(
		router *models.NetworkRouter,
		id string,
		keyMaterial string,
	) error
}

// PPPActiveSessionTerminator is deliberately separate from PPPSecretWriter so
// existing test and third-party writers remain compatible. The production
// writer implements it, ensuring a disable takes effect on a live subscriber
// immediately instead of only preventing the next login.
type PPPActiveSessionTerminator interface {
	DisconnectPPPActiveSessions(
		router *models.NetworkRouter,
		username string,
		keyMaterial string,
	) error
}

type MikroTikPPPSecretWriter struct{}

func routerAPIPassword(
	router *models.NetworkRouter,
	keyMaterial string,
) (string, error) {
	if router == nil {
		return "", fmt.Errorf("router is required")
	}

	if strings.TrimSpace(
		router.APIPasswordEncrypted,
	) == "" {
		return "", fmt.Errorf(
			"router API credentials are not configured",
		)
	}

	password, err := security.DecryptSecret(
		router.APIPasswordEncrypted,
		keyMaterial,
	)
	if err != nil {
		return "", fmt.Errorf(
			"decrypt router API credential: %w",
			err,
		)
	}

	return password, nil
}

func (MikroTikPPPSecretWriter) AddPPPSecret(
	router *models.NetworkRouter,
	input mikrotik.PPPSecretInput,
	keyMaterial string,
) (string, error) {
	password, err := routerAPIPassword(
		router,
		keyMaterial,
	)
	if err != nil {
		return "", err
	}

	return mikrotik.AddPPPSecret(
		router.Host,
		router.APIPort,
		router.UseTLS,
		router.APIUsername,
		password,
		input,
	)
}

func (MikroTikPPPSecretWriter) SetPPPSecret(
	router *models.NetworkRouter,
	id string,
	input mikrotik.PPPSecretInput,
	keyMaterial string,
) error {
	password, err := routerAPIPassword(
		router,
		keyMaterial,
	)
	if err != nil {
		return err
	}

	return mikrotik.SetPPPSecret(
		router.Host,
		router.APIPort,
		router.UseTLS,
		router.APIUsername,
		password,
		id,
		input,
	)
}

func (MikroTikPPPSecretWriter) EnablePPPSecret(
	router *models.NetworkRouter,
	id string,
	keyMaterial string,
) error {
	password, err := routerAPIPassword(
		router,
		keyMaterial,
	)
	if err != nil {
		return err
	}

	return mikrotik.EnablePPPSecret(
		router.Host,
		router.APIPort,
		router.UseTLS,
		router.APIUsername,
		password,
		id,
	)
}

func (MikroTikPPPSecretWriter) DisablePPPSecret(
	router *models.NetworkRouter,
	id string,
	keyMaterial string,
) error {
	password, err := routerAPIPassword(
		router,
		keyMaterial,
	)
	if err != nil {
		return err
	}

	return mikrotik.DisablePPPSecret(
		router.Host,
		router.APIPort,
		router.UseTLS,
		router.APIUsername,
		password,
		id,
	)
}

func (MikroTikPPPSecretWriter) DisconnectPPPActiveSessions(
	router *models.NetworkRouter,
	username string,
	keyMaterial string,
) error {
	password, err := routerAPIPassword(router, keyMaterial)
	if err != nil {
		return err
	}

	return mikrotik.DisconnectPPPActiveSessions(
		router.Host,
		router.APIPort,
		router.UseTLS,
		router.APIUsername,
		password,
		username,
	)
}

type PPPSecretReconciliationExecution struct {
	Action   PPPSecretReconciliationAction
	Executed bool
	Reason   string
	SecretID string
}

func ExecuteSubscriptionPPPSecretReconciliationPlan(
	plan PPPSecretReconciliationPlan,
	keyMaterial string,
	writer PPPSecretWriter,
) (PPPSecretReconciliationExecution, error) {
	if plan.SubscriptionID == 0 {
		return PPPSecretReconciliationExecution{},
			fmt.Errorf("subscription id is required")
	}

	if plan.RouterID == 0 {
		return PPPSecretReconciliationExecution{},
			fmt.Errorf("router id is required")
	}

	if writer == nil {
		return PPPSecretReconciliationExecution{},
			fmt.Errorf("PPP secret writer is required")
	}

	subscription, err :=
		repositories.GetSubscriptionByID(
			plan.SubscriptionID,
		)
	if err != nil {
		return PPPSecretReconciliationExecution{},
			fmt.Errorf("subscription not found")
	}
	applyCustomerInternetAccount(subscription)

	if subscription.RouterID != plan.RouterID {
		return PPPSecretReconciliationExecution{},
			fmt.Errorf(
				"reconciliation plan router no longer matches subscription",
			)
	}

	router, err := repositories.GetNetworkRouter(
		plan.RouterID,
	)
	if err != nil {
		return PPPSecretReconciliationExecution{},
			fmt.Errorf("router not found")
	}

	if strings.ToUpper(
		strings.TrimSpace(router.Status),
	) != "ACTIVE" {
		return PPPSecretReconciliationExecution{},
			fmt.Errorf(
				"subscription requires an active router",
			)
	}

	execution := PPPSecretReconciliationExecution{
		Action: plan.Action,
		Reason: plan.Reason,
	}

	switch plan.Action {
	case PPPSecretActionNoop:
		return execution, nil

	case PPPSecretActionConflict:
		return execution, fmt.Errorf(
			"PPP secret reconciliation conflict: %s",
			plan.Reason,
		)

	case PPPSecretActionCreate:
		password, err :=
			GetSubscriptionPPPoEPassword(
				subscription,
				keyMaterial,
			)
		if err != nil {
			return execution, err
		}

		id, err := writer.AddPPPSecret(
			router,
			mikrotik.PPPSecretInput{
				Name:          plan.Username,
				Password:      password,
				Service:       "pppoe",
				Profile:       plan.Profile,
				CallerID:      plan.CallerID,
				RemoteAddress: plan.RemoteAddress,
				Disabled:      false,
			},
			keyMaterial,
		)
		if err != nil {
			return execution, fmt.Errorf(
				"create RouterOS PPP secret: %w",
				err,
			)
		}

		execution.Executed = true
		execution.SecretID = id

		return execution, nil

	case PPPSecretActionUpdate:
		if plan.CurrentSecret == nil ||
			strings.TrimSpace(
				plan.CurrentSecret.ID,
			) == "" {
			return execution, fmt.Errorf(
				"current RouterOS PPP secret id is required for update",
			)
		}

		password, err :=
			GetSubscriptionPPPoEPassword(
				subscription,
				keyMaterial,
			)
		if err != nil {
			return execution, err
		}

		if err := writer.SetPPPSecret(
			router,
			plan.CurrentSecret.ID,
			mikrotik.PPPSecretInput{
				Name:          plan.Username,
				Password:      password,
				Service:       "pppoe",
				Profile:       plan.Profile,
				CallerID:      plan.CallerID,
				RemoteAddress: plan.RemoteAddress,
				Disabled:      plan.CurrentSecret.Disabled,
			},
			keyMaterial,
		); err != nil {
			return execution, fmt.Errorf(
				"update RouterOS PPP secret: %w",
				err,
			)
		}

		execution.Executed = true
		execution.SecretID =
			plan.CurrentSecret.ID

		return execution, nil

	case PPPSecretActionEnable:
		if plan.CurrentSecret == nil ||
			strings.TrimSpace(
				plan.CurrentSecret.ID,
			) == "" {
			return execution, fmt.Errorf(
				"current RouterOS PPP secret id is required for enable",
			)
		}

		if err := writer.EnablePPPSecret(
			router,
			plan.CurrentSecret.ID,
			keyMaterial,
		); err != nil {
			return execution, fmt.Errorf(
				"enable RouterOS PPP secret: %w",
				err,
			)
		}

		execution.Executed = true
		execution.SecretID =
			plan.CurrentSecret.ID

		return execution, nil

	case PPPSecretActionDisable:
		if plan.CurrentSecret == nil ||
			strings.TrimSpace(
				plan.CurrentSecret.ID,
			) == "" {
			return execution, fmt.Errorf(
				"current RouterOS PPP secret id is required for disable",
			)
		}

		if err := writer.DisablePPPSecret(
			router,
			plan.CurrentSecret.ID,
			keyMaterial,
		); err != nil {
			return execution, fmt.Errorf(
				"disable RouterOS PPP secret: %w",
				err,
			)
		}

		execution.Executed = true
		execution.SecretID =
			plan.CurrentSecret.ID

		if terminator, ok := writer.(PPPActiveSessionTerminator); ok {
			if err := terminator.DisconnectPPPActiveSessions(
				router,
				plan.Username,
				keyMaterial,
			); err != nil {
				return execution, fmt.Errorf(
					"disconnect active RouterOS PPP session: %w",
					err,
				)
			}
		}

		return execution, nil

	default:
		return execution, fmt.Errorf(
			"unsupported PPP secret reconciliation action %q",
			plan.Action,
		)
	}
}

func DisableMigratedPPPSecret(
	oldRouterID uint,
	oldUsername string,
	newRouterID uint,
	newUsername string,
	keyMaterial string,
	reader PPPSecretReader,
	writer PPPSecretWriter,
) error {
	oldUsername = strings.TrimSpace(oldUsername)
	newUsername = strings.TrimSpace(newUsername)

	if oldRouterID == 0 || oldUsername == "" {
		return nil
	}

	if oldRouterID == newRouterID &&
		strings.EqualFold(oldUsername, newUsername) {
		return nil
	}

	if reader == nil {
		return fmt.Errorf("PPP secret reader is required")
	}
	if writer == nil {
		return fmt.Errorf("PPP secret writer is required")
	}

	router, err := repositories.GetNetworkRouter(oldRouterID)
	if err != nil {
		return fmt.Errorf("old router not found")
	}

	secrets, err := reader.ListPPPSecrets(
		router,
		oldUsername,
		keyMaterial,
	)
	if err != nil {
		return fmt.Errorf("read old RouterOS PPP secret: %w", err)
	}

	matches := make([]mikrotik.PPPSecret, 0, 1)
	for _, secret := range secrets {
		if strings.EqualFold(
			strings.TrimSpace(secret.Name),
			oldUsername,
		) {
			matches = append(matches, secret)
		}
	}

	if len(matches) == 0 {
		return nil
	}
	if len(matches) > 1 {
		return fmt.Errorf(
			"multiple old RouterOS PPP secrets match migrated username",
		)
	}

	secret := matches[0]
	if strings.TrimSpace(secret.ID) == "" {
		return fmt.Errorf("old RouterOS PPP secret is missing internal id")
	}
	if secret.Disabled {
		return nil
	}

	if err := writer.DisablePPPSecret(
		router,
		secret.ID,
		keyMaterial,
	); err != nil {
		return fmt.Errorf("disable old RouterOS PPP secret: %w", err)
	}

	return nil
}

type PPPSecretReconciliationResult struct {
	Plan      PPPSecretReconciliationPlan
	Execution PPPSecretReconciliationExecution
}

func ReconcileSubscriptionPPPSecret(
	subscriptionID uint,
	keyMaterial string,
	reader PPPSecretReader,
	writer PPPSecretWriter,
) (PPPSecretReconciliationResult, error) {
	if subscriptionID == 0 {
		return PPPSecretReconciliationResult{},
			fmt.Errorf("subscription id is required")
	}

	if reader == nil {
		return PPPSecretReconciliationResult{},
			fmt.Errorf("PPP secret reader is required")
	}

	if writer == nil {
		return PPPSecretReconciliationResult{},
			fmt.Errorf("PPP secret writer is required")
	}

	plan, err :=
		BuildSubscriptionPPPSecretReconciliationPlan(
			subscriptionID,
			keyMaterial,
			reader,
		)
	if err != nil {
		return PPPSecretReconciliationResult{},
			fmt.Errorf(
				"build PPP secret reconciliation plan: %w",
				err,
			)
	}

	execution, err :=
		ExecuteSubscriptionPPPSecretReconciliationPlan(
			plan,
			keyMaterial,
			writer,
		)
	if err != nil {
		return PPPSecretReconciliationResult{
				Plan:      plan,
				Execution: execution,
			},
			fmt.Errorf(
				"execute PPP secret reconciliation plan: %w",
				err,
			)
	}

	return PPPSecretReconciliationResult{
		Plan:      plan,
		Execution: execution,
	}, nil
}

func ReconcileSubscriptionPPPMigration(
	subscriptionID uint,
	oldRouterID uint,
	oldUsername string,
	keyMaterial string,
	reader PPPSecretReader,
	writer PPPSecretWriter,
) (PPPSecretReconciliationResult, error) {
	result, err := ReconcileSubscriptionPPPSecret(
		subscriptionID,
		keyMaterial,
		reader,
		writer,
	)
	if err != nil {
		return result, err
	}

	if err := DisableMigratedPPPSecret(
		oldRouterID,
		oldUsername,
		result.Plan.RouterID,
		result.Plan.Username,
		keyMaterial,
		reader,
		writer,
	); err != nil {
		return result, fmt.Errorf(
			"target PPP secret reconciled, but old PPP secret cleanup failed: %w",
			err,
		)
	}

	return result, nil
}

func ReconcileSubscriptionPPPMigrationWithMikroTik(
	subscriptionID uint,
	oldRouterID uint,
	oldUsername string,
	keyMaterial string,
) (PPPSecretReconciliationResult, error) {
	return ReconcileSubscriptionPPPMigration(
		subscriptionID,
		oldRouterID,
		oldUsername,
		keyMaterial,
		MikroTikPPPSecretReader{},
		MikroTikPPPSecretWriter{},
	)
}

func ReconcileSubscriptionPPPSecretWithMikroTik(
	subscriptionID uint,
	keyMaterial string,
) (PPPSecretReconciliationResult, error) {
	return ReconcileSubscriptionPPPSecret(
		subscriptionID,
		keyMaterial,
		MikroTikPPPSecretReader{},
		MikroTikPPPSecretWriter{},
	)
}

// ReconcileSubscriptionPPPSecretCredentialWithMikroTik forces an UPDATE when
// the RouterOS record otherwise looks unchanged. RouterOS does not return PPP
// secret passwords, so a password change cannot be detected through reads.
func ReconcileSubscriptionPPPSecretCredentialWithMikroTik(
	subscriptionID uint,
	keyMaterial string,
) (PPPSecretReconciliationResult, error) {
	plan, err := BuildSubscriptionPPPSecretReconciliationPlan(
		subscriptionID,
		keyMaterial,
		MikroTikPPPSecretReader{},
	)
	if err != nil {
		return PPPSecretReconciliationResult{}, fmt.Errorf("build PPP secret reconciliation plan: %w", err)
	}
	if plan.Action == PPPSecretActionNoop && plan.CurrentSecret != nil {
		plan.Action = PPPSecretActionUpdate
		plan.Reason = "customer PPPoE password changed; reapply write-only RouterOS credential"
	}
	execution, err := ExecuteSubscriptionPPPSecretReconciliationPlan(plan, keyMaterial, MikroTikPPPSecretWriter{})
	result := PPPSecretReconciliationResult{Plan: plan, Execution: execution}
	if err != nil {
		return result, fmt.Errorf("execute PPP secret reconciliation plan: %w", err)
	}
	return result, nil
}
