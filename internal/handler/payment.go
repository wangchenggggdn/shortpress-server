package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/middleware"
	"shortpress-server/internal/repository/db/user"
	"shortpress-server/internal/service"
	"shortpress-server/pkg/log"

	"shortpress-server/internal/service/payment/coins"
	"shortpress-server/internal/service/payment/paypal"
	"shortpress-server/internal/service/payment/stripe"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PaymentHandler struct {
	*Handler
	stripeService    stripe.StripeService
	paypalService    paypal.PaypalService
	coinsService     coins.CoinsService
	userRepository   user.UserRepository //TODO 需要挪到service 中...
	analyticsService service.AnalyticsService
}

func NewPaymentHandler(
	handler *Handler,
	stripeService stripe.StripeService,
	paypalService paypal.PaypalService,
	coinsService coins.CoinsService,
	userRepository user.UserRepository,
	analyticsService service.AnalyticsService,
) *PaymentHandler {
	return &PaymentHandler{
		Handler:          handler,
		stripeService:    stripeService,
		paypalService:    paypalService,
		coinsService:     coinsService,
		userRepository:   userRepository,
		analyticsService: analyticsService,
	}
}

// AccountInfo godoc
// @Summary Get payment account information
// @Description Get the currently configured payment provider's account information
// @Tags payment
// @Accept json
// @Produce json
// @Param siteId query string true "Site ID"
// @Param provider query string true "Payment provider (stripe)"
// @Success 200 {object} api.PaymentAccountInfo "Return account information"
// @Router /api/payment/account/info [get]
func (h *PaymentHandler) AccountInfo(ctx *gin.Context) {
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

	//query 中读取 provider
	provider := ctx.Query("provider")
	switch provider {
	case "stripe":
		info, err := h.stripeService.GetAccountInfo(ctx, siteID)
		if err != nil {
			log.Error(ctx, fmt.Sprintf("Failed to get Stripe account info for site %s: %v", siteID, err))
			api.HandleError(ctx, err, nil)
			return
		}
		api.HandleSuccess(ctx, info)
	case "paypal":
		info, err := h.paypalService.GetAccountInfo(ctx, siteID)
		if err != nil {
			log.Error(ctx, fmt.Sprintf("Failed to get PayPal account info for site %s: %v", siteID, err))
			api.HandleError(ctx, err, nil)
			return
		}
		api.HandleSuccess(ctx, info)
	default:
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("unsupported payment provider"), nil)
	}
}

// ConfTest godoc
// @Summary Test payment configuration
// @Description Verify the payment provider's key configuration is valid
// @Tags payment
// @Accept json
// @Produce json
// @Param config body api.PaymentProviderConfig true "Payment configuration information"
// @Success 200 {object} api.Response "Return success information"
// @Router /api/payment/conf/test [post]
func (h *PaymentHandler) ConfTest(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var config api.PaymentProviderConfig
	if err := ctx.ShouldBindJSON(&config); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}
	provider := config.Provider
	switch provider {
	case "stripe":
		err := h.stripeService.ConfTest(ctx, config.StripeConf.SecretKey)
		if err != nil {
			log.Error(ctx, fmt.Sprintf("Stripe configuration test failed: %v", err))
			api.HandleError(ctx, common.ErrTestConfigNotAvailable, nil)
			return
		}
		api.HandleSuccess(ctx, nil)
	default:
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("unsupported payment provider"), nil)
	}
}

// ConfSave godoc
// @Summary Save payment configuration
// @Description Save user's payment provider configuration information, including keys
// @Tags payment
// @Accept json
// @Produce json
// @Param config body api.PaymentProviderConfig true "Payment configuration information"
// @Success 200 {object} api.Response "Return success information"
// @Router /api/payment/conf/save [post]
func (h *PaymentHandler) ConfSave(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var config api.PaymentProviderConfig
	if err := ctx.ShouldBindJSON(&config); err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to bind payment configuration: %v", err))
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	switch config.Provider {
	case "stripe":
		err := h.stripeService.SaveConfig(ctx, config)
		if err != nil {
			api.HandleError(ctx, err, nil)
			return
		}
		api.HandleSuccess(ctx, nil)
	case "paypal":
		err := h.paypalService.SaveConfig(ctx, config)
		if err != nil {
			api.HandleError(ctx, err, nil)
			return
		}
		api.HandleSuccess(ctx, nil)
	default:
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("unsupported payment provider"), nil)
	}
}

