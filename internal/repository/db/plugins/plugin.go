package plugins

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

// PluginRepository 插件注册仓库接口
type PluginRepository interface {
	db.BaseOperation
	// GetByPluginID 根据 plugin_id 获取插件
	GetByPluginID(ctx context.Context, pluginID string) (*model.Plugin, error)
	// List 获取所有插件列表
	List(ctx context.Context) ([]*model.Plugin, error)
	// ListByStatus 根据状态获取插件列表
	ListByStatus(ctx context.Context, status string) ([]*model.Plugin, error)
	// UpdateStatus 更新插件状态
	UpdateStatus(ctx context.Context, pluginID string, status string) error
}

type pluginRepository struct {
	*db.Repository
}

// NewPluginRepository 创建插件仓库
func NewPluginRepository(r *db.Repository) PluginRepository {
	return &pluginRepository{
		Repository: r,
	}
}

func (r *pluginRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *pluginRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

func (r *pluginRepository) GetByPluginID(ctx context.Context, pluginID string) (*model.Plugin, error) {
	var plugin model.Plugin
	err := r.DB(ctx).Where("plugin_id = ?", pluginID).First(&plugin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &plugin, nil
}

func (r *pluginRepository) List(ctx context.Context) ([]*model.Plugin, error) {
	var plugins []*model.Plugin
	err := r.DB(ctx).Find(&plugins).Error
	if err != nil {
		return nil, err
	}
	return plugins, nil
}

func (r *pluginRepository) ListByStatus(ctx context.Context, status string) ([]*model.Plugin, error) {
	var plugins []*model.Plugin
	err := r.DB(ctx).Where("status = ?", status).Find(&plugins).Error
	if err != nil {
		return nil, err
	}
	return plugins, nil
}

func (r *pluginRepository) UpdateStatus(ctx context.Context, pluginID string, status string) error {
	return r.DB(ctx).Model(&model.Plugin{}).
		Where("plugin_id = ?", pluginID).
		Update("status", status).Error
}
