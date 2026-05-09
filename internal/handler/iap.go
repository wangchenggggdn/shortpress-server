package handler

import (
	"net/http"
	"shortpress-server/internal/api"
	"shortpress-server/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthClaims struct {
	UserId    string
	CreatorId string
	jwt.RegisteredClaims
}

type IapHandler struct {
	*Handler
	paymentBiz service.PaymentBiz
}

func NewIapHandler(
	handler *Handler,
	paymentBiz service.PaymentBiz,
) *IapHandler {
	return &IapHandler{
		Handler:    handler,
		paymentBiz: paymentBiz,
	}
}

// 验证订阅状态
func (h *IapHandler) VerifySub(ctx *gin.Context) {
	// 绑定并验证请求参数
	var req service.PaymentVerifyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	if ctx.GetHeader("Authorization") == "" {
		h.logger.Error("Missing Authorization header")
		return
	}
	if ctx.GetHeader("X-Site-Id") == "" {
		h.logger.Error("Missing X-Site-Id header")
		return
	}

	req.SiteID = ctx.GetHeader("X-Site-Id")
	userId := ctx.GetString("user_id")
	req.UserID = userId

	balance, err := h.paymentBiz.VerifyAndSyncSubStatus(ctx, &req)
	if err != nil {
		api.HandleError(ctx, err, "Failed to get coin balance")
		return
	}

	api.HandleSuccess(ctx, balance)
}

// 验证应用内购买
func (h *IapHandler) Verify(ctx *gin.Context) {
	h.logger.Error("Missing X-Site-Id header")
	// 绑定并验证请求参数
	var req service.PaymentVerifyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Missing Authorization header")
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	if ctx.GetHeader("Authorization") == "" {
		h.logger.Error("Missing Authorization header")
		return
	}
	if ctx.GetHeader("X-Site-Id") == "" {
		h.logger.Error("Missing X-Site-Id header")
		return
	}

	req.SiteID = ctx.GetHeader("X-Site-Id")
	userId := ctx.GetString("user_id")
	req.UserID = userId
	h.logger.Error("VerifyInAppPurchase")
	balance, err := h.paymentBiz.VerifyInAppPurchase(ctx, &req)
	if err != nil {
		//h.logger.Error("VerifyInAppPurchase failed", "err", err, "userID", userID, "req", req)
		api.HandleError(ctx, err, "Failed to get coin balance")
		return
	}

	api.HandleSuccess(ctx, balance)
}

// 处理支付通知
func (h *IapHandler) Notify(ctx *gin.Context) {
	data, _ := ctx.GetRawData()
	account := ctx.Query("account")
	balance, err := h.paymentBiz.Notify(ctx, &service.PaymentNotifyRequest{
		Account: account,
		Body:    data,
	})
	if err != nil {
		api.HandleError(ctx, err, "Failed to get coin balance")
		return
	}

	api.HandleSuccess(ctx, balance)
}
