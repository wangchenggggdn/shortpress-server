package plugins

import (
	"shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/plugins"
	"shortpress-server/internal/repository/db/user"
)

const (
	// HookActiveCall 在handle层提供单独接口供插件直接调用，report后会调用传入的action及参数
	HookActiveCall = "hook_active_call"
	// HookActiveTrigger 在handle层提供单独接口供插件直接触发，report后会自动调用预先设定好的action及参数
	HookActiveTrigger = "hook_active_trigger"
)

type Plugins struct {
	Hook
}

func NewPlugins(
	userCoinsRepo payment.UserCoinsRepository,
	coinTransactionRepo payment.CoinTransactionRepository,
	userRepo user.UserRepository,
	pluginOrderRepo plugins.PluginOrderRepository,
	coinPackageRepo payment.CoinPackageRepository,
) *Plugins {
	return &Plugins{
		Hook: NewHook(userCoinsRepo, coinTransactionRepo, userRepo, pluginOrderRepo, coinPackageRepo),
	}
}
