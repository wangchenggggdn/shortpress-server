package plugins

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"shortpress-server/internal/common"
	"shortpress-server/internal/plugins/action"
	"shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/plugins"
	"shortpress-server/internal/repository/db/user"
	"shortpress-server/pkg/log"
)

type Hook struct {
	action *action.Action
}

func NewHook(
	userCoinsRepository payment.UserCoinsRepository,
	coinTransactionRepository payment.CoinTransactionRepository,
	userRepository user.UserRepository,
	pluginOrderRepository plugins.PluginOrderRepository,
	coinPackageRepo payment.CoinPackageRepository,
) Hook {
	return Hook{
		action: action.NewAction(userCoinsRepository, coinTransactionRepository, userRepository, pluginOrderRepository, coinPackageRepo),
	}
}

type ActionParams struct {
	Action string
	Params json.RawMessage
}

func (h *Hook) getPluginInfo(ctx *gin.Context, hook string) (string, string, error) {
	// 需要在中间件中需要验证plugins的id和对应的secret
	siteId := ctx.GetString("site_id")
	pluginId := ctx.GetString("plugin_id")

	// 在这里查询数据库中查找siteId,pluginId及hook是否对应关系
	// 第一版可以先不查

	return siteId, pluginId, nil

}

func (h *Hook) Report(ctx *gin.Context, hook string, actions ...ActionParams) (json.RawMessage, error) {
	siteId, pluginId, err := h.getPluginInfo(ctx, hook)
	if err != nil {
		return nil, common.ErrHookRunFailed
	}
	log.AddNotice(ctx, "hook", hook)
	log.AddNotice(ctx, "site_id", siteId)
	log.AddNotice(ctx, "plugin_id", pluginId)

	// 如果是主动触发hook，则执行传入的逻辑。
	if hook == HookActiveCall {
		if len(actions) != 1 {
			return nil, common.ErrHookActiveCallFailed
		}
	} else {
		// 在数据库中根据plugin_id,hook 查找后续需要执行的action。(需要实现hook相关表单，第一版不涉及自动触发的插件，所以先不做)
		// 后续需要引入redis,避免频繁查询数据库
		// 再后续建议将插件服务拆成微服务，数据库用mongo，存各种插件配置更加灵活
		// 大概率没后续了，所以先不考虑...

		// 将数据库中的action拼装成ActionParams，这里不查，所以只拼装一个空数组
		actions = []ActionParams{}
	}
	for _, act := range actions {
		log.AddNotice(ctx, "action", act.Action)
		log.AddNotice(ctx, "params", string(act.Params))
		// 执行action
		var result json.RawMessage
		result, err = h.action.Process(ctx, act.Action, act.Params)
		if err != nil {
			return nil, common.ErrHookRunFailed
		}
		// 如果是主动触发的hook，则返回结果,否则打印日志，继续执行下一个 action
		if hook == HookActiveCall {
			return result, nil
		} else {
			// TODO: 完善日志输出
			log.AddNotice(ctx, "result", string(result))
		}

	}
	return nil, nil
}
