package handler

import (
	"fmt"
	"net/http"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/middleware"
	"shortpress-server/internal/service/payment/stripe"
	paypalservice "shortpress-server/internal/service/payment/paypal"
	"shortpress-server/internal/service/payment/sub"
	"shortpress-server/pkg/log"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SubscriptionHandler handles subscription-related operations
type SubscriptionHandler struct {
	*Handler
	subscriptionService sub.SubscriptionService
	stripeService       stripe.StripeService
	paypalService       paypalservice.PaypalService
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(
	handler *Handler,
	subscriptionService sub.SubscriptionService,
	stripeService stripe.StripeService,
	paypalService paypalservice.PaypalService,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		Handler:             handler,
		subscriptionService: subscriptionService,
		stripeService:       stripeService,
		paypalService:       paypalService,
	}
}

// CreateSubscription godoc
// @Summary Create subscription package
// @Description Create a new subscription package for users to subscribe to
// @Tags payment
// @Accept json
// @Produce json
// @Param req body api.SubscriptionData true "Subscription package creation request"
// @Success 200 {object} api.SubscriptionData "Return the successfully created package ID"
// @Router /api/payment/subscription/create [post]
func (h *SubscriptionHandler) CreateSubscription(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.SubscriptionData
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// Create the subscription package
	pkg, err := h.subscriptionService.CreateSubscriptionPackage(ctx, req.SiteID, &req)
	if err != nil {
		log.Error(ctx, "Failed to create subscription package: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	response := map[string]string{
		"packageId": pkg.PackageID,
	}

	api.HandleSuccess(ctx, response)
}

// ModifySubscription godoc
// @Summary Modify subscription package
// @Description Modify existing subscription package information
// @Tags payment
// @Accept json
// @Produce json
// @Param req body api.SubscriptionData true "Subscription package modification request"
// @Param packageId query string true "Subscription Package ID"
// @Success 200 {object} api.Response "Return successfully modified package ID"
// @Router /api/payment/subscription/modify [post]
func (h *SubscriptionHandler) ModifySubscription(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.SubscriptionData
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}
	if req.PackageID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("packageId is required"), nil)
		return
	}

	// Update the subscription package
	err := h.subscriptionService.UpdateSubscriptionPackage(ctx, req.PackageID, &req)
	if err != nil {
		log.Error(ctx, "Failed to update subscription package: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	response := map[string]string{
		"packageId": req.PackageID,
	}

	api.HandleSuccess(ctx, response)
}

