package handler

import (
	"fmt"
	"net/http"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/middleware"
	"shortpress-server/internal/model"
	"shortpress-server/internal/service/payment/coins"
	"shortpress-server/pkg/log"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CoinsHandler handles coin-related operations
type CoinsHandler struct {
	*Handler
	coinsService coins.CoinsService
}

// NewCoinsHandler creates a new coins handler
func NewCoinsHandler(
	handler *Handler,
	coinsService coins.CoinsService,
) *CoinsHandler {
	return &CoinsHandler{
		Handler:      handler,
		coinsService: coinsService,
	}
}

// GetBalance godoc
// @Summary Get user coin balance
// @Description Get current logged-in user's coin account balance at the specified site
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Success 200 {object} api.Response{data=api.UserCoinsResponse} "Return user coin account information"
// @Router /api/client/payment/user/coins/balance [get]
func (h *CoinsHandler) GetBalance(ctx *gin.Context) {
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

	balance, err := h.coinsService.GetBalance(ctx, userID, siteID)
	if err != nil {
		api.HandleError(ctx, err, "Failed to get coin balance")
		return
	}

	api.HandleSuccess(ctx, balance)
}

// GetTransactionHistory godoc
// @Summary Get user coin transaction records
// @Description Get current logged-in user's coin transaction history records
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Items per page, default 20" minimum(1) maximum(100) example(20)
// @Success 200 {object} api.Response{data=api.CoinTransactionHistoryResponse} "Return transaction record list"
// @Router /api/client/payment/user/coins/transactions [get]
func (h *CoinsHandler) GetAddCoionsHistory(ctx *gin.Context) {
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

	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	response, err := h.coinsService.GetTransactionHistory(ctx, userID, siteID, page, pageSize)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// PkgClientList godoc
// @Summary Get client-side coin package list
// @Description Get all available coin packages for the specified site, for client display
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Success 200 {object} api.CoinPackageResponseData "Return available coin package list"
// @Router /api/client/payment/coins/package/list [get]
func (h *CoinsHandler) PkgClientList(ctx *gin.Context) {
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	packages, err := h.coinsService.ListPackages(ctx, siteID, 1)
	if err != nil {
		api.HandleError(ctx, err, "Failed to list coin packages")
		return
	}

	api.HandleSuccess(ctx, packages)
}

// PkgClientList godoc
// @Summary Get coin package list
// @Description Get all coin packages for the specified site
// @Tags payment
// @Accept json
// @Produce json
// @Param siteId query string true "Site ID"
// @Success 200 {object} api.CoinPackageResponseData "Return available coin package list"
// @Router /api/payment/coins/package/list [get]
func (h *CoinsHandler) ListCoinPackage(ctx *gin.Context) {
	siteID := ctx.Query("siteId")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	packages, err := h.coinsService.ListPackages(ctx, siteID, -1)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, packages)
}

// GetVideoUnlockHistory godoc
// @Summary Get user video unlock records
// @Description Get current logged-in user's video and playlist unlock history records
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Items per page, default 20" minimum(1) maximum(100) example(20)
// @Success 200 {object} api.Response{data=api.VideoUnlockHistoryResponse} "Return unlock record list"
// @Router /api/client/payment/user/coins/videos/transactions [get]
func (h *CoinsHandler) GetVideoUnlockHistory(ctx *gin.Context) {
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

	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	response, err := h.coinsService.GetContentUnlockHistory(ctx, userID, siteID, page, pageSize)
	if err != nil {
		api.HandleError(ctx, err, "Failed to get content unlock history")
		return
	}

	api.HandleSuccess(ctx, response)
}

// BuyVideoWithCoins godoc
// @Summary Purchase video with coins
// @Description Use user's coin account balance to purchase video or playlist viewing rights
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param request body api.BuyContentWithCoinsRequest true "Video purchase request"
// @Success 200 {object} api.Response{data=api.BuyContentWithCoinsResponse} "Return purchase result"
// @Router /api/client/payment/coins/videos/buy [post]
func (h *CoinsHandler) BuyVideoWithCoins(ctx *gin.Context) {
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

	var req api.BuyContentWithCoinsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrBadRequest, err.Error())
		return
	}

	response, err := h.coinsService.BuyContentWithCoins(ctx, userID, siteID, req)
	if err != nil {
		h.logger.Error("Failed to process purchase", zap.Error(err))
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// GrantCoinsToCustomer godoc
// @Summary Creator manually adds user coins
// @Description Allow creators to manually add coins to specific users as rewards or compensation
// @Tags payment
// @Accept json
// @Produce json
// @Param req body api.GrantCoinsRequest true "Grant coins request"
// @Success 200 {object} api.Response{data=api.GrantCoinsResponse} "Return operation result"
// @Router /api/payment/customers/coins/grant [post]
func (h *CoinsHandler) GrantCoinsToCustomer(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.GrantCoinsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	result, err := h.coinsService.GrantCoins(ctx, req.UserEmail, req.SiteID, req.CoinAmount, req.Reason, creatorID)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, result)
}

