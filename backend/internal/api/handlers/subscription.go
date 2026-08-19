package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/repositories"
	"github.com/tscommunication/ts-cloud/internal/services"
)

// GetSubscriptions godoc
//
//	@Summary		List Subscriptions
//	@Description	Get all subscriptions
//	@Tags			Subscription
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Router			/api/v1/subscriptions [get]
func GetSubscriptions(c *gin.Context) {
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	validStatuses := map[string]bool{
		"": true, "ACTIVE": true, "SUSPENDED": true, "EXPIRED": true, "DISCONNECTED": true,
	}
	if !validStatuses[status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription status"})
		return
	}

	expiringWithinDays := 0
	if value := c.Query("expiring_within_days"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 365 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Expiry window must be between 1 and 365 days"})
			return
		}
		expiringWithinDays = parsed
	}

	subscriptions, err := services.ListSubscriptions(repositories.SubscriptionListParams{
		Status:             status,
		ExpiringWithinDays: expiringWithinDays,
	}, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch subscriptions",
		})
		return
	}

	response := make([]dto.SubscriptionResponse, 0)

	for _, s := range subscriptions {
		response = append(response, dto.ToSubscriptionResponse(s))
	}

	c.JSON(http.StatusOK, gin.H{
		"count":         len(response),
		"subscriptions": response,
	})
}

// GetSubscription godoc
//
//	@Summary		Get Subscription
//	@Description	Get subscription by ID
//	@Tags			Subscription
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		int	true	"Subscription ID"
//	@Success		200	{object}	dto.SubscriptionResponse
//	@Router			/api/v1/subscriptions/{id} [get]
func GetSubscription(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID",
		})
		return
	}

	subscription, err := services.GetSubscriptionByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Subscription not found",
		})
		return
	}

	c.JSON(
		http.StatusOK,
		dto.ToSubscriptionResponse(*subscription),
	)
}

// CreateSubscription godoc
//
//	@Summary		Create Subscription
//	@Description	Create new subscription
//	@Tags			Subscription
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body	dto.CreateSubscriptionRequest	true	"Subscription"
//	@Success		201		{object}	dto.SubscriptionResponse
//	@Router			/api/v1/subscriptions [post]
func CreateSubscription(
	cfg *config.Config,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateSubscriptionRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request",
			})
			return
		}

		subscriptionCode := "SUB-000001"

		lastSubscription, err := services.GetLastSubscription()
		if err == nil {
			subscriptionCode = fmt.Sprintf(
				"SUB-%06d",
				lastSubscription.ID+1,
			)
		}

		activationDate := time.Now()

		if req.ActivationDate != "" {
			t, err := time.Parse(
				"2006-01-02",
				req.ActivationDate,
			)
			if err == nil {
				activationDate = t
			}
		}

		nextBillingDate := activationDate.AddDate(0, 1, 0)
		expiryDate := activationDate.AddDate(0, 1, 0)

		subscription := models.Subscription{
			SubscriptionCode: subscriptionCode,

			CustomerID: req.CustomerID,
			PackageID:  req.PackageID,

			ActivationDate:  activationDate,
			NextBillingDate: nextBillingDate,
			ExpiryDate:      expiryDate,

			BillingDay: int(req.BillingDay),

			RouterID: req.RouterID,

			PPPoEUsername: req.PPPoEUsername,

			Status: "ACTIVE",

			Remarks: req.Remarks,
		}

		if err := services.SetSubscriptionPPPoEPassword(
			&subscription,
			req.PPPoEPassword,
			cfg.CredentialKey,
		); err != nil {
			c.JSON(
				http.StatusUnprocessableEntity,
				gin.H{"error": err.Error()},
			)
			return
		}

		if err := services.CreateSubscription(
			&subscription,
		); err != nil {
			c.JSON(
				http.StatusUnprocessableEntity,
				gin.H{"error": err.Error()},
			)
			return
		}

		c.JSON(
			http.StatusCreated,
			dto.ToSubscriptionResponse(subscription),
		)
	}
}

