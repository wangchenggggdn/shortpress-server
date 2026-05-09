package action

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"time"
)

// ==================== 类型定义 ====================

// CreateOrderArgs 创建订单参数
type CreateOrderArgs struct {
	Amount         int    `json:"amount"`           // 金额（必填，正整数）
	Type           string `json:"type"`             // 类型：add 或 deduct（必填）
	Reason         string `json:"reason"`           // 原因（可选）
	ThirdPartyID   string `json:"third_party_id"`   // 第三方订单ID（可选，如充值平台订单号）
	ThirdPartyName string `json:"third_party_name"` // 第三方平台名称（可选，如微信、支付宝、Stripe等）
}

// AddCoinsArgs 加点参数
type AddCoinsArgs struct {
	OrderID string `json:"order_id"` // 订单ID（必填）
}

// DeductCoinsArgs 扣点参数
type DeductCoinsArgs struct {
	OrderID string `json:"order_id"` // 订单ID（必填）
}

// CoinResponse 支付响应
type CoinResponse struct {
	TransactionID string `json:"transaction_id"` // 交易ID
	BeforeBalance int    `json:"before_balance"` // 操作前余额
	AfterBalance  int    `json:"after_balance"`  // 操作后余额
	Amount        int    `json:"amount"`         // 金额
	Type          string `json:"type"`           // 类型：add 或 deduct
}

// ==================== 常量定义 ====================

const (
	// 加点上限
	MaxAddCoinsAmount = 100000
	// 扣点上限
	MaxDeductCoinsAmount = 10000
)

// ==================== Action 处理器 ====================

// CreateOrder 创建订单
func (a *Action) CreateOrder(ctx *gin.Context, args json.RawMessage) (json.RawMessage, error) {
	// 1. 提取认证信息
	userID, siteID, pluginID, err := a.extractContext(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 解析参数
	var params CreateOrderArgs
	if err = json.Unmarshal(args, &params); err != nil {
		return nil, common.ErrBadRequest
	}

	// 3. 验证参数
	if err = a.validateCreateOrderParams(params); err != nil {
		return nil, err
	}

	// 4. 验证用户归属
	if err = a.validateUserBelonging(ctx, userID, siteID); err != nil {
		return nil, err
	}

	// 5. 生成订单
	orderID, err := a.executeCreateOrder(ctx, userID, siteID, pluginID, params)
	if err != nil {
		return nil, err
	}

	// 6. 构建响应
	response := map[string]interface{}{
		"order_id": orderID,
		"amount":   params.Amount,
		"type":     params.Type,
	}

	return json.Marshal(response)
}

// AddCoins 增加点数
func (a *Action) AddCoins(ctx *gin.Context, args json.RawMessage) (json.RawMessage, error) {
	// 1. 提取认证信息
	userID, siteID, _, err := a.extractContext(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 解析参数
	var params AddCoinsArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, common.ErrBadRequest
	}

	// 3. 验证参数
	if err := a.validateAddCoinsParams(params); err != nil {
		return nil, err
	}

	// 4. 验证订单并执行加点
	beforeBalance, afterBalance, transactionID, err := a.executeAddCoinsByOrder(ctx, userID, siteID, params.OrderID)
	if err != nil {
		return nil, err
	}

	// 5. 构建响应
	response := CoinResponse{
		TransactionID: transactionID,
		BeforeBalance: beforeBalance,
		AfterBalance:  afterBalance,
		Amount:        afterBalance - beforeBalance,
		Type:          "add",
	}

	return json.Marshal(response)
}

// DeductCoins 扣除点数
func (a *Action) DeductCoins(ctx *gin.Context, args json.RawMessage) (json.RawMessage, error) {
	// 1. 提取认证信息
	userID, siteID, _, err := a.extractContext(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 解析参数
	var params DeductCoinsArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, common.ErrBadRequest
	}

	// 3. 验证参数
	if err := a.validateDeductCoinsParams(params); err != nil {
		return nil, err
	}

	// 4. 验证订单并执行扣点
	beforeBalance, afterBalance, transactionID, err := a.executeDeductCoinsByOrder(ctx, userID, siteID, params.OrderID)
	if err != nil {
		return nil, err
	}

	// 5. 构建响应
	response := CoinResponse{
		TransactionID: transactionID,
		BeforeBalance: beforeBalance,
		AfterBalance:  afterBalance,
		Amount:        beforeBalance - afterBalance,
		Type:          "deduct",
	}

	return json.Marshal(response)
}

// ==================== Context 提取 ====================

// extractContext 从 gin.Context 中提取认证信息
func (a *Action) extractContext(ctx *gin.Context) (userID, siteID, pluginID string, err error) {
	userID = ctx.GetString("user_id")
	siteID = ctx.GetString("site_id")
	pluginID = ctx.GetString("plugin_id")

	if userID == "" || siteID == "" || pluginID == "" {
		return "", "", "", errors.New("missing user_id, site_id or plugin_id in context")
	}

	return userID, siteID, pluginID, nil
}

// ==================== 参数验证 ====================

// validateCreateOrderParams 验证创建订单参数
func (a *Action) validateCreateOrderParams(params CreateOrderArgs) error {
	if params.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if params.Type != "add" && params.Type != "deduct" {
		return errors.New("type must be 'add' or 'deduct'")
	}

	// 根据类型验证金额上限
	if params.Type == "add" && params.Amount > MaxAddCoinsAmount {
		return fmt.Errorf("amount exceeds maximum limit of %d", MaxAddCoinsAmount)
	}
	if params.Type == "deduct" && params.Amount > MaxDeductCoinsAmount {
		return fmt.Errorf("amount exceeds maximum limit of %d", MaxDeductCoinsAmount)
	}

	return nil
}

// validateAddCoinsParams 验证加点参数
func (a *Action) validateAddCoinsParams(params AddCoinsArgs) error {
	if params.OrderID == "" {
		return errors.New("order_id is required")
	}
	return nil
}

// validateDeductCoinsParams 验证扣点参数
func (a *Action) validateDeductCoinsParams(params DeductCoinsArgs) error {
	if params.OrderID == "" {
		return errors.New("order_id is required")
	}
	return nil
}

// validateUserBelonging 验证用户归属关系
func (a *Action) validateUserBelonging(ctx *gin.Context, userID, siteID string) error {
	user, err := a.userRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return common.ErrUserNotFound
	}
	if user.SiteID != siteID {
		return errors.New("user site_id mismatch")
	}
	return nil
}

