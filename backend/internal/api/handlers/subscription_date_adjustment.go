package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tscommunication/ts-cloud/internal/api/dto"
	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/models"
	"github.com/tscommunication/ts-cloud/internal/services"
)

type subscriptionDateAdjustmentRunner func(
	subscription *models.Subscription,
	newExpiryDate time.Time,
	reason string,
	adjustedByUserID uint,
	now time.Time,
) (*services.SubscriptionDateAdjustmentResult, error)

type subscriptionDateAdjustmentPPPReconciliationRunner func(
	subscriptionID uint,
	keyMaterial string,
) (services.PPPSecretReconciliationResult, error)

func adjustSubscriptionDateHandler(
	cfg *config.Config,
	adjuster subscriptionDateAdjustmentRunner,
	reconciler subscriptionDateAdjustmentPPPReconciliationRunner,
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
				gin.H{"error": "Invalid subscription ID"},
			)
			return
		}

		var req dto.AdjustSubscriptionDateRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": "new_expiry_date and reason are required",
				},
			)
			return
		}

		newExpiryDate, err := time.Parse(
			"2006-01-02",
			req.NewExpiryDate,
		)
		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": "new_expiry_date must use YYYY-MM-DD format",
				},
			)
			return
		}

		if adjuster == nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "subscription date adjustment runner is not configured",
				},
			)
			return
		}

		subscription, err :=
			services.GetSubscriptionByID(uint(id))
		if err != nil {
			c.JSON(
				http.StatusNotFound,
				gin.H{"error": "Subscription not found"},
			)
			return
		}

		actorID := c.GetUint("user_id")
		if actorID == 0 {
			c.JSON(
				http.StatusUnauthorized,
				gin.H{"error": "authenticated user is required"},
			)
			return
		}

		result, err := adjuster(
			subscription,
			newExpiryDate,
			req.Reason,
			actorID,
			time.Now(),
		)
		if err != nil {
			c.JSON(
				http.StatusUnprocessableEntity,
				gin.H{"error": err.Error()},
			)
			return
		}

		response := gin.H{
			"subscription": result.Subscription,
			"adjustment":   result.Audit,
		}

		// DB + audit have already been committed at this point.
		// PPP reconciliation is deliberately post-commit and must
		// never roll back the administrative date correction.
		if reconciler == nil {
			response["pppoe_reconciliation_error"] =
				"PPP reconciliation runner is not configured"

			c.JSON(
				http.StatusOK,
				response,
			)
			return
		}

		reconciliation, reconcileErr := reconciler(
			result.Subscription.ID,
			cfg.CredentialKey,
		)

		response["pppoe_reconciliation"] =
			subscriptionPPPSecretReconciliationResponse{
				SubscriptionID: reconciliation.Plan.SubscriptionID,
				RouterID:       reconciliation.Plan.RouterID,
				RouterCode:     reconciliation.Plan.RouterCode,
				Username:       reconciliation.Plan.Username,
				Profile:        reconciliation.Plan.Profile,
				Action: string(
					reconciliation.Execution.Action,
				),
				Executed: reconciliation.Execution.Executed,
				Reason:   reconciliation.Execution.Reason,
				SecretID: reconciliation.Execution.SecretID,
			}

		if reconcileErr != nil {
			response["pppoe_reconciliation_error"] =
				reconcileErr.Error()
		}

		c.JSON(
			http.StatusOK,
			response,
		)
	}
}

func AdjustSubscriptionDate(
	cfg *config.Config,
) gin.HandlerFunc {
	if cfg == nil {
		return func(c *gin.Context) {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": fmt.Sprintf(
						"%s",
						"application configuration is not available",
					),
				},
			)
		}
	}

	return adjustSubscriptionDateHandler(
		cfg,
		services.AdjustSubscriptionDateWithoutBilling,
		services.ReconcileSubscriptionPPPSecretWithMikroTik,
	)
}
