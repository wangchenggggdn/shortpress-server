package coins

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	wheelPaidCostDefault  = 30
	wheelPaidCostVIP      = 25
	wheelPaidDailyCap     = 50
	wheelFirstSpinMinFree = 15
	wheelFirstSpinTaskKey = "wheel_first_spin_done"
)

type wheelPrize struct {
	ID     string
	Coins  int
	Weight int
}

var wheelFreePrizes = []wheelPrize{
	{ID: "thanks", Coins: 0, Weight: 250},
	{ID: "coins_5", Coins: 5, Weight: 300},
	{ID: "coins_10", Coins: 10, Weight: 250},
	{ID: "coins_15", Coins: 15, Weight: 150},
	{ID: "coins_20", Coins: 20, Weight: 40},
	{ID: "coins_30", Coins: 30, Weight: 9},
	{ID: "coins_50", Coins: 50, Weight: 1},
}

var wheelPaidPrizes = []wheelPrize{
	{ID: "thanks", Coins: 0, Weight: 280},
	{ID: "break_even", Coins: 10, Weight: 220},
	{ID: "coins_25", Coins: 25, Weight: 200},
	{ID: "coins_50", Coins: 50, Weight: 140},
	{ID: "coins_80", Coins: 80, Weight: 80},
	{ID: "coins_120", Coins: 120, Weight: 50},
	{ID: "coins_200", Coins: 200, Weight: 25},
	{ID: "coins_300", Coins: 300, Weight: 5},
}

var wheelCST = time.FixedZone("CST", 8*3600)

func wheelDateKey() string {
	return time.Now().In(wheelCST).Format("2006-01-02")
}

func nextWheelResetAt() int64 {
	now := time.Now().In(wheelCST)
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, wheelCST)
	return tomorrow.Unix()
}

func wheelFreeTaskKey(dateKey string) string {
	return fmt.Sprintf("wheel_free_%s", dateKey)
}

func wheelPaidTaskKey(dateKey string, index int) string {
	return fmt.Sprintf("wheel_paid_%s_%d", dateKey, index)
}

func countPaidSpinsToday(completedTasks, dateKey string) int {
	prefix := fmt.Sprintf("wheel_paid_%s_", dateKey)
	count := 0
	for _, task := range splitTasks(completedTasks) {
		if len(task) > len(prefix) && task[:len(prefix)] == prefix {
			count++
		}
	}
	return count
}

func toWheelPrizeItems(prizes []wheelPrize) []api.WheelPrizeItem {
	items := make([]api.WheelPrizeItem, 0, len(prizes))
	for _, prize := range prizes {
		items = append(items, api.WheelPrizeItem{
			ID:    prize.ID,
			Coins: prize.Coins,
		})
	}
	return items
}

func paidSpinCost(isVIP bool) int {
	if isVIP {
		return wheelPaidCostVIP
	}
	return wheelPaidCostDefault
}

func (s *coinsService) isVIPUser(ctx *gin.Context, userID string) bool {
	user, err := s.userRepository.GetByUserID(ctx, userID)
	if err != nil || user == nil {
		return false
	}
	return user.PremiumType >= 1
}

func secureRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	var buf [8]byte
	for {
		if _, err := rand.Read(buf[:]); err != nil {
			return 0
		}
		n := int(binary.BigEndian.Uint64(buf[:]) % uint64(max))
		return n
	}
}

func pickWheelPrize(prizes []wheelPrize, minCoins int) (wheelPrize, int) {
	total := 0
	for _, prize := range prizes {
		total += prize.Weight
	}
	if total <= 0 {
		return prizes[0], 0
	}

	n := secureRandInt(total)
	cumulative := 0
	picked := prizes[0]
	pickedIndex := 0
	for i, prize := range prizes {
		cumulative += prize.Weight
		if n < cumulative {
			picked = prize
			pickedIndex = i
			break
		}
	}

	if minCoins > 0 && picked.Coins < minCoins {
		bestIndex := pickedIndex
		best := picked
		for i, prize := range prizes {
			if prize.Coins >= minCoins && (best.Coins < minCoins || prize.Coins < best.Coins) {
				best = prize
				bestIndex = i
			}
		}
		if best.Coins >= minCoins {
			picked = best
			pickedIndex = bestIndex
		}
	}

	return picked, pickedIndex
}