// ==================== 订单验证 ====================

// validateAndLockOrder 验证订单并锁定（防止重复处理）
func (a *Action) validateAndLockOrder(ctx *gin.Context, orderID string, expectedType string) (*model.PluginOrder, error) {
	// 1. 获取待处理的订单
	order, err := a.pluginOrderRepo.GetPendingByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("order not found or already processed")
	}

	// 2. 验证订单类型
	if order.Type != expectedType {
		return nil, fmt.Errorf("order type mismatch: expected %s, got %s", expectedType, order.Type)
	}

	// 3. 验证用户归属（从 context 中获取的用户信息必须与订单一致）
	userID := ctx.GetString("user_id")
	siteID := ctx.GetString("site_id")
	if userID != order.UserID || siteID != order.SiteID {
		return nil, errors.New("order user mismatch")
	}

	return order, nil
}

// ==================== 执行逻辑 ====================

// executeCreateOrder 执行创建订单
func (a *Action) executeCreateOrder(ctx *gin.Context, userID, siteID, pluginID string, params CreateOrderArgs) (orderID string, err error) {
	// 1. 针对add类型进行防重放检查（只对 add 类型防重放，deduct 不需要）
	if params.Type == "add" {
		recentOrder, err := a.pluginOrderRepo.CheckRecentAddOrder(ctx, userID, siteID, params.Amount, 10*time.Second)
		if err != nil {
			return "", fmt.Errorf("failed to check recent orders: %w", err)
		}
		if recentOrder != nil {
			// 发现重复add订单，返回已存在的订单ID，不创建新订单
			return recentOrder.OrderID, fmt.Errorf("duplicate add order detected: recent add order with same amount found (order_id: %s)", recentOrder.OrderID)
		}
	}

	// 2. 生成订单ID
	orderID = fmt.Sprintf("plugin_order_%s", uuid.New().String())

	// 3. 构建订单
	order := &model.PluginOrder{
		OrderID:        orderID,
		UserID:         userID,
		SiteID:         siteID,
		PluginID:       pluginID,
		ThirdPartyID:   params.ThirdPartyID,
		ThirdPartyName: params.ThirdPartyName,
		Amount:         params.Amount,
		Type:           params.Type,
		Status:         model.PluginOrderStatusPending,
		Reason:         params.Reason,
	}

	// 4. 创建订单
	err = a.pluginOrderRepo.Create(ctx, order)
	if err != nil {
		return "", fmt.Errorf("failed to create order: %w", err)
	}

	return orderID, nil
}

