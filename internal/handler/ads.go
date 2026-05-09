package handler

import (
	"fmt"
	"net/http"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/middleware"
	"shortpress-server/internal/service"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
)

// AdsHandler handles requests related to advertisements
type AdsHandler struct {
	*Handler
	adsService service.AdsService
}

// NewAdsHandler creates a new ads handler
func NewAdsHandler(
	handler *Handler,
	adsService service.AdsService,
) *AdsHandler {
	return &AdsHandler{
		Handler:    handler,
		adsService: adsService,
	}
}

// UnitList godoc
// @Summary List advertising units
// @Schemes
// @Description Get a list of advertising units
// @Tags ads
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param siteId query string true "Site ID" example("site123")
// @Success 200 {object} api.AdUnitListResponseData "Returns a list of ad units"
// @Router /api/ads/unit/list [get]
func (h *AdsHandler) UnitList(ctx *gin.Context) {
	// 1. Validate user authentication
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// 2. Get parameters
	siteID := ctx.Query("siteId")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteID is required"), nil)
		return
	}

	// 3. Call service layer to get ad unit list
	adUnits, err := h.adsService.ListAdUnits(ctx, siteID)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("failed to list ad units, siteID: %s, error: %v", siteID, err))
		api.HandleError(ctx, err, nil)
		return
	}

	// 4. Return success response
	responseData := api.AdUnitListResponseData{
		Items: adUnits,
	}

	api.HandleSuccess(ctx, responseData)
}

// CreateAdUnit godoc
// @Summary Create a new advertising unit
// @Schemes
// @Description Create a new advertising unit with specified configuration
// @Tags ads
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.AdUnit true "Ad unit creation parameters"
// @Success 200 {object} api.Response "Returns the created ad unit information"
// @Router /api/ads/unit/create [post]
func (h *AdsHandler) CreateAdUnit(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.AdUnit
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	_, err := h.adsService.CreateAdUnit(ctx, req)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to create ad unit for site %s: %v", req.SiteID, err))
		api.HandleError(ctx, err, nil)
		return
	}

	// 5. Return success response
	api.HandleSuccess(ctx, nil)
}

// UnitModify godoc
// @Summary Modify an ad unit
// @Schemes
// @Description Modify an existing advertisement unit
// @Tags ads
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer creator token"
// @Param request body api.AdUnit true "Ad unit modification parameters"
// @Success 200 {object} api.Response "Success response"
// @Router /api/ads/unit/modify [post]
func (h *AdsHandler) UnitModify(ctx *gin.Context) {
	var req api.AdUnit
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	err := h.adsService.ModifyAdUnit(ctx, req)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to modify ad unit: %v", err))
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, nil)
}

// UnitConf godoc
// @Summary Get ad unit configuration
// @Schemes
// @Description Retrieve ad unit configuration for a specific location in a site
// @Tags ads
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param location query string true "Ad location identifier" example("feed_top")
// @Success 200 {object} api.AdUnitConf "Returns ad unit configuration"
// @Failure 400 {object} api.Response "Bad request"
// @Failure 404 {object} api.Response "Ad unit not found"
// @Failure 500 {object} api.Response "Internal server error"
// @Router /api/ads/unit/conf [get]
func (h *AdsHandler) UnitConf(ctx *gin.Context) {
	// 1. Get site ID from the context (set by middleware)
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("missing site ID"), nil)
		return
	}

	// 2. Get location parameter
	location := ctx.Query("location")
	if location == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("missing location parameter"), nil)
		return
	}

	// 3. Call service to get ad unit configuration
	adConf, err := h.adsService.GetAdUnitConf(ctx, siteID, location)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to get ad unit configuration for site %s, location %s: %v", siteID, location, err))
		api.HandleError(ctx, err, nil)
		return
	}

	// 4. Return the configuration
	api.HandleSuccess(ctx, adConf)
}