func appendCompletedTask(completedTasks, taskName string) string {
	if completedTasks == "" {
		return taskName
	}
	return completedTasks + "," + taskName
}

func (s *coinsService) ensureUserCoins(ctx *gin.Context, userID, siteID string) (*model.UserCoins, error) {
	userCoins, err := s.userCoinsRepo.GetByUserAndSite(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}
	if userCoins != nil {
		return userCoins, nil
	}

	userCoins = &model.UserCoins{
		UserID:         userID,
		SiteID:         siteID,
		Balance:        0,
		TotalEarned:    0,
		TotalSpent:     0,
		CompletedTasks: "",
	}
	if err := s.userCoinsRepo.Create(ctx, userCoins); err != nil {
		return nil, err
	}
	return userCoins, nil
}

// GetWheelStatus returns wheel availability and prize config.
func (s *coinsService) GetWheelStatus(ctx *gin.Context, userID, siteID string, isVIP bool) (*api.WheelStatusResponse, error) {
	if !isVIP {
		isVIP = s.isVIPUser(ctx, userID)
	}
	userCoins, err := s.ensureUserCoins(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}

	dateKey := wheelDateKey()
	freeTaskKey := wheelFreeTaskKey(dateKey)
	freeAvailable := !containsTask(userCoins.CompletedTasks, freeTaskKey)

	return &api.WheelStatusResponse{
		Balance:           userCoins.Present + userCoins.Balance,
		FreeSpinAvailable: freeAvailable,
		NextFreeSpinAt:    nextWheelResetAt(),
		PaidCostPerSpin:   paidSpinCost(isVIP),
		PaidSpinsToday:    countPaidSpinsToday(userCoins.CompletedTasks, dateKey),
		PaidDailyCap:      wheelPaidDailyCap,
		FreePrizes:        toWheelPrizeItems(wheelFreePrizes),
		PaidPrizes:        toWheelPrizeItems(wheelPaidPrizes),
	}, nil
}

// SpinWheel executes a free or paid wheel spin.
func (s *coinsService) SpinWheel(ctx *gin.Context, userID, siteID string, isVIP bool, mode string) (*api.WheelSpinResponse, error) {
	if !isVIP {
		isVIP = s.isVIPUser(ctx, userID)
	}
	userCoins, err := s.ensureUserCoins(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}

	dateKey := wheelDateKey()
	balance := userCoins.Present + userCoins.Balance

	switch mode {
	case "free":
		freeTaskKey := wheelFreeTaskKey(dateKey)
		if containsTask(userCoins.CompletedTasks, freeTaskKey) {
			return &api.WheelSpinResponse{
				Success: false,
				Mode:    mode,
				Cost:    0,
				Balance: balance,
				Message: "Free spin already used today",
			}, nil
		}

		minCoins := 0
		if !containsTask(userCoins.CompletedTasks, wheelFirstSpinTaskKey) {
			minCoins = wheelFirstSpinMinFree
		}

		prize, index := pickWheelPrize(wheelFreePrizes, minCoins)
		completedTasks := appendCompletedTask(userCoins.CompletedTasks, freeTaskKey)
		completedTasks = appendCompletedTask(completedTasks, wheelFirstSpinTaskKey)

		if prize.Coins > 0 {
			balance, err = s.applyWheelReward(ctx, userID, siteID, prize.Coins, mode, prize.ID, balance)
			if err != nil {
				return nil, err
			}
		}

		userCoins, err = s.userCoinsRepo.GetByUserAndSite(ctx, userID, siteID)
		if err != nil {
			return nil, err
		}
		userCoins.CompletedTasks = completedTasks
		if err := s.userCoinsRepo.Update(ctx, userCoins); err != nil {
			return nil, err
		}

		return &api.WheelSpinResponse{
			Success: true,
			Mode:    mode,
			Cost:    0,
			Results: []api.WheelSpinResultItem{{
				PrizeID: prize.ID,
				Coins:   prize.Coins,
				Index:   index,
			}},
			Balance: balance,
			Message: "Free spin completed",
		}, nil

	case "paid":
		paidCount := countPaidSpinsToday(userCoins.CompletedTasks, dateKey)
		if paidCount >= wheelPaidDailyCap {
			return &api.WheelSpinResponse{
				Success: false,
				Mode:    mode,
				Cost:    0,
				Balance: balance,
				Message: "Daily paid spin limit reached",
			}, nil
		}

		cost := paidSpinCost(isVIP)
		if balance < cost {
			return &api.WheelSpinResponse{
				Success: false,
				Mode:    mode,
				Cost:    cost,
				Balance: balance,
				Message: "Insufficient coins",
			}, common.ErrInsufficientCoins
		}

		prize, index := pickWheelPrize(wheelPaidPrizes, 0)

		balance, err = s.applyWheelCost(ctx, userID, siteID, cost, mode, balance)
		if err != nil {
			return nil, err
		}

		if prize.Coins > 0 {
			balance, err = s.applyWheelReward(ctx, userID, siteID, prize.Coins, mode, prize.ID, balance)
			if err != nil {
				return nil, err
			}
		}

		paidTaskKey := wheelPaidTaskKey(dateKey, paidCount+1)
		userCoins, err = s.userCoinsRepo.GetByUserAndSite(ctx, userID, siteID)
		if err != nil {
			return nil, err
		}
		userCoins.CompletedTasks = appendCompletedTask(userCoins.CompletedTasks, paidTaskKey)
		if err := s.userCoinsRepo.Update(ctx, userCoins); err != nil {
			return nil, err
		}

		return &api.WheelSpinResponse{
			Success: true,
			Mode:    mode,
			Cost:    cost,
			Results: []api.WheelSpinResultItem{{
				PrizeID: prize.ID,
				Coins:   prize.Coins,
				Index:   index,
			}},
			Balance: balance,
			Message: "Paid spin completed",
		}, nil

	default:
		return nil, fmt.Errorf("invalid wheel mode: %s", mode)
	}
}