// executeAddCoinsByOrder 根据订单执行加点操作
func (a *Action) executeAddCoinsByOrder(ctx *gin.Context, userID, siteID, orderID string) (beforeBalance, afterBalance int, transactionID string, err error) {
	// 1. 验证并锁定订单
	order, err := a.validateAndLockOrder(ctx, orderID, "add")
	if err != nil {
		return 0, 0, "", err
	}

	// 2. 查询当前余额
	userCoins, err := a.userCoinsRepo.GetByUserAndSite(ctx, userID, siteID)
	if err != nil {
		return 0, 0, "", err
	}
	if userCoins != nil {
		beforeBalance = userCoins.Present + userCoins.Balance
	}

	// 3. 更新余额
	userCoins, err = a.userCoinsRepo.UpdateBalance(ctx, userID, siteID, order.Amount, 0)
	if err != nil {
		// 标记订单为失败
		_ = a.pluginOrderRepo.UpdateStatus(ctx, orderID, model.PluginOrderStatusFailed)
		return 0, 0, "", err
	}
	afterBalance = userCoins.Present + userCoins.Balance

	// 4. 构建交易记录
	transactionID = uuid.New().String()
	coinTx := &model.CoinTransaction{
		TransactionID: transactionID,
		UserID:        userID,
		SiteID:        siteID,
		Amount:        order.Amount,
		BeforeBalance: beforeBalance,
		Balance:       afterBalance,
		Source:        model.CoinSourcePluginAdd,
		RelatedID:     order.OrderID,
		RelatedType:   model.CoinRelatedTypePlugin,
		Description:   order.Reason,
		Snapshot: model.JSONMap{
			"order_id":  order.OrderID,
			"plugin_id": order.PluginID,
		},
	}

	// 5. 创建交易记录
	err = a.coinTransactionRepo.Create(ctx, coinTx)
	if err != nil {
		// 标记订单为失败
		_ = a.pluginOrderRepo.UpdateStatus(ctx, orderID, model.PluginOrderStatusFailed)
		return 0, 0, "", fmt.Errorf("failed to create coin transaction: %w", err)
	}

	// 6. 标记订单为已完成
	err = a.pluginOrderRepo.MarkAsProcessed(ctx, orderID)
	if err != nil {
		// 交易已经创建，但订单标记失败，记录日志但不返回错误
		// TODO: 记录日志
	}

	return beforeBalance, afterBalance, transactionID, nil
}

// executeDeductCoinsByOrder 根据订单执行扣点操作
func (a *Action) executeDeductCoinsByOrder(ctx *gin.Context, userID, siteID, orderID string) (beforeBalance, afterBalance int, transactionID string, err error) {
	// 1. 验证并锁定订单
	order, err := a.validateAndLockOrder(ctx, orderID, "deduct")
	if err != nil {
		return 0, 0, "", err
	}

	// 2. 查询当前余额
	userCoins, err := a.userCoinsRepo.GetByUserAndSite(ctx, userID, siteID)
	if err != nil {
		return 0, 0, "", err
	}
	if userCoins == nil {
		// 标记订单为失败
		_ = a.pluginOrderRepo.UpdateStatus(ctx, orderID, model.PluginOrderStatusFailed)
		return 0, 0, "", common.ErrInsufficientCoins
	}
	beforeBalance = userCoins.Present + userCoins.Balance

	// 3. 检查余额充足性（present + balance）
	totalCoins := userCoins.Present + userCoins.Balance
	if totalCoins < order.Amount {
		// 标记订单为失败
		_ = a.pluginOrderRepo.UpdateStatus(ctx, orderID, model.PluginOrderStatusFailed)
		return 0, 0, "", common.ErrInsufficientCoins
	}

	// 4. 更新余额（扣点，优先扣除 present）
	userCoins, err = a.userCoinsRepo.DeductCoins(ctx, userID, siteID, order.Amount)
	if err != nil {
		// 标记订单为失败
		_ = a.pluginOrderRepo.UpdateStatus(ctx, orderID, model.PluginOrderStatusFailed)
		return 0, 0, "", err
	}
	afterBalance = userCoins.Present + userCoins.Balance

	// 5. 构建交易记录
	transactionID = uuid.New().String()
	coinTx := &model.CoinTransaction{
		TransactionID: transactionID,
		UserID:        userID,
		SiteID:        siteID,
		Amount:        -order.Amount,
		BeforeBalance: beforeBalance,
		Balance:       afterBalance,
		Source:        model.CoinSourcePluginDeduct,
		RelatedID:     order.OrderID,
		RelatedType:   model.CoinRelatedTypePlugin,
		Description:   order.Reason,
		Snapshot: model.JSONMap{
			"order_id":  order.OrderID,
			"plugin_id": order.PluginID,
		},
	}

	// 6. 创建交易记录
	err = a.coinTransactionRepo.Create(ctx, coinTx)
	if err != nil {
		// 标记订单为失败
		_ = a.pluginOrderRepo.UpdateStatus(ctx, orderID, model.PluginOrderStatusFailed)
		return 0, 0, "", fmt.Errorf("failed to create coin transaction: %w", err)
	}

	// 7. 标记订单为已完成
	err = a.pluginOrderRepo.MarkAsProcessed(ctx, orderID)
	if err != nil {
		// 交易已经创建，但订单标记失败，记录日志但不返回错误
		// TODO: 记录日志
	}

	return beforeBalance, afterBalance, transactionID, nil
}
