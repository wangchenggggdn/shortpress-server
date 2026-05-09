package handler

import (
	"errors"
	"net/http"
	"shortpress-server/internal/api"
	"shortpress-server/internal/middleware"
	"shortpress-server/internal/service"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
)

type SitePageConfigHandler struct {
	*Handler
	sitePageConfigService service.SitePageConfigService
}

func NewSitePageConfigHandler(
	handler *Handler,
	sitePageConfigService service.SitePageConfigService,
) *SitePageConfigHandler {
	return &SitePageConfigHandler{
		Handler:               handler,
		sitePageConfigService: sitePageConfigService,
	}
}

// Create godoc
// @Summary Create a site page config
// @Description Create a new site page config or update if exists with same site_id and type
// @Tags site-page-config
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.SitePageConfigCreateRequest true "Request body"
// @Success 200 {object} api.Response{data=api.SitePageConfigCreateResponse} "Returns config ID"
// @Router /api/site/page-config/create [post]
func (h *SitePageConfigHandler) Create(ctx *gin.Context) {
	var req api.SitePageConfigCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error(ctx, "bind request failed: "+err.Error())
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, errors.New("siteId not found in context"), nil)
		return
	}

	id, err := h.sitePageConfigService.Create(ctx, siteID, &req)
	if err != nil {
		log.Error(ctx, "create site page config failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, &api.SitePageConfigCreateResponse{ID: id})
}

// Update godoc
// @Summary Update a site page config
// @Description Update an existing site page config by site_id and type
// @Tags site-page-config
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.SitePageConfigUpdateRequest true "Request body"
// @Success 200 {object} api.Response{data=api.SitePageConfigUpdateResponse} "Returns config ID"
// @Router /api/site/page-config/update [post]
func (h *SitePageConfigHandler) Update(ctx *gin.Context) {
	var req api.SitePageConfigUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error(ctx, "bind request failed: "+err.Error())
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, errors.New("siteId not found in context"), nil)
		return
	}

	id, err := h.sitePageConfigService.Update(ctx, siteID, &req)
	if err != nil {
		log.Error(ctx, "update site page config failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, &api.SitePageConfigUpdateResponse{ID: id})
}

// Get godoc
// @Summary Get a site page config
// @Description Get a site page config by type (siteId from X-Site-Id header)
// @Tags site-page-config
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param X-Site-Id header string true "Site ID"
// @Param type query string true "Page type"
// @Success 200 {object} api.Response{data=api.SitePageConfigItem} "Returns config"
// @Router /api/site/page-config/get [get]
func (h *SitePageConfigHandler) Get(ctx *gin.Context) {
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, errors.New("siteId not found in context"), nil)
		return
	}

	pageType := ctx.Query("type")
	if pageType == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, errors.New("type is required"), nil)
		return
	}

	config, err := h.sitePageConfigService.GetBySiteIDAndType(ctx, siteID, pageType)
	if err != nil {
		log.Error(ctx, "get site page config failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	if config == nil {
		api.HandleSuccess(ctx, nil)
		return
	}

	api.HandleSuccess(ctx, &api.SitePageConfigItem{
		ID:     config.ID,
		SiteID: config.SiteID,
		Type:   config.Type,
		Config: config.Config,
	})
}

// List godoc
// @Summary List site page configs
// @Description List all site page configs for a site (siteId from X-Site-Id header)
// @Tags site-page-config
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param X-Site-Id header string true "Site ID"
// @Success 200 {object} api.Response{data=api.SitePageConfigListResponse} "Returns config list"
// @Router /api/site/page-config/list [get]
func (h *SitePageConfigHandler) List(ctx *gin.Context) {
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, errors.New("siteId not found in context"), nil)
		return
	}

	configs, _, err := h.sitePageConfigService.ListBySiteID(ctx, siteID)
	if err != nil {
		log.Error(ctx, "list site page config failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	items := make([]*api.SitePageConfigItem, 0, len(configs))
	for _, config := range configs {
		items = append(items, &api.SitePageConfigItem{
			ID:     config.ID,
			SiteID: config.SiteID,
			Type:   config.Type,
			Config: config.Config,
		})
	}

	api.HandleSuccess(ctx, &api.SitePageConfigListResponse{
		Items: items,
		Total: int64(len(items)),
	})
}
