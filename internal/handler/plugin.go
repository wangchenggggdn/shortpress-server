package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"shortpress-server/internal/plugins"
	"shortpress-server/pkg/crypto"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"shortpress-server/internal/api"
	"shortpress-server/internal/service"
)

// PluginHandler 插件处理器
type PluginHandler struct {
	*Handler
	PluginService service.PluginService
	plugin        *plugins.Plugins
	conf          *viper.Viper
}

// NewPluginHandler 创建插件处理器
func NewPluginHandler(
	handler *Handler,
	pluginService service.PluginService,
	plugin *plugins.Plugins,
	conf *viper.Viper,
) *PluginHandler {
	return &PluginHandler{
		Handler:       handler,
		PluginService: pluginService,
		plugin:        plugin,
		conf:          conf,
	}
}

// RegisterPluginRequest 注册插件请求
type RegisterPluginRequest struct {
	Name        string `json:"name" binding:"required"` // 插件名称
	Description string `json:"description"`             // 插件描述
	Type        string `json:"type" binding:"required"` // 插件类型：payment, content, analytics 等
	Version     string `json:"version"`                 // 插件版本
	Author      string `json:"author"`                  // 插件作者
}

// InstallPluginRequest 安装插件请求
type InstallPluginRequest struct {
	PluginID string `json:"plugin_id" binding:"required"` // 插件ID
	Config   string `json:"config"`                       // 插件配置（可选，JSON格式）
}

// UninstallPluginRequest 卸载插件请求
type UninstallPluginRequest struct {
	PluginID string `json:"plugin_id" binding:"required"` // 插件ID
}

// RegisterPlugin godoc
// @Summary 注册插件
// @Description 在系统中注册新插件
// @Tags plugin-management
// @Accept json
// @Produce json
// @Param request body RegisterPluginRequest true "注册插件请求"
// @Success 200 {object} api.Response "注册成功"
// @Failure 400 {object} api.Response "参数错误"
// @Failure 500 {object} api.Response "服务器错误"
// @Router /api/plugins/register [post]
func (h *PluginHandler) RegisterPlugin(ctx *gin.Context) {
	var req RegisterPluginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// 调用 service 层注册插件（plugin_id 由系统自动生成）
	pluginID, err := h.PluginService.RegisterPlugin(ctx, req.Name, req.Description, req.Type, req.Version, req.Author)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, map[string]interface{}{
		"plugin_id": pluginID,
	})
}

// InstallPlugin godoc
// @Summary 安装插件到站点
// @Description 为当前站点安装指定插件
// @Tags plugin-management
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param request body InstallPluginRequest true "安装插件请求"
// @Success 200 {object} api.Response "安装成功，返回密钥"
// @Failure 400 {object} api.Response "参数错误"
// @Failure 401 {object} api.Response "未授权"
// @Failure 404 {object} api.Response "插件不存在"
// @Failure 500 {object} api.Response "服务器错误"
// @Router /api/plugins/install [post]
func (h *PluginHandler) InstallPlugin(ctx *gin.Context) {
	siteID := ctx.GetString("site_id")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("missing site_id in context"), nil)
		return
	}

	var req InstallPluginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// 在 Handler 层生成插件密钥
	secretCrypto := crypto.NewPluginSecretCrypto(h.conf)
	secret := secretCrypto.GenerateSecret(req.PluginID, siteID)

	// 调用 service 层安装插件
	sitePlugin, err := h.PluginService.InstallPlugin(ctx, siteID, req.PluginID, secret, req.Config)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, map[string]interface{}{
		"plugin_id": req.PluginID,
		"secret":    sitePlugin.Secret,
	})
}

