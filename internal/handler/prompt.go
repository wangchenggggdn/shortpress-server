package handler

import (
	"net/http"
	"strings"

	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/service"

	"github.com/gin-gonic/gin"
)

type PromptHandler struct {
	*Handler
	promptService service.PromptService
}

func NewPromptHandler(
	handler *Handler,
	promptService service.PromptService,
) *PromptHandler {
	return &PromptHandler{
		Handler:       handler,
		promptService: promptService,
	}
}

// Optimize godoc
// @Summary Optimize prompt
// @Schemes
// @Description Optimize user prompt content through Grok
// @Tags prompt
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.PromptOptimizeRequest true "Prompt optimization request"
// @Success 200 {object} api.Response{data=api.PromptOptimizeResponse} "Returns optimized prompt"
// @Router /api/prompt/optimize [post]
func (h *PromptHandler) Optimize(ctx *gin.Context) {
	if ctx.GetString("user_id") == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.PromptOptimizeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}
	if strings.TrimSpace(req.UserInput) == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrBadRequest, nil)
		return
	}

	resp, err := h.promptService.OptimizePrompt(ctx, &req)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, resp)
}
