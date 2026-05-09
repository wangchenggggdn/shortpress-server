package handler

import (
	"fmt"
	"shortpress-server/internal/api"
	"shortpress-server/internal/middleware"
	"shortpress-server/internal/model"
	"shortpress-server/internal/service/payment/coins"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
)

// CoinsInternalHandler handles internal service-to-service coin operations
type CoinsInternalHandler struct {
	coinsService coins.CoinsService
	logger       *log.Logger
}

// NewCoinsInternalHandler creates a new instance of CoinsInternalHandler
func NewCoinsInternalHandler(coinsService coins.CoinsService, logger *log.Logger) *CoinsInternalHandler {
	return &CoinsInternalHandler{
		coinsService: coinsService,
		logger:       logger,
	}
}

// InternalAddCoins godoc
// @Summary Internal API: Add coins to user account
// @Description Internal service-to-service API for adding coins (treated as coin package purchase). Records both coin changes and actual amount. userId and siteId are obtained from middleware.
// @Tags internal-coins
// @Accept json
// @Produce json
// @Param request body api.InternalAddCoinsRequest true "Add coins request"
// @Success 200 {object} api.InternalAddCoinsResponse "Coins added successfully"
// @Failure 400 {object} api.Response "Invalid request"
// @Failure 401 {object} api.Response "Unauthorized"
// @Failure 500 {object} api.Response "Internal server error"
// @Router /api/internal/coins/add [post]
func (h *CoinsInternalHandler) InternalAddCoins(ctx *gin.Context) {
	// Get userId from JWT middleware
	userID := ctx.GetString("user_id")
	if userID == "" {
		log.Error(ctx, "Unauthorized: userId not found in context")
		api.HandleErrorWithHttpCode(ctx, 401, fmt.Errorf("unauthorized"), nil)
		return
	}

	// Get siteId from site middleware
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		log.Error(ctx, "SiteID not found in context")
		api.HandleErrorWithHttpCode(ctx, 400, fmt.Errorf("siteId is required"), nil)
		return
	}

	var req api.InternalAddCoinsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error(ctx, "Invalid request format: "+err.Error())
		api.HandleErrorWithHttpCode(ctx, 400, err, nil)
		return
	}

	// Set default currency if not provided
	if req.Currency == "" {
		req.Currency = "USD"
	}

	response, err := h.coinsService.InternalAddCoins(ctx, userID, siteID, req)
	if err != nil {
		log.Error(ctx, "Failed to add coins: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// InternalGetBalance godoc
// @Summary Internal API: Get user coin balance
// @Description Internal service-to-service API for getting user coin balance before deduction. userId and siteId are obtained from middleware.
// @Tags internal-coins
// @Accept json
// @Produce json
// @Success 200 {object} api.InternalGetBalanceResponse "User coin balance retrieved successfully"
// @Failure 400 {object} api.Response "Invalid request"
// @Failure 401 {object} api.Response "Unauthorized"
// @Failure 500 {object} api.Response "Internal server error"
// @Router /api/internal/coins/balance [get]
func (h *CoinsInternalHandler) InternalGetBalance(ctx *gin.Context) {
	// Get userId from JWT middleware
	userID := ctx.GetString("user_id")
	if userID == "" {
		log.Error(ctx, "Unauthorized: userId not found in context")
		api.HandleErrorWithHttpCode(ctx, 401, fmt.Errorf("unauthorized"), nil)
		return
	}

	// Get siteId from site middleware
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		log.Error(ctx, "SiteID not found in context")
		api.HandleErrorWithHttpCode(ctx, 400, fmt.Errorf("siteId is required"), nil)
		return
	}

	response, err := h.coinsService.InternalGetBalance(ctx, userID, siteID)
	if err != nil {
		log.Error(ctx, "Failed to get coin balance: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// InternalDeductCoins godoc
// @Summary Internal API: Deduct coins from user account
// @Description Internal service-to-service API for deducting coins. Records coin changes only. userId and siteId are obtained from middleware.
// @Tags internal-coins
// @Accept json
// @Produce json
// @Param request body api.InternalDeductCoinsRequest true "Deduct coins request"
// @Success 200 {object} api.InternalDeductCoinsResponse "Coins deducted successfully"
// @Failure 400 {object} api.Response "Invalid request"
// @Failure 401 {object} api.Response "Unauthorized"
// @Failure 409 {object} api.Response "Insufficient coins"
// @Failure 500 {object} api.Response "Internal server error"
// @Router /api/internal/coins/deduct [post]
func (h *CoinsInternalHandler) InternalDeductCoins(ctx *gin.Context) {
	// Get userId from JWT middleware
	userID := ctx.GetString("user_id")
	if userID == "" {
		log.Error(ctx, "Unauthorized: userId not found in context")
		api.HandleErrorWithHttpCode(ctx, 401, fmt.Errorf("unauthorized"), nil)
		return
	}

	// Get siteId from site middleware
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		log.Error(ctx, "SiteID not found in context")
		api.HandleErrorWithHttpCode(ctx, 400, fmt.Errorf("siteId is required"), nil)
		return
	}

	var req api.InternalDeductCoinsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error(ctx, "Invalid request format: "+err.Error())
		api.HandleErrorWithHttpCode(ctx, 400, err, nil)
		return
	}

	// Set default source if not provided
	if req.Source == "" {
		req.Source = model.CoinSourcePluginDeduct
	}

	response, err := h.coinsService.InternalDeductCoins(ctx, userID, siteID, req)
	if err != nil {
		log.Error(ctx, "Failed to deduct coins: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}
