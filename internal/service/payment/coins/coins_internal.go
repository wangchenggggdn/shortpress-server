package coins

import (
	"fmt"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// InternalGetBalance gets user coin balance for internal service-to-service calls
func (s *coinsService) InternalGetBalance(ctx *gin.Context, userID, siteID string) (*api.InternalGetBalanceResponse, error) {
	// Get user coin account
	userCoins, err := s.userCoinsRepo.GetByUserAndSite(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}

	// Return balance (0 if account doesn't exist)
	balance := 0
	if userCoins != nil {
		balance = userCoins.Present + userCoins.Balance
	}

	return &api.InternalGetBalanceResponse{
		Balance: balance,
	}, nil
}

// InternalAddCoins adds coins to user account (treated as coin package purchase)
func (s *coinsService) InternalAddCoins(ctx *gin.Context, userID, siteID string, req api.InternalAddCoinsRequest) (*api.InternalAddCoinsResponse, error) {
	// Get or create user coin account
	userCoin, err := s.userCoinsRepo.GetByUserAndSite(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}

	// Record balance before transaction
	beforeBalance := 0
	if userCoin != nil {
		beforeBalance = userCoin.Balance
	}

	// Update user coin balance using UpdateBalance method
	userCoin, err = s.userCoinsRepo.UpdateBalance(ctx, userID, siteID, req.CoinAmount, req.Amount)
	if err != nil {
		return nil, err
	}

	// Create payment transaction record (like coin package purchase)
	transactionID := req.TransactionID
	if transactionID == "" {
		transactionID = fmt.Sprintf("txn_%s", uuid.New().String())
	}

	snapshot := make(model.JSONMap)
	snapshot["coin_amount"] = req.CoinAmount
	if req.Description != "" {
		snapshot["description"] = req.Description
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	provider := req.Provider
	if provider == "" {
		provider = "internal"
	}

	paymentTransaction := &model.PaymentTransaction{
		TransactionID: transactionID,
		OrderID:       fmt.Sprintf("order_%s", uuid.New().String()),
		UserID:        userID,
		SiteID:        siteID,
		Amount:        req.Amount,
		Currency:      currency,
		Provider:      provider,
		PaymentType:   model.PaymentTypeGrantByAdmin,
		Status:        model.PaymentStatusSuccess,
		RelatedID:     userID,
		RelatedType:   model.RelatedTypeGrantByAdmin,
		Snapshot:      snapshot,
	}

	if err := s.paymentTransactionRepository.Create(ctx, paymentTransaction); err != nil {
		return nil, err
	}

	// Create coin transaction record
	coinTransactionID := uuid.New().String()
	description := req.Description
	if description == "" {
		description = "Coin addition"
	}

	coinTransaction := &model.CoinTransaction{
		TransactionID: coinTransactionID,
		UserID:        userID,
		SiteID:        siteID,
		Amount:        req.CoinAmount,
		BeforeBalance: beforeBalance,
		Balance:       userCoin.Balance,
		Source:        model.CoinSourceAdminAdjustment,
		RelatedID:     paymentTransaction.TransactionID,
		RelatedType:   1, // CoinRelatedTypePayment (写死为1)
		Description:   description,
		Snapshot:      make(model.JSONMap),
	}

	if err := s.coinTransactionRepo.Create(ctx, coinTransaction); err != nil {
		return nil, err
	}

	return &api.InternalAddCoinsResponse{
		Success:       true,
		TransactionID: coinTransactionID,
		Balance:       userCoin.Balance,
		Message:       "Coins added successfully",
	}, nil
}

// InternalDeductCoins deducts coins from user account
func (s *coinsService) InternalDeductCoins(ctx *gin.Context, userID, siteID string, req api.InternalDeductCoinsRequest) (*api.InternalDeductCoinsResponse, error) {
	// Check if user has enough coins first
	userCoin, err := s.userCoinsRepo.GetByUserAndSite(ctx, userID, siteID)
	if err != nil {
		return &api.InternalDeductCoinsResponse{
			Success: false,
			Message: "Failed to get user coin account",
		}, err
	}

	if userCoin == nil {
		return &api.InternalDeductCoinsResponse{
			Success: false,
			Message: "User coin account not found",
		}, common.ErrInsufficientCoins
	}

	// Check total coins (present + balance)
	totalCoins := userCoin.Present + userCoin.Balance
	if totalCoins < req.CoinAmount {
		return &api.InternalDeductCoinsResponse{
			Success: false,
			Balance: totalCoins,
			Message: fmt.Sprintf("Insufficient coins. Current balance: %d, required: %d", totalCoins, req.CoinAmount),
		}, common.ErrInsufficientCoins
	}

	// Record balance before transaction
	beforeBalance := totalCoins

	// Deduct coins (prioritizing present first)
	userCoin, err = s.userCoinsRepo.DeductCoins(ctx, userID, siteID, req.CoinAmount)
	if err != nil {
		return nil, err
	}

	// Create coin transaction record
	coinTransactionID := uuid.New().String()

	source := req.Source
	if source == "" {
		source = model.CoinSourceUnlock
	}

	description := req.Description
	if description == "" {
		description = "Coin deduction"
	}

	coinTransaction := &model.CoinTransaction{
		TransactionID: coinTransactionID,
		UserID:        userID,
		SiteID:        siteID,
		Amount:        -req.CoinAmount, // Negative for deduction
		BeforeBalance: beforeBalance,
		Balance:       userCoin.Present + userCoin.Balance,
		Source:        source,
		RelatedID:     req.RelatedID,
		RelatedType:   5, // CoinRelatedTypePlugin (写死为5)
		Description:   description,
		Snapshot:      make(model.JSONMap),
	}

	if err := s.coinTransactionRepo.Create(ctx, coinTransaction); err != nil {
		return nil, err
	}

	return &api.InternalDeductCoinsResponse{
		Success:       true,
		TransactionID: coinTransactionID,
		Balance:       userCoin.Present + userCoin.Balance,
		Message:       "Coins deducted successfully",
	}, nil
}

// ClaimTaskReward allows users to claim rewards for completing tasks
// 每个任务奖励100金币，每个任务只能完成一次
func (s *coinsService) ClaimTaskReward(ctx *gin.Context, userID, siteID, taskName string) (*api.ClaimTaskRewardResponse, error) {
	// 获取用户金币账户
	userCoins, err := s.userCoinsRepo.GetByUserAndSite(ctx, userID, siteID)
	if err != nil {
		return &api.ClaimTaskRewardResponse{
			Success: false,
			Balance: 0,
			Message: "Failed to get user coin account",
		}, err
	}

	// 如果账户不存在，创建新账户
	if userCoins == nil {
		userCoins = &model.UserCoins{
			UserID:         userID,
			SiteID:         siteID,
			Balance:        0,
			TotalEarned:    0,
			TotalSpent:     0,
			CompletedTasks: "",
		}
		if err := s.userCoinsRepo.Create(ctx, userCoins); err != nil {
			return &api.ClaimTaskRewardResponse{
				Success: false,
				Balance: 0,
				Message: "Failed to create user coin account",
			}, err
		}
	}

	// 检查任务是否已完成
	if userCoins.CompletedTasks != "" {
		// 检查任务名是否在已完成任务列表中
		if containsTask(userCoins.CompletedTasks, taskName) {
			// 任务已完成，返回 false
			return &api.ClaimTaskRewardResponse{
				Success: false,
				Balance: userCoins.Present + userCoins.Balance,
				Message: "Task already completed",
			}, nil
		}
	}

	// 任务未完成，奖励100金币
	const taskReward = 30

	// 更新金币余额和已完成任务列表
	beforeBalance := userCoins.Present + userCoins.Balance

	// 更新 completed_tasks 字段
	if userCoins.CompletedTasks == "" {
		userCoins.CompletedTasks = taskName
	} else {
		userCoins.CompletedTasks = userCoins.CompletedTasks + "," + taskName
	}

	completedTasks := userCoins.CompletedTasks
	// 更新金币余额
	userCoins, err = s.userCoinsRepo.UpdateBalance(ctx, userID, siteID, taskReward, 0)
	if err != nil {
		return &api.ClaimTaskRewardResponse{
			Success: false,
			Balance: beforeBalance,
			Message: "Failed to update coin balance",
		}, err
	}

	// 需要手动更新 CompletedTasks 字段（因为 UpdateBalance 可能不包含这个字段）
	userCoins.CompletedTasks = completedTasks
	if err := s.userCoinsRepo.Update(ctx, userCoins); err != nil {
		return &api.ClaimTaskRewardResponse{
			Success: false,
			Balance: userCoins.Present + userCoins.Balance,
			Message: "Failed to update completed tasks",
		}, err
	}

	// // 创建金币交易记录
	// coinTransactionID := uuid.New().String()
	// coinTransaction := &model.CoinTransaction{
	// 	TransactionID: coinTransactionID,
	// 	UserID:        userID,
	// 	SiteID:        siteID,
	// 	Amount:        taskReward,
	// 	BeforeBalance: beforeBalance,
	// 	Balance:       userCoins.Present + userCoins.Balance,
	// 	Source:        "task_reward",
	// 	RelatedID:     taskName,
	// 	RelatedType:   6, // CoinRelatedTypeTaskReward (新增任务奖励类型)
	// 	Description:   fmt.Sprintf("Task reward: %s", taskName),
	// 	Snapshot:      make(model.JSONMap),
	// }

	// if err := s.coinTransactionRepo.Create(ctx, coinTransaction); err != nil {
	// 	// 交易记录创建失败不影响奖励发放
	// 	// 但记录错误日志
	// 	// s.logger.Error("Failed to create coin transaction", zap.Error(err))
	// }

	return &api.ClaimTaskRewardResponse{
		Success: true,
		Balance: userCoins.Present + userCoins.Balance,
		Message: "Task reward claimed successfully",
	}, nil
}

// containsTask 检查任务名是否在已完成任务列表中
func containsTask(completedTasks, taskName string) bool {
	if completedTasks == "" {
		return false
	}
	tasks := splitTasks(completedTasks)
	for _, t := range tasks {
		if t == taskName {
			return true
		}
	}
	return false
}

// splitTasks 分割任务列表
func splitTasks(completedTasks string) []string {
	tasks := make([]string, 0)
	start := 0
	for i, c := range completedTasks {
		if c == ',' {
			if start < i {
				tasks = append(tasks, completedTasks[start:i])
			}
			start = i + 1
		}
	}
	if start < len(completedTasks) {
		tasks = append(tasks, completedTasks[start:])
	}
	return tasks
}