// ConfInfo godoc
// @Summary Get payment configuration information
// @Description Get site payment configuration information, including configured payment providers and their public keys
// @Tags payment
// @Accept json
// @Produce json
// @Param siteId query string true "Site ID"
// @Success 200 {object} api.Response{data=api.PaymentConfigInfoResponse} "Return configuration information"
// @Router /api/payment/conf/info [get]
func (h *PaymentHandler) ConfInfo(ctx *gin.Context) {
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

	// Get configuration info from all payment providers
	response := &api.PaymentConfigInfoResponse{}

	// Get Stripe configuration
	stripeResp, err := h.stripeService.GetConfigInfo(ctx, siteID)
	if err != nil {
		log.Warning(ctx, "Failed to get Stripe config: "+err.Error())
	} else if stripeResp != nil && stripeResp.Stripe != nil {
		response.Stripe = stripeResp.Stripe
	}

	// Get PayPal configuration
	paypalResp, err := h.paypalService.GetConfigInfo(ctx, siteID)
	if err != nil {
		log.Warning(ctx, "Failed to get PayPal config: "+err.Error())
	} else if paypalResp != nil && paypalResp.PayPal != nil {
		response.PayPal = paypalResp.PayPal
	}

	api.HandleSuccess(ctx, response)
}

// CreateCoinPackage godoc
// @Summary Create coin package
// @Description Create a new coin package for users to purchase
// @Tags payment
// @Accept json
// @Produce json
// @Param req body api.CoinPackageCreateRequest true "Coin package creation request"
// @Success 200 {object} api.CoinPackageCreateResponse "Return the successfully created package ID"
// @Router /api/payment/coins/package/create [post]
func (h *PaymentHandler) CreateCoinPackage(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.CoinPackageCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}
	if req.Price <= 0 || req.OriginalPrice <= 0 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("price must be greater than 0"), nil)
		return
	}

	// Create the coin package
	response, err := h.stripeService.CreateCoinPackage(ctx, req)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to create coin package: %v", err))
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// OrderCreate godoc
// @Summary Create payment order
// @Description Create a payment order for purchasing a coin package or subscription, return payment URLs
// @Tags client-payment
// @Accept json
// @Produce json
// @Param req body api.OrderCreateRequest true "Order creation request"
// @Success 200 {object} api.OrderCreateResponse "Return order information and payment URLs"
// @Router /api/client/payment/order/create [post]
func (h *PaymentHandler) OrderCreate(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	if userID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.OrderCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// Create the order based on payment method
	var response *api.OrderCreateResponse
	var err error

	switch req.PaymentMethod {
	case "stripe":
		response, err = h.stripeService.CreateOrder(ctx, userID, req)
	case "paypal":
		response, err = h.paypalService.CreateOrder(ctx, userID, req)
	default:
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("unsupported payment method: %s", req.PaymentMethod), nil)
		return
	}

	if err != nil {
		api.HandleError(ctx, err, "Failed to create payment order")
		return
	}

	api.HandleSuccess(ctx, response)
}

// StripeCallback handles webhook events from Stripe
func (h *PaymentHandler) StripeCallback(ctx *gin.Context) {

	// Process the webhook
	err := h.stripeService.HandleWebhook(ctx)
	if err != nil {
		// Important: Even on error, return 200 OK to Stripe
		// This prevents Stripe from retrying the webhook excessively
		// We'll log the error but tell Stripe we received it
		ctx.String(http.StatusOK, "Webhook received")
		return
	}

	// Return success
	ctx.String(http.StatusOK, "Webhook processed successfully")
}

