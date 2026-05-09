package service

import (
	"context"
	"errors"
	"fmt"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/plugins"

	"github.com/google/uuid"
)

// PluginService 插件服务
type PluginService interface {
	// RegisterPlugin 注册插件（自动生成 plugin_id）
	RegisterPlugin(ctx context.Context, name, description, pluginType, version, author string) (string, error)
	// ListPlugins 获取所有插件列表
	ListPlugins(ctx context.Context) ([]*model.Plugin, error)
	// GetPluginByID 根据ID获取插件
	GetPluginByID(ctx context.Context, pluginID string) (*model.Plugin, error)
	// InstallPlugin 安装插件到站点
	InstallPlugin(ctx context.Context, siteID, pluginID, secret, config string) (*model.SitePlugin, error)
	// UninstallPlugin 卸载站点插件
	UninstallPlugin(ctx context.Context, siteID, pluginID string) error
	// ListInstalledPlugins 获取站点已安装插件列表
	ListInstalledPlugins(ctx context.Context, siteID string) ([]*model.PluginWithInfo, error)
	// GetSitePlugin 获取站点插件信息
	GetSitePlugin(ctx context.Context, siteID, pluginID string) (*model.SitePlugin, error)
	// GeneratePluginSecret 生成插件密钥
	GeneratePluginSecret(siteID, pluginID string) string
}

type PluginWithInfo struct {
	Plugin     *model.Plugin
	SitePlugin *model.SitePlugin
}

type pluginService struct {
	pluginRepo     plugins.PluginRepository
	sitePluginRepo plugins.SitePluginRepository
}

// NewPluginService 创建插件服务
func NewPluginService(
	pluginRepo plugins.PluginRepository,
	sitePluginRepo plugins.SitePluginRepository,
) PluginService {
	return &pluginService{
		pluginRepo:     pluginRepo,
		sitePluginRepo: sitePluginRepo,
	}
}

// RegisterPlugin 注册插件（自动生成 plugin_id）
func (s *pluginService) RegisterPlugin(ctx context.Context, name, description, pluginType, version, author string) (string, error) {
	// 1. 验证必填字段
	if name == "" {
		return "", errors.New("name is required")
	}
	if pluginType == "" {
		return "", errors.New("type is required")
	}

	// 2. 自动生成唯一的 plugin_id（使用 UUID，与其他 ID 保持一致）
	pluginID := uuid.New().String()

	// 3. 创建插件
	plugin := &model.Plugin{
		PluginID:    pluginID,
		Name:        name,
		Description: description,
		Type:        pluginType,
		Version:     version,
		Author:      author,
		Status:      model.PluginStatusActive,
		PageURL:     "",
	}

	err := s.pluginRepo.Create(ctx, plugin)
	if err != nil {
		return "", fmt.Errorf("failed to create plugin: %w", err)
	}

	return pluginID, nil
}

// ListPlugins 获取所有插件列表
func (s *pluginService) ListPlugins(ctx context.Context) ([]*model.Plugin, error) {
	plugins, err := s.pluginRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}
	return plugins, nil
}

// GetPluginByID 根据ID获取插件
func (s *pluginService) GetPluginByID(ctx context.Context, pluginID string) (*model.Plugin, error) {
	if pluginID == "" {
		return nil, errors.New("plugin_id is required")
	}

	plugin, err := s.pluginRepo.GetByPluginID(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}
	if plugin == nil {
		return nil, errors.New("plugin not found")
	}

	return plugin, nil
}