func (s *coinsService) applyWheelCost(ctx *gin.Context, userID, siteID string, cost int, mode string, beforeBalance int) (int, error) {
	userCoins, err := s.userCoinsRepo.DeductCoins(ctx, userID, siteID, cost)
	if err != nil {
		return beforeBalance, err
	}

	transactionID := uuid.New().String()
	coinTx := &model.CoinTransaction{
		TransactionID: transactionID,
		UserID:        userID,
		SiteID:        siteID,
		Amount:        -cost,
		BeforeBalance: beforeBalance,
		Balance:       userCoins.Present + userCoins.Balance,
		Source:        model.CoinSourceWheelSpin,
		RelatedID:     mode,
		RelatedType:   model.CoinRelatedTypeWheel,
		Description:   fmt.Sprintf("Lucky wheel spin cost (%s)", mode),
		Snapshot:      make(model.JSONMap),
	}
	if err := s.coinTransactionRepo.Create(ctx, coinTx); err != nil {
		return beforeBalance, err
	}

	return userCoins.Present + userCoins.Balance, nil
}

func (s *coinsService) applyWheelReward(ctx *gin.Context, userID, siteID string, amount int, mode, prizeID string, beforeBalance int) (int, error) {
	userCoins, err := s.userCoinsRepo.UpdateBalance(ctx, userID, siteID, amount, 0)
	if err != nil {
		return beforeBalance, err
	}

	transactionID := uuid.New().String()
	coinTx := &model.CoinTransaction{
		TransactionID: transactionID,
		UserID:        userID,
		SiteID:        siteID,
		Amount:        amount,
		BeforeBalance: beforeBalance,
		Balance:       userCoins.Present + userCoins.Balance,
		Source:        model.CoinSourceWheelReward,
		RelatedID:     prizeID,
		RelatedType:   model.CoinRelatedTypeWheel,
		Description:   fmt.Sprintf("Lucky wheel reward (%s)", mode),
		Snapshot:      make(model.JSONMap),
	}
	if err := s.coinTransactionRepo.Create(ctx, coinTx); err != nil {
		return beforeBalance, err
	}

	return userCoins.Present + userCoins.Balance, nil
}
