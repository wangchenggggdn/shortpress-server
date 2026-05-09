package handler

import (
	"net/http"
	"shortpress-server/pkg/translate"
	"strconv"

	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/service"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
)

type PagesBuilderHandler struct {
	*Handler
	translator          translate.Translator
	pagesBuilderService service.PagesBuilderService
}

func NewPagesBuilderHandler(
	handler *Handler,
	pagesBuilderService service.PagesBuilderService,
) *PagesBuilderHandler {
	return &PagesBuilderHandler{
		Handler:             handler,
		pagesBuilderService: pagesBuilderService,
	}
}

// SavePagesBuilderData godoc
// @Summary Save pages builder data
// @Schemes
// @Description Save complete site builder data including pages, sections, and configuration to the draft state
// @Tags pages-builder
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.SavePagesBuilderDataRequest true "Pages builder data to save"
// @Success 200 {object} api.Response{data=api.SavePagesBuilderDataResponse} "Returns successful result of saving pages builder data"
// @Router /api/pages-builder/save [post]
func (h *PagesBuilderHandler) SavePagesBuilderData(ctx *gin.Context) {

	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.SavePagesBuilderDataRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	response, err := h.pagesBuilderService.SavePagesBuilderData(ctx, creatorID, &req)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// GetPagesBuilderData godoc
// @Summary Get pages builder data
// @Schemes
// @Description Get the complete site builder data including site_data and version information from site_builder_data table
// @Tags pages-builder
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param siteId query string true "Site ID" example("550e8400-e29b-41d4-a716-446655440000")
// @Success 200 {object} api.Response{data=api.GetPagesBuilderDataResponse} "Returns site builder data with version information"
// @Router /api/pages-builder/info [get]
func (h *PagesBuilderHandler) GetPagesBuilderData(ctx *gin.Context) {
	// creatorID := ctx.GetString("creator_id")
	// if creatorID == "" {
	// 	api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
	// 	return
	// }

	siteID := ctx.Query("siteId")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrSiteNotFound, nil)
		return
	}
	builderData, err := h.pagesBuilderService.GetPagesBuilderData(ctx, "", siteID)
	if err != nil {
		log.Error(ctx, "get pages builder data failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, builderData)
}

// PublishPagesBuilderData godoc
// @Summary Publish pages builder data
// @Schemes
// @Description Publish the current site builder data from site_builder_data table to site_current_published and site_published_history tables, making it live
// @Tags pages-builder
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.PublishPagesBuilderDataRequest true "Publish request with site ID"
// @Success 200 {object} api.Response{data=api.PublishPagesBuilderDataResponse} "Returns published data confirmation with version and timestamp"
// @Router /api/pages-builder/publish [post]
func (h *PagesBuilderHandler) PublishPagesBuilderData(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.PublishPagesBuilderDataRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	response, err := h.pagesBuilderService.PublishPagesBuilderData(ctx, creatorID, &req)
	if err != nil {
		log.Error(ctx, "publish pages builder data failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// GetPublishHistory godoc
// @Summary Get publish history
// @Schemes
// @Description Get the publish history of a site from site_published_history table with pagination and current version indication
// @Tags pages-builder
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param site_id query string true "Site ID" example("550e8400-e29b-41d4-a716-446655440000")
// @Param limit query int false "Limit for pagination (max 100, default 10)" example(10)
// @Param offset query int false "Offset for pagination (default 0)" example(0)
// @Success 200 {object} api.Response{data=api.GetPublishHistoryResponse} "Returns publish history with pagination and current version indication"
// @Router /api/pages-builder/publish/history [get]
func (h *PagesBuilderHandler) GetPublishHistory(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.GetPublishHistoryRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	response, err := h.pagesBuilderService.GetPublishHistory(ctx, creatorID, &req)
	if err != nil {
		log.Error(ctx, "get publish history failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// TemplatesList godoc
// @Summary Get page templates list
// @Schemes
// @Description List available page templates for site creation or customization
// @Tags pages-builder
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(10)
// @Success 200 {object} api.Response{data=api.TemplateListResponseData} "Returns page templates list"
// @Router /api/pages-builder/templates [get]
func (h *PagesBuilderHandler) TemplatesList(ctx *gin.Context) {
	page := 1
	pageSize := 10
	if v := ctx.Query("page"); v != "" {
		if pv, err := strconv.Atoi(v); err == nil && pv > 0 {
			page = pv
		}
	}
	if v := ctx.Query("pageSize"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	data, err := h.pagesBuilderService.ListTemplates(ctx, page, pageSize)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}
	api.HandleSuccess(ctx, data)
}

// TranslatePages godoc
// @Summary Translate pages
// @Schemes
// @Description Translate page content to different languages
// @Tags pages-builder
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.PageTranslateRequest true "Translate request with site ID and site data"
// @Success 200 {object} api.Response{data=api.PageTranslateResponse} "Returns translated page content"
// @Router /api/pages-builder/translate [post]
func (h *PagesBuilderHandler) TranslatePages(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}
	var req api.PageTranslateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	data, err := h.pagesBuilderService.TranslatePages(ctx, &req)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}
	api.HandleSuccess(ctx, data)
}