// UpdateSubscription godoc
//
//	@Summary		Update Subscription
//	@Description	Update subscription information
//	@Tags			Subscription
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id path int true "Subscription ID"
//	@Param			request body dto.CreateSubscriptionRequest true "Subscription"
//	@Success		200 {object} dto.SubscriptionResponse
//	@Router			/api/v1/subscriptions/{id} [put]
func UpdateSubscription(
	cfg *config.Config,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid ID",
			})
			return
		}

		subscription, err :=
			services.GetSubscriptionByID(uint(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Subscription not found",
			})
			return
		}

		var req dto.UpdateSubscriptionRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		subscription.BillingDay = int(req.BillingDay)
		subscription.RouterID = req.RouterID
		subscription.PPPoEUsername =
			req.PPPoEUsername
		subscription.Remarks = req.Remarks

		// Blank password preserves the existing encrypted
		// credential. A non-blank password replaces it.
		if err := services.SetSubscriptionPPPoEPassword(
			subscription,
			req.PPPoEPassword,
			cfg.CredentialKey,
		); err != nil {
			c.JSON(
				http.StatusUnprocessableEntity,
				gin.H{"error": err.Error()},
			)
			return
		}

		if err := services.UpdateSubscription(
			subscription,
		); err != nil {
			c.JSON(
				http.StatusUnprocessableEntity,
				gin.H{"error": err.Error()},
			)
			return
		}

		c.JSON(
			http.StatusOK,
			dto.ToSubscriptionResponse(*subscription),
		)
	}
}

type subscriptionLifecycleReconciliationRunner func(
	subscription *models.Subscription,
	action services.SubscriptionLifecycleAction,
	keyMaterial string,
) (services.SubscriptionLifecycleReconciliationResult, error)

func suspendSubscriptionHandler(
	cfg *config.Config,
	runner subscriptionLifecycleReconciliationRunner,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		subscription, ok := getSubscriptionForAction(c)
		if !ok {
			return
		}

		if err := services.SuspendSubscription(
			subscription,
		); err != nil {
			c.JSON(
				http.StatusConflict,
				gin.H{"error": err.Error()},
			)
			return
		}

		reconciliation, reconciliationErr :=
			runner(
				subscription,
				services.SubscriptionLifecycleSuspend,
				cfg.CredentialKey,
			)

		response := gin.H{
			"subscription":         dto.ToSubscriptionResponse(*subscription),
			"pppoe_reconciliation": reconciliation,
		}

		if reconciliationErr != nil {
			response["pppoe_reconciliation_error"] =
				reconciliationErr.Error()
		}

		c.JSON(
			http.StatusOK,
			response,
		)
	}
}

func SuspendSubscription(
	cfg *config.Config,
) gin.HandlerFunc {
	return suspendSubscriptionHandler(
		cfg,
		services.ReconcileSubscriptionLifecycleWithMikroTikPostCommit,
	)
}

func activateSubscriptionHandler(
	cfg *config.Config,
	runner subscriptionLifecycleReconciliationRunner,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		subscription, ok := getSubscriptionForAction(c)
		if !ok {
			return
		}

		if err := services.ActivateSubscription(
			subscription,
			time.Now(),
		); err != nil {
			c.JSON(
				http.StatusConflict,
				gin.H{"error": err.Error()},
			)
			return
		}

		reconciliation, reconciliationErr :=
			runner(
				subscription,
				services.SubscriptionLifecycleActivate,
				cfg.CredentialKey,
			)

		response := gin.H{
			"subscription":         dto.ToSubscriptionResponse(*subscription),
			"pppoe_reconciliation": reconciliation,
		}

		if reconciliationErr != nil {
			response["pppoe_reconciliation_error"] =
				reconciliationErr.Error()
		}

		c.JSON(
			http.StatusOK,
			response,
		)
	}
}

func ActivateSubscription(
	cfg *config.Config,
) gin.HandlerFunc {
	return activateSubscriptionHandler(
		cfg,
		services.ReconcileSubscriptionLifecycleWithMikroTikPostCommit,
	)
}

func renewSubscriptionHandler(
	cfg *config.Config,
	runner subscriptionLifecycleReconciliationRunner,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		subscription, ok := getSubscriptionForAction(c)
		if !ok {
			return
		}

		var req dto.RenewSubscriptionRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": "Renewal months must be between 1 and 12",
				},
			)
			return
		}

		if err := services.RenewSubscription(
			subscription,
			req.Months,
			time.Now(),
		); err != nil {
			c.JSON(
				http.StatusConflict,
				gin.H{"error": err.Error()},
			)
			return
		}

		reconciliation, reconciliationErr :=
			runner(
				subscription,
				services.SubscriptionLifecycleRenew,
				cfg.CredentialKey,
			)

		response := gin.H{
			"subscription":         dto.ToSubscriptionResponse(*subscription),
			"pppoe_reconciliation": reconciliation,
		}

		if reconciliationErr != nil {
			response["pppoe_reconciliation_error"] =
				reconciliationErr.Error()
		}

		c.JSON(
			http.StatusOK,
			response,
		)
	}
}

