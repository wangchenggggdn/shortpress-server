package handler

import (
	"fmt"
	"net/http"

	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/middleware"
	"shortpress-server/internal/service"

	"github.com/gin-gonic/gin"
)

type A2EHandler struct {
	*Handler
	a2eService service.A2EService
}

func NewA2EHandler(
	handler *Handler,
	a2eService service.A2EService,
) *A2EHandler {
	return &A2EHandler{
		Handler:    handler,
		a2eService: a2eService,
	}
}

// InvokeWan27 godoc
// @Summary Invoke A2E wan2.7 generation
// @Schemes
// @Description Start an A2E wan2.7 generation task through the generation service
// @Tags a2e
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param X-Site-Id header string true "Site ID"
// @Param request body api.A2EWan27InvokeRequest true "wan2.7 invoke request"
// @Success 200 {object} api.Response{data=api.A2EWan27InvokeResponse} "Returns generation task info"
// @Router /api/a2e/wan2_7/invoke [post]
func (h *A2EHandler) InvokeWan27(ctx *gin.Context) {
	if ctx.GetString("user_id") == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}
	if middleware.GetSiteID(ctx) == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	var req api.A2EWan27InvokeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	resp, err := h.a2eService.InvokeWan27(ctx, &req)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}
	api.HandleSuccess(ctx, resp)
}