// UninstallPlugin godoc
// @Summary 卸载站点插件
// @Description 从当前站点卸载指定插件
// @Tags plugin-management
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param request body UninstallPluginRequest true "卸载插件请求"
// @Success 200 {object} api.Response "卸载成功"
// @Failure 400 {object} api.Response "参数错误"
// @Failure 401 {object} api.Response "未授权"
// @Failure 404 {object} api.Response "插件未安装"
// @Failure 500 {object} api.Response "服务器错误"
// @Router /api/plugins/uninstall [post]
func (h *PluginHandler) UninstallPlugin(ctx *gin.Context) {
	siteID := ctx.GetString("site_id")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("missing site_id in context"), nil)
		return
	}

	var req UninstallPluginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// 调用 service 层卸载插件
	err := h.PluginService.UninstallPlugin(ctx, siteID, req.PluginID)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, map[string]interface{}{
		"plugin_id": req.PluginID,
	})
}

// ListPlugins godoc
// @Summary 获取可用插件列表
// @Description 获取所有已注册的插件列表
// @Tags plugin-management
// @Accept json
// @Produce json
// @Success 200 {object} api.Response "获取成功"
// @Failure 500 {object} api.Response "服务器错误"
// @Router /api/plugins/list [get]
func (h *PluginHandler) ListPlugins(ctx *gin.Context) {
	plugins, err := h.PluginService.ListPlugins(ctx)
	if err != nil {
		api.HandleError(ctx, err, "Failed to list plugins")
		return
	}

	api.HandleSuccess(ctx, plugins)
}

// ListInstalledPlugins godoc
// @Summary 获取站点已安装插件列表
// @Description 获取当前站点已安装的插件列表
// @Tags plugin-management
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Success 200 {object} api.Response "获取成功"
// @Failure 400 {object} api.Response "参数错误"
// @Failure 401 {object} api.Response "未授权"
// @Failure 500 {object} api.Response "服务器错误"
// @Router /api/plugins/installed/list [get]
func (h *PluginHandler) ListInstalledPlugins(ctx *gin.Context) {
	siteID := ctx.GetString("site_id")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("missing site_id in context"), nil)
		return
	}

	// 调用 service 层获取已安装插件
	result, err := h.PluginService.ListInstalledPlugins(ctx, siteID)
	if err != nil {
		api.HandleError(ctx, err, "Failed to list installed plugins")
		return
	}

	// 格式化返回数据
	response := make([]map[string]interface{}, 0, len(result))
	for _, item := range result {
		response = append(response, map[string]interface{}{
			"plugin_id":     item.Plugin.PluginID,
			"name":          item.Plugin.Name,
			"description":   item.Plugin.Description,
			"type":          item.Plugin.Type,
			"version":       item.Plugin.Version,
			"author":        item.Plugin.Author,
			"plugin_status": item.Plugin.Status,
			"status":        item.SitePlugin.Status,
			"config":        item.SitePlugin.Config,
			"secret":        item.SitePlugin.Secret,
			"installed_at":  item.SitePlugin.InstalledAt,
			"last_used_at":  item.SitePlugin.LastUsedAt,
		})
	}

	api.HandleSuccess(ctx, response)
}

// ActiveCallRequest 主动调用请求
type ActiveCallRequest struct {
	Hook    string                 `json:"hook" binding:"required"`    // hook 类型
	Actions []plugins.ActionParams `json:"actions" binding:"required"` // action 列表
}

// Call godoc
// @Summary 插件主动调用接口
// @Description 插件通过此接口主动调用指定的 action
// @Tags plugins
// @Accept json
// @Produce json
// @Param X-User-ID header string true "User ID"
// @Param X-Site-Id header string true "Site ID"
// @Param X-Plugins-Id header string true "Plugins ID"
// @Param request body ActiveCallRequest true "请求参数"
// @Success 200 {object} api.Response "成功返回 action 执行结果"
// @Failure 400 {object} api.Response "参数错误"
// @Failure 401 {object} api.Response "未授权"
// @Router /api/plugins/hook/active_call [post]
func (h *PluginHandler) Call(ctx *gin.Context) {

	var req struct {
		Action string          `json:"action"`
		Params json.RawMessage `json:"params"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	result, err := h.plugin.Hook.Report(ctx, plugins.HookActiveCall, plugins.ActionParams{Action: req.Action, Params: req.Params})
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, result)
}