// InstallPlugin 安装插件到站点
func (s *pluginService) InstallPlugin(ctx context.Context, siteID, pluginID, secret, config string) (*model.SitePlugin, error) {
	// 1. 验证必填字段
	if siteID == "" {
		return nil, errors.New("site_id is required")
	}
	if pluginID == "" {
		return nil, errors.New("plugin_id is required")
	}
	if secret == "" {
		return nil, errors.New("secret is required")
	}

	// 2. 检查插件是否存在
	plugin, err := s.pluginRepo.GetByPluginID(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}
	if plugin == nil {
		return nil, errors.New("plugin not found")
	}

	// 3. 检查插件状态
	if plugin.Status != model.PluginStatusActive {
		return nil, fmt.Errorf("plugin is not active, current status: %s", plugin.Status)
	}

	// 4. 检查是否已安装
	existing, err := s.sitePluginRepo.GetBySiteAndPluginID(ctx, siteID, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to check installation status: %w", err)
	}
	if existing != nil {
		return nil, errors.New("plugin already installed")
	}

	// 5. 安装插件（使用传入的 secret）
	sitePlugin, err := s.sitePluginRepo.InstallWithSecret(ctx, siteID, pluginID, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to install plugin: %w", err)
	}

	// 6. 如果有配置，更新配置
	if config != "" {
		err = s.sitePluginRepo.UpdateConfig(ctx, siteID, pluginID, config)
		if err != nil {
			return nil, fmt.Errorf("plugin installed but failed to update config: %w", err)
		}
	}

	return sitePlugin, nil
}

// UninstallPlugin 卸载站点插件
func (s *pluginService) UninstallPlugin(ctx context.Context, siteID, pluginID string) error {
	// 1. 验证必填字段
	if siteID == "" {
		return errors.New("site_id is required")
	}
	if pluginID == "" {
		return errors.New("plugin_id is required")
	}

	// 2. 检查插件是否已安装
	existing, err := s.sitePluginRepo.GetBySiteAndPluginID(ctx, siteID, pluginID)
	if err != nil {
		return fmt.Errorf("failed to check installation status: %w", err)
	}
	if existing == nil {
		return errors.New("plugin not installed")
	}

	// 3. 卸载插件
	err = s.sitePluginRepo.Uninstall(ctx, siteID, pluginID)
	if err != nil {
		return fmt.Errorf("failed to uninstall plugin: %w", err)
	}

	return nil
}

// ListInstalledPlugins 获取站点已安装插件列表
func (s *pluginService) ListInstalledPlugins(ctx context.Context, siteID string) ([]*model.PluginWithInfo, error) {
	// 1. 验证站点ID
	if siteID == "" {
		return nil, errors.New("site_id is required")
	}

	// 2. 获取站点已安装的插件
	sitePlugins, err := s.sitePluginRepo.GetBySiteID(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get installed plugins: %w", err)
	}

	// 3. 获取每个插件的详细信息
	result := make([]*model.PluginWithInfo, 0, len(sitePlugins))
	for _, sp := range sitePlugins {
		plugin, err := s.pluginRepo.GetByPluginID(ctx, sp.PluginID)
		if err != nil {
			// 跳过获取失败的插件
			continue
		}
		if plugin == nil {
			continue
		}

		result = append(result, &model.PluginWithInfo{
			Plugin:     plugin,
			SitePlugin: sp,
		})
	}

	return result, nil
}

// GetSitePlugin 获取站点插件信息
func (s *pluginService) GetSitePlugin(ctx context.Context, siteID, pluginID string) (*model.SitePlugin, error) {
	// 1. 验证必填字段
	if siteID == "" {
		return nil, errors.New("site_id is required")
	}
	if pluginID == "" {
		return nil, errors.New("plugin_id is required")
	}

	// 2. 获取站点插件
	sitePlugin, err := s.sitePluginRepo.GetBySiteAndPluginID(ctx, siteID, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site plugin: %w", err)
	}
	if sitePlugin == nil {
		return nil, errors.New("plugin not installed")
	}

	return sitePlugin, nil
}

// GeneratePluginSecret 生成插件密钥
// 注意：此方法需要 viper.Viper 参数，建议在 Handler 层调用 crypto.GenerateSecret
func (s *pluginService) GeneratePluginSecret(siteID, pluginID string) string {
	// 由于不再注入 crypto，这里返回空字符串
	// 实际的 secret 生成应该在 Handler 层完成
	return ""
}