// ListSubscriptions godoc
// @Summary Get subscription package list
// @Description Get all subscription packages for the specified site
// @Tags payment
// @Accept json
// @Produce json
// @Param siteId query string true "Site ID"
// @Param status query int false "Filter by status (-1 for all, 1 for active, 2 for inactive)"
// @Success 200 {object} []api.SubscriptionData "Return subscription package list"
// @Router /api/payment/subscription/list [get]
func (h *SubscriptionHandler) ListSubscriptions(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	siteID := ctx.Query("siteId")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	// Parse status if provided
	status := -1 // Default to all
	if statusStr := ctx.Query("status"); statusStr != "" {
		var err error
		status, err = h.parseQueryInt(statusStr, -1)
		if err != nil {
			api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("invalid status value"), nil)
			return
		}
	}

	// List packages
	packages, err := h.subscriptionService.ListBySiteID(ctx, siteID, status)
	if err != nil {
		log.Error(ctx, "Failed to list subscription packages: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, packages)
}

// GetSubscription godoc
// @Summary Get subscription package details
// @Description Get detailed information of a specific subscription package
// @Tags payment
// @Accept json
// @Produce json
// @Param siteId query string true "Site ID"
// @Param packageId query string true "Package ID"
// @Success 200 {object} api.SubscriptionData "Return subscription package details"
// @Router /api/payment/subscription/get [get]
func (h *SubscriptionHandler) GetSubscription(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// Get query parameters
	siteID := ctx.Query("siteId")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	packageID := ctx.Query("packageId")
	if packageID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("packageId is required"), nil)
		return
	}

	// Get subscription package details
	pkg, err := h.subscriptionService.GetByPackageID(ctx, siteID, packageID)
	if err != nil {
		log.Error(ctx, "Failed to get subscription package: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, pkg)
}

// ListClientSubscriptions godoc
// @Summary Get client-side subscription package list
// @Description Get all available subscription packages for the specified site, for client display
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Success 200 {object} []api.SubscriptionData "Return available subscription package list"
// @Router /api/client/payment/subscription/package/list [get]
func (h *SubscriptionHandler) ListClientSubscriptions(ctx *gin.Context) {
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	// Only list active packages (status = 1) for client-side
	packages, err := h.subscriptionService.ListBySiteID(ctx, siteID, 1)
	if err != nil {
		log.Error(ctx, "Failed to list subscription packages: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, packages)
}

// SubscriptionCreate godoc
// @Summary Create subscription order
// @Description Create a subscription order for the current user
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param req body api.OrderCreateRequest true "Order request containing subscription package ID"
// @Success 200 {object} api.OrderCreateResponse "Return subscription order information and checkout URL"
// @Router /api/client/payment/subscription/create [post]
func (h *SubscriptionHandler) SubscriptionCreate(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	if userID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	var req api.OrderCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// Ensure the site ID matches
	req.SiteID = siteID

	// Set the order type to subscription
	req.OrderType = api.OrderTypeSubscription

	// Create the subscription order based on payment method
	var response *api.OrderCreateResponse
	var err error

	switch req.PaymentMethod {
	case "stripe":
		response, err = h.stripeService.CreateSubscriptionOrder(ctx, userID, req)
	case "paypal":
		response, err = h.paypalService.CreateSubscriptionOrder(ctx, userID, req)
	default:
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("unsupported payment method: %s", req.PaymentMethod), nil)
		return
	}

	if err != nil {
		log.Error(ctx, "Failed to create subscription order: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// GetUserSubscriptions godoc
// @Summary Get user subscriptions
// @Description Get all subscription information for the current user
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Success 200 {object} []api.UserSubscriptionResponse "Return user subscription information"
// @Router /api/client/payment/subscription/user/list [get]
func (h *SubscriptionHandler) GetUserSubscriptions(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	if userID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// Get all subscriptions for the user
	subscriptions, err := h.subscriptionService.GetUserSubscriptions(ctx, userID)
	if err != nil {
		log.Error(ctx, "Failed to get user subscriptions: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, subscriptions)
}

// CancelSubscription godoc
// @Summary Cancel user subscription
// @Description Cancel a subscription for the current user
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param req body api.SubscriptionCancelRequest true "Cancel subscription request"
// @Success 200 {object} api.Response "Return success response"
// @Router /api/client/payment/subscription/cancel [post]
func (h *SubscriptionHandler) CancelSubscription(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	if userID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.SubscriptionCancelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	if req.SubscriptionID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("subscriptionId is required"), nil)
		return
	}

	// Get the subscription to determine which payment provider to use
	subscription, err := h.subscriptionService.GetUserSubscription(ctx, req.SubscriptionID)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to get subscription %s: %v", req.SubscriptionID, err))
		api.HandleErrorWithHttpCode(ctx, http.StatusNotFound, fmt.Errorf("subscription not found"), nil)
		return
	}

	if subscription == nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusNotFound, fmt.Errorf("subscription not found"), nil)
		return
	}

	// Check if subscription belongs to user
	if subscription.UserID != userID {
		api.HandleErrorWithHttpCode(ctx, http.StatusForbidden, fmt.Errorf("subscription does not belong to user"), nil)
		return
	}

	// Call the appropriate service based on the provider
	var cancelErr error
	switch subscription.Provider {
	case "stripe":
		cancelErr = h.stripeService.CancelSubscription(ctx, userID, req.SubscriptionID, req.CancelAtPeriodEnd)
	case "paypal":
		cancelErr = h.paypalService.CancelSubscription(ctx, userID, req.SubscriptionID, req.CancelAtPeriodEnd)
	default:
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("unsupported payment provider: %s", subscription.Provider), nil)
		return
	}

	if cancelErr != nil {
		log.Error(ctx, fmt.Sprintf("Failed to cancel subscription for user %s: %v", userID, cancelErr))
		api.HandleError(ctx, cancelErr, nil)
		return
	}

	// Return success response
	api.HandleSuccess(ctx, nil)
}

// Helper method to parse int query parameters
func (h *SubscriptionHandler) parseQueryInt(value string, defaultValue int) (int, error) {
	if value == "" {
		return defaultValue, nil
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue, err
	}

	return intValue, nil
}