func RenewSubscription(
	cfg *config.Config,
) gin.HandlerFunc {
	return renewSubscriptionHandler(
		cfg,
		services.ReconcileSubscriptionLifecycleWithMikroTikPostCommit,
	)
}

func getSubscriptionForAction(c *gin.Context) (*models.Subscription, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return nil, false
	}

	subscription, err := services.GetSubscriptionByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return nil, false
	}
	return subscription, true
}

// DisconnectSubscription godoc
//
//	@Summary		Disconnect Subscription
//	@Description	Disconnect a subscription while preserving its billing history
//	@Tags			Subscription
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id path int true "Subscription ID"
//	@Success		200 {object} dto.SubscriptionResponse
//	@Router			/api/v1/subscriptions/{id}/disconnect [post]
func disconnectSubscriptionHandler(
	cfg *config.Config,
	runner subscriptionLifecycleReconciliationRunner,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		subscription, ok := getSubscriptionForAction(c)
		if !ok {
			return
		}

		if err := services.DisconnectSubscription(
			subscription,
		); err != nil {
			c.JSON(
				http.StatusConflict,
				gin.H{"error": err.Error()},
			)
			return
		}

		reconciliation, reconciliationErr :=
			runner(
				subscription,
				services.SubscriptionLifecycleDisconnect,
				cfg.CredentialKey,
			)

		response := gin.H{
			"subscription":         dto.ToSubscriptionResponse(*subscription),
			"pppoe_reconciliation": reconciliation,
		}

		if reconciliationErr != nil {
			response["pppoe_reconciliation_error"] =
				reconciliationErr.Error()
		}

		c.JSON(
			http.StatusOK,
			response,
		)
	}
}

func DisconnectSubscription(
	cfg *config.Config,
) gin.HandlerFunc {
	return disconnectSubscriptionHandler(
		cfg,
		services.ReconcileSubscriptionLifecycleWithMikroTikPostCommit,
	)
}

type subscriptionPPPSecretReconciliationResponse struct {
	SubscriptionID uint   `json:"subscription_id"`
	RouterID       uint   `json:"router_id"`
	RouterCode     string `json:"router_code"`
	Username       string `json:"username"`
	Profile        string `json:"profile"`

	Action   string `json:"action"`
	Executed bool   `json:"executed"`
	Reason   string `json:"reason"`

	SecretID string `json:"secret_id,omitempty"`
}

type subscriptionPPPSecretReconciliationRunner func(
	subscriptionID uint,
	keyMaterial string,
) (services.PPPSecretReconciliationResult, error)

func reconcileSubscriptionPPPSecretHandler(
	cfg *config.Config,
	runner subscriptionPPPSecretReconciliationRunner,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(
			c.Param("id"),
			10,
			64,
		)
		if err != nil || id == 0 {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": "Invalid subscription ID",
				},
			)
			return
		}

		if runner == nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "PPP reconciliation runner is not configured",
				},
			)
			return
		}

		result, err := runner(
			uint(id),
			cfg.CredentialKey,
		)

		response :=
			subscriptionPPPSecretReconciliationResponse{
				SubscriptionID: result.Plan.SubscriptionID,
				RouterID:       result.Plan.RouterID,
				RouterCode:     result.Plan.RouterCode,
				Username:       result.Plan.Username,
				Profile:        result.Plan.Profile,
				Action: string(
					result.Execution.Action,
				),
				Executed: result.Execution.Executed,
				Reason:   result.Execution.Reason,
				SecretID: result.Execution.SecretID,
			}

		if response.Action == "" {
			response.Action = string(
				result.Plan.Action,
			)
		}

		if response.Reason == "" {
			response.Reason =
				result.Plan.Reason
		}

		if err != nil {
			c.JSON(
				http.StatusUnprocessableEntity,
				gin.H{
					"error":          err.Error(),
					"reconciliation": response,
				},
			)
			return
		}

		c.JSON(
			http.StatusOK,
			response,
		)
	}
}

func ReconcileSubscriptionPPPSecret(
	cfg *config.Config,
) gin.HandlerFunc {
	return reconcileSubscriptionPPPSecretHandler(
		cfg,
		services.ReconcileSubscriptionPPPSecretWithMikroTik,
	)
}