// PayPalCallback handles webhook events from PayPal
func (h *PaymentHandler) PayPalCallback(ctx *gin.Context) {
	// Log incoming webhook request
	h.logger.Info("🔔 PayPal webhook received",
		zap.String("method", ctx.Request.Method),
		zap.String("path", ctx.Request.URL.Path),
		zap.String("query", ctx.Request.URL.RawQuery),
		zap.String("remote_addr", ctx.Request.RemoteAddr),
		zap.String("user_agent", ctx.Request.UserAgent()),
	)

	// Log important headers
	paypalTransId := ctx.GetHeader("Paypal-Transmission-Id")
	paypalCertId := ctx.GetHeader("Paypal-Cert-Id")
	paypalAuthAlgo := ctx.GetHeader("Paypal-Auth-Algo")

	h.logger.Info("📋 PayPal webhook headers",
		zap.String("paypal_transmission_id", paypalTransId),
		zap.String("paypal_cert_id", paypalCertId),
		zap.String("paypal_auth_algo", paypalAuthAlgo),
	)

	// Read and log request body
	bodyBytes, err := ctx.GetRawData()
	if err != nil {
		h.logger.Error("❌ Failed to read request body", zap.Error(err))
		ctx.String(http.StatusBadRequest, "Failed to read body")
		return
	}

	// Log the payload (truncated if too long)
	bodyString := string(bodyBytes)
	if len(bodyString) > 500 {
		h.logger.Info("📦 PayPal webhook payload (truncated)",
			zap.String("payload_preview", bodyString[:500]+"..."),
			zap.Int("payload_size", len(bodyString)),
		)
	} else {
		h.logger.Info("📦 PayPal webhook payload",
			zap.String("payload", bodyString),
		)
	}

	// Restore request body for further processing
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Process the webhook
	err = h.paypalService.HandleWebhook(ctx)
	if err != nil {
		// Important: Even on error, return 200 OK to PayPal
		// This prevents PayPal from retrying the webhook excessively
		// We'll log the error but tell PayPal we received it
		h.logger.Error("❌ PayPal webhook processing error", zap.Error(err))
		ctx.String(http.StatusOK, "Webhook received")
		return
	}

	h.logger.Info("✅ PayPal webhook processed successfully")
	// Return success
	ctx.String(http.StatusOK, "Webhook processed successfully")
}

// ProviderList godoc
// @Summary Get payment method list
// @Description Get the list of payment service providers supported by the current site
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Success 200 {object} []api.PaymentProviderInfo "Return payment service provider list"
// @Router /api/client/payment/provider/list [get]
func (h *PaymentHandler) ProviderList(ctx *gin.Context) {
	// Get site ID from header
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrBadRequest, "siteId is required")
		return
	}

	// Query database to check which providers are configured for this site
	providers := []*api.PaymentProviderInfo{}

	// Check Stripe configuration
	stripeConfigInfo, err := h.stripeService.GetConfigInfo(ctx, siteID)
	if err != nil {
		log.Warning(ctx, "Failed to get Stripe config: "+err.Error())
	} else if stripeConfigInfo != nil && stripeConfigInfo.Stripe != nil {
		// Stripe is configured and active
		providers = append(providers, &api.PaymentProviderInfo{
			Provider:    "stripe",
			DisplayName: "Stripe",
			Enabled:     true,
		})
	}

	// Check PayPal configuration
	paypalConfigInfo, err := h.paypalService.GetConfigInfo(ctx, siteID)
	if err != nil {
		log.Warning(ctx, "Failed to get PayPal config: "+err.Error())
	} else if paypalConfigInfo != nil && paypalConfigInfo.PayPal != nil {
		// PayPal is configured and active
		providers = append(providers, &api.PaymentProviderInfo{
			Provider:    "paypal",
			DisplayName: "PayPal",
			Enabled:     true,
		})
	}

	api.HandleSuccess(ctx, providers)
}

