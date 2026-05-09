package action

import (
	"encoding/json"
	"shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/plugins"
	"shortpress-server/internal/repository/db/user"

	"github.com/gin-gonic/gin"
)

type Action struct {
	actionProcessors    map[string]actionFunc
	userCoinsRepo       payment.UserCoinsRepository
	coinTransactionRepo payment.CoinTransactionRepository
	userRepo            user.UserRepository
	pluginOrderRepo     plugins.PluginOrderRepository
	coinPackageRepo     payment.CoinPackageRepository
}

type actionFunc func(ctx *gin.Context, args json.RawMessage) (json.RawMessage, error)

func NewAction(
	userCoinsRepository payment.UserCoinsRepository,
	coinTransactionRepository payment.CoinTransactionRepository,
	userRepository user.UserRepository,
	pluginOrderRepository plugins.PluginOrderRepository,
	coinPackageRepository payment.CoinPackageRepository,
) *Action {
	var action = &Action{
		userCoinsRepo:       userCoinsRepository,
		coinTransactionRepo: coinTransactionRepository,
		userRepo:            userRepository,
		pluginOrderRepo:     pluginOrderRepository,
		coinPackageRepo:     coinPackageRepository,
	}
	action.register()
	return action
}

func (a *Action) Process(ctx *gin.Context, action string, args json.RawMessage) (json.RawMessage, error) {
	if processor, ok := a.actionProcessors[action]; ok {
		return processor(ctx, args)
	}
	return nil, nil
}

func (a *Action) register() {
	a.actionProcessors = map[string]actionFunc{
		"create_order": a.CreateOrder,
		"add_coins":    a.AddCoins,
		"deduct_coins": a.DeductCoins,
	}
}
