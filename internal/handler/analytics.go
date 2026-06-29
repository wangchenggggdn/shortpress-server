package handler

import (
	"fmt"
	"net/http"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/service"
	"shortpress-server/pkg/log"
	"time"

	"github.com/gin-gonic/gin"
)

// AnalyticsHandler handles analytics-related operations
type AnalyticsHandler struct {
	*Handler
	analyticsService service.AnalyticsService
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(
	handler *Handler,
	analyticsService service.AnalyticsService,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		Handler:          handler,
		analyticsService: analyticsService,
	}
}

// IncomeTransactions godoc
// @Summary Get site payment transaction records
// @Description Get site payment transaction records by time range
// @Tags analytics
// @Accept json
// @Produce json
// @Param request body api.IncomeTransactionsRequest true "Query parameters"
// @Success 200 {object} api.Response{data=api.IncomeTransactionHistoryResponse} "Return transaction records"
// @Router /api/analytics/income/transactions [post]
func (h *AnalyticsHandler) IncomeTransactions(ctx *gin.Context) {
	// Verify creator authentication
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// Parse request body
	var req api.IncomeTransactionsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// Add business context
	log.AddNotice(ctx, "operation", "query_income_transactions")

	// Set default pagination values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// Set default time range if not provided
	// If no end time, use current time
	if req.EndTime == 0 {
		req.EndTime = time.Now().Unix()
	}
	// No need to set default start time, as 0 means no lower bound

	// Convert timestamps to time objects
	startTime := time.Unix(req.StartTime, 0)
	endTime := time.Unix(req.EndTime, 0)

	// Call service to get transaction history
	response, err := h.analyticsService.GetPaymentTransactions(
		ctx,
		req.SiteID,
		req.UserID,
		req.UserEmail,
		startTime,
		endTime,
		req.Page,
		req.PageSize,
	)

	if err != nil {
		log.Error(ctx, fmt.Sprintf("Service error in income transactions query: %v", err))
		api.HandleError(ctx, common.ErrInternalServerError, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// IncomeTransactionsInfo godoc
// @Summary Get transaction record details
// @Description Get detailed information for a specific transaction ID
// @Tags analytics
// @Accept json
// @Produce json
// @Param transactionId query string true "Transaction ID" example("trans-123456")
// @Success 200 {object} api.Response{data=api.IncomeTransactionDetailResponse} "Return transaction details"
// @Router /api/analytics/income/transactions/info [get]
func (h *AnalyticsHandler) IncomeTransactionsInfo(ctx *gin.Context) {
	// Verify creator authentication
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// Get transaction ID from query params
	transactionID := ctx.Query("transactionId")
	if transactionID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("transactionId is required"), nil)
		return
	}

	// Call service to get transaction details
	transaction, err := h.analyticsService.GetTransactionByID(ctx, transactionID)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Service error in income transaction details query: %v", err))
		api.HandleError(ctx, err, nil)
		return
	}

	if transaction == nil {
		log.Error(ctx, fmt.Sprintf("Transaction not found for ID: %s", transactionID))
		api.HandleError(ctx, common.ErrNotFound, "Transaction not found")
		return
	}

	api.HandleSuccess(ctx, transaction)
}

// IncomStatistics godoc
// @Summary Get site income statistics
// @Description Statistics of site transaction income by day
// @Tags analytics
// @Accept json
// @Produce json
// @Param request body api.IncomeStatisticsRequest true "Query parameters"
// @Success 200 {object} api.Response{data=api.IncomeStatisticsResponse} "Return income statistics data by day"
// @Router /api/analytics/income/statistics [post]
func (h *AnalyticsHandler) IncomStatistics(ctx *gin.Context) {
	// Verify creator authentication
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// Parse request body
	var req api.IncomeStatisticsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// Set default time range if not provided
	// If no end time, use current time
	if req.EndTime == 0 {
		req.EndTime = time.Now().Unix()
	}
	// No need to set default start time, as 0 means no lower bound

	// Convert timestamps to time objects
	startTime := time.Unix(req.StartTime, 0)
	endTime := time.Unix(req.EndTime, 0)

	// Call service to get income statistics
	response, err := h.analyticsService.GetIncomeStatistics(
		ctx,
		req.SiteID,
		startTime,
		endTime,
	)

	if err != nil {
		log.Error(ctx, fmt.Sprintf("Service error in income statistics query: %v", err))
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}