// CustomerTransactionHistory godoc
// @Summary Query specific user's coin transaction records
// @Description Administrators/creators query a specific user's coin transaction history by email
// @Tags payment
// @Accept json
// @Produce json
// @Param siteId query string true "Site ID"
// @Param email query string true "User email"
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Items per page, default 20" minimum(1) maximum(100) example(20)
// @Success 200 {object} api.Response{data=api.CoinTransactionHistoryResponse} "Return transaction record list"
// @Router /api/payment/customers/coins/transactions [get]
func (h *PaymentHandler) CustomerTransactionHistory(ctx *gin.Context) {
	// Verify admin/creator authentication
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// Get required parameters
	siteID := ctx.Query("siteId")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	email := ctx.Query("email")
	if email == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("email is required"), nil)
		return
	}

	// Get pagination parameters
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Find the user by email and site ID
	user, err := h.userRepository.GetByEmailAndSiteID(ctx, email, siteID)
	if err != nil {
		log.Error(ctx, "Failed to get user by email: "+err.Error())
		api.HandleError(ctx, common.ErrInternalServerError, nil)
		return
	}

	if user == nil {
		api.HandleError(ctx, common.ErrUserNotFound, nil)
		return
	}

	// Get transaction history for the user
	response, err := h.coinsService.GetTransactionHistory(ctx, user.UserID, siteID, page, pageSize)
	if err != nil {
		log.Error(ctx, "Failed to get transaction history: "+err.Error())
		api.HandleError(ctx, common.ErrInternalServerError, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// CustomerVideoUnlockHistory godoc
// @Summary Query specific user's video unlock records
// @Description Administrators/creators query a specific user's video and playlist unlock history by email
// @Tags payment
// @Accept json
// @Produce json
// @Param siteId query string true "Site ID"
// @Param email query string true "User email"
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Items per page, default 20" minimum(1) maximum(100) example(20)
// @Success 200 {object} api.Response{data=api.VideoUnlockHistoryResponse} "Return unlock record list"
// @Router /api/payment/customers/coins/videos/transactions [get]
func (h *PaymentHandler) CustomerVideoUnlockHistory(ctx *gin.Context) {
	// Verify admin/creator authentication
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// Get required parameters
	siteID := ctx.Query("siteId")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	email := ctx.Query("email")
	if email == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("email is required"), nil)
		return
	}

	// Get pagination parameters
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Find the user by email and site ID
	user, err := h.userRepository.GetByEmailAndSiteID(ctx, email, siteID)
	if err != nil {
		log.Error(ctx, "Failed to get user by email: "+err.Error())
		api.HandleError(ctx, common.ErrInternalServerError, nil)
		return
	}

	if user == nil {
		api.HandleError(ctx, common.ErrUserNotFound, nil)
		return
	}

	// Get video unlock history for the user
	response, err := h.coinsService.GetContentUnlockHistory(ctx, user.UserID, siteID, page, pageSize)
	if err != nil {
		log.Error(ctx, "Failed to get content unlock history: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// GetUserPurchases godoc
// @Summary Get user purchase records
// @Description Get current logged-in user's purchase history including coin purchases and subscriptions
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Items per page, default 20" minimum(1) maximum(100) example(20)
// @Success 200 {object} api.Response{data=api.PurchaseHistoryResponse} "Return purchase record list"
// @Router /api/client/payment/purchases [get]
func (h *PaymentHandler) GetUserPurchases(ctx *gin.Context) {
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

	// Get pagination parameters
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Get purchase records
	items, total, err := h.stripeService.GetUserPurchases(ctx, userID, siteID, page, pageSize)
	if err != nil {
		log.Error(ctx, "Failed to get user purchases: "+err.Error())
		api.HandleError(ctx, err, "Failed to get purchase records")
		return
	}

	response := &api.PurchaseHistoryResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	api.HandleSuccess(ctx, response)
}