// GetBalance godoc
// @Summary Get coin information for a specific email user
// @Description Get coin account information for a specific email user
// @Tags payment
// @Accept json
// @Produce json
// @Param userEmail query string true "User email"
// @Param siteId query string true "Site ID"
// @Success 200 {object} api.Response{data=api.UserCoinsResponse} "Return user coin account information"
// @Router /api/payment/customers/coins/balance [get]
func (h *CoinsHandler) GetCustomerCoinsBalance(ctx *gin.Context) {
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
	userEmail := ctx.Query("userEmail")
	if userEmail == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("userEmail is required"), nil)
		return
	}

	balance, err := h.coinsService.GetBalanceByEmail(ctx, userEmail, siteID)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, balance)
}

// ModifyCoinPackage godoc
// @Summary Modify coin package
// @Description Modify existing coin package information
// @Tags payment
// @Accept json
// @Produce json
// @Param req body api.CoinPackageModifyRequest true "Coin package modification request"
// @Success 200 {object} api.CoinPackageModifyResponse "Return successfully modified package ID"
// @Router /api/payment/coins/package [post]
func (h *CoinsHandler) ModifyCoinPackage(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.CoinPackageModifyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	coinPackage := &model.CoinPackage{
		PackageID:    req.PackageID,
		SiteID:       req.SiteID,
		Name:         req.Name,
		Description:  req.Description,
		Features:     req.Features,
		Status:       req.Status,
		IOSProductID: req.IOSProductID,
	}

	err := h.coinsService.UpdateCoinPackage(ctx, coinPackage)
	if err != nil {
		log.Error(ctx, "Failed to update coin package, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	response := &api.CoinPackageModifyResponse{
		PackageID: req.PackageID,
	}

	api.HandleSuccess(ctx, response)
}

// ClaimTaskReward godoc
// @Summary Claim task reward
// @Description Claim coins reward for completing a task. Each task rewards 100 coins and can only be completed once.
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param request body api.ClaimTaskRewardRequest true "Task reward claim request"
// @Success 200 {object} api.Response{data=api.ClaimTaskRewardResponse} "Return claim result"
// @Failure 400 {object} api.Response "Invalid request"
// @Failure 401 {object} api.Response "Unauthorized"
// @Router /api/client/payment/user/coins/claim-task [post]
func (h *CoinsHandler) ClaimTaskReward(ctx *gin.Context) {
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

	var req api.ClaimTaskRewardRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	response, err := h.coinsService.ClaimTaskReward(ctx, userID, siteID, req.TaskName)
	if err != nil {
		log.Error(ctx, "Failed to claim task reward: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// GetWheelStatus godoc
// @Summary Get lucky wheel status
// @Tags client-payment
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Success 200 {object} api.Response{data=api.WheelStatusResponse}
// @Router /api/client/payment/coins/wheel-status [get]
func (h *CoinsHandler) GetWheelStatus(ctx *gin.Context) {
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

	isVIP := false
	response, err := h.coinsService.GetWheelStatus(ctx, userID, siteID, isVIP)
	if err != nil {
		log.Error(ctx, "Failed to get wheel status: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// SpinWheel godoc
// @Summary Spin lucky wheel
// @Tags client-payment
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param request body api.WheelSpinRequest true "Wheel spin request"
// @Success 200 {object} api.Response{data=api.WheelSpinResponse}
// @Router /api/client/payment/coins/wheel-spin [post]
func (h *CoinsHandler) SpinWheel(ctx *gin.Context) {
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

	var req api.WheelSpinRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	isVIP := false
	response, err := h.coinsService.SpinWheel(ctx, userID, siteID, isVIP, req.Mode)
	if err != nil {
		if err == common.ErrInsufficientCoins {
			api.HandleSuccess(ctx, response)
			return
		}
		log.Error(ctx, "Failed to spin wheel: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}
