package plugins

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SitePluginRepository 网站插件仓库接口
type SitePluginRepository interface {
	db.BaseOperation
	// GetBySiteAndPluginID 根据站点ID和插件ID获取安装记录
	GetBySiteAndPluginID(ctx context.Context, siteID, pluginID string) (*model.SitePlugin, error)
	// GetBySiteID 获取站点的所有已安装插件
	GetBySiteID(ctx context.Context, siteID string) ([]*model.SitePlugin, error)
	// GetBySiteIDAndStatus 根据站点ID和状态获取插件
	GetBySiteIDAndStatus(ctx context.Context, siteID, status string) ([]*model.SitePlugin, error)
	// Install 安装插件到站点（生成 secret）
	Install(ctx context.Context, siteID, pluginID string) (*model.SitePlugin, error)
	// InstallWithSecret 安装插件到站点（使用指定的 secret）
	InstallWithSecret(ctx context.Context, siteID, pluginID, secret string) (*model.SitePlugin, error)
	// Uninstall 卸载站点插件
	Uninstall(ctx context.Context, siteID, pluginID string) error
	// UpdateConfig 更新插件配置
	UpdateConfig(ctx context.Context, siteID, pluginID, config string) error
	// UpdateStatus 更新插件状态
	UpdateStatus(ctx context.Context, siteID, pluginID, status string) error
	// UpdateLastUsedAt 更新最后使用时间
	UpdateLastUsedAt(ctx context.Context, siteID, pluginID string) error
	// GetSecret 获取插件密钥
	GetSecret(ctx context.Context, siteID, pluginID string) (string, error)
}

type sitePluginRepository struct {
	*db.Repository
}

// NewSitePluginRepository 创建网站插件仓库
func NewSitePluginRepository(r *db.Repository) SitePluginRepository {
	return &sitePluginRepository{
		Repository: r,
	}
}

func (r *sitePluginRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *sitePluginRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

func (r *sitePluginRepository) GetBySiteAndPluginID(ctx context.Context, siteID, pluginID string) (*model.SitePlugin, error) {
	var sitePlugin model.SitePlugin
	err := r.DB(ctx).Where("site_id = ? AND plugin_id = ?", siteID, pluginID).First(&sitePlugin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &sitePlugin, nil
}

func (r *sitePluginRepository) GetBySiteID(ctx context.Context, siteID string) ([]*model.SitePlugin, error) {
	var sitePlugins []*model.SitePlugin
	err := r.DB(ctx).Where("site_id = ?", siteID).Find(&sitePlugins).Error
	if err != nil {
		return nil, err
	}
	return sitePlugins, nil
}

func (r *sitePluginRepository) GetBySiteIDAndStatus(ctx context.Context, siteID, status string) ([]*model.SitePlugin, error) {
	var sitePlugins []*model.SitePlugin
	err := r.DB(ctx).Where("site_id = ? AND status = ?", siteID, status).Find(&sitePlugins).Error
	if err != nil {
		return nil, err
	}
	return sitePlugins, nil
}

func (r *sitePluginRepository) Install(ctx context.Context, siteID, pluginID string) (*model.SitePlugin, error) {
	// 1. 检查是否已安装
	existing, err := r.GetBySiteAndPluginID(ctx, siteID, pluginID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, gorm.ErrDuplicatedKey
	}

	// 2. 生成密钥
	secret := r.generateSecret()

	// 3. 创建安装记录
	sitePlugin := &model.SitePlugin{
		SiteID:     siteID,
		PluginID:   pluginID,
		Secret:     secret,
		Status:     model.PluginStatusActive,
		Config:     "{}",
		LastUsedAt: time.Now(),
	}

	err = r.DB(ctx).Create(sitePlugin).Error
	if err != nil {
		return nil, err
	}

	return sitePlugin, nil
}

func (r *sitePluginRepository) InstallWithSecret(ctx context.Context, siteID, pluginID, secret string) (*model.SitePlugin, error) {
	// 1. 检查是否已安装
	existing, err := r.GetBySiteAndPluginID(ctx, siteID, pluginID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, gorm.ErrDuplicatedKey
	}

	// 2. 创建安装记录（使用指定的 secret）
	sitePlugin := &model.SitePlugin{
		SiteID:     siteID,
		PluginID:   pluginID,
		Secret:     secret,
		Status:     model.PluginStatusActive,
		Config:     "{}",
		LastUsedAt: time.Now(),
	}

	err = r.DB(ctx).Create(sitePlugin).Error
	if err != nil {
		return nil, err
	}

	return sitePlugin, nil
}

func (r *sitePluginRepository) Uninstall(ctx context.Context, siteID, pluginID string) error {
	return r.DB(ctx).Where("site_id = ? AND plugin_id = ?", siteID, pluginID).Delete(&model.SitePlugin{}).Error
}

func (r *sitePluginRepository) UpdateConfig(ctx context.Context, siteID, pluginID, config string) error {
	return r.DB(ctx).Model(&model.SitePlugin{}).
		Where("site_id = ? AND plugin_id = ?", siteID, pluginID).
		Update("config", config).Error
}

func (r *sitePluginRepository) UpdateStatus(ctx context.Context, siteID, pluginID, status string) error {
	return r.DB(ctx).Model(&model.SitePlugin{}).
		Where("site_id = ? AND plugin_id = ?", siteID, pluginID).
		Update("status", status).Error
}

func (r *sitePluginRepository) UpdateLastUsedAt(ctx context.Context, siteID, pluginID string) error {
	return r.DB(ctx).Model(&model.SitePlugin{}).
		Where("site_id = ? AND plugin_id = ?", siteID, pluginID).
		Update("last_used_at", time.Now()).Error
}

func (r *sitePluginRepository) GetSecret(ctx context.Context, siteID, pluginID string) (string, error) {
	sitePlugin, err := r.GetBySiteAndPluginID(ctx, siteID, pluginID)
	if err != nil {
		return "", err
	}
	if sitePlugin == nil {
		return "", gorm.ErrRecordNotFound
	}
	return sitePlugin.Secret, nil
}

// generateSecret 生成插件密钥
func (r *sitePluginRepository) generateSecret() string {
	return "sp_" + uuid.New().String()
}
