package video

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
)

type VideoSourceRepository interface {
	Create(ctx context.Context, entity *model.VideoSource) error
	BatchCreate(ctx context.Context, entities []*model.VideoSource) error
	ListByVID(ctx context.Context, vid string) ([]*model.VideoSource, error)
	GetPrimaryByVID(ctx context.Context, vid string) (*model.VideoSource, error)
	UpdateUploadStatusByVIDProvider(ctx context.Context, vid string, provider string, status int8) error
	UpdateUploadStatusBySourceID(ctx context.Context, sourceID string, status int8) error
	UpdateBySourceID(ctx context.Context, sourceID string, fields map[string]interface{}) error
	DelNotIn(ctx context.Context, vid string, keepIDs []string) error
}

func NewVideoSourceRepository(repository *db.Repository) VideoSourceRepository {
	return &videoSourceRepository{Repository: repository}
}

type videoSourceRepository struct{ *db.Repository }

func (r *videoSourceRepository) Create(ctx context.Context, entity *model.VideoSource) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *videoSourceRepository) BatchCreate(ctx context.Context, entities []*model.VideoSource) error {
	if len(entities) == 0 {
		return nil
	}
	return r.DB(ctx).Create(&entities).Error
}

func (r *videoSourceRepository) ListByVID(ctx context.Context, vid string) ([]*model.VideoSource, error) {
	var items []*model.VideoSource
	err := r.DB(ctx).Where("vid = ? and status = 1", vid).Order("priority ASC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *videoSourceRepository) GetPrimaryByVID(ctx context.Context, vid string) (*model.VideoSource, error) {
	var item model.VideoSource
	err := r.DB(ctx).Where("vid = ? AND status = 1", vid).Order("priority ASC").First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *videoSourceRepository) UpdateUploadStatusByVIDProvider(ctx context.Context, vid string, provider string, status int8) error {
	return r.DB(ctx).Model(&model.VideoSource{}).Where("vid = ? AND provider = ?", vid, provider).Update("upload_status", status).Error
}

func (r *videoSourceRepository) UpdateUploadStatusBySourceID(ctx context.Context, sourceID string, status int8) error {
	return r.DB(ctx).Model(&model.VideoSource{}).Where("source_id = ?", sourceID).Update("upload_status", status).Error
}

func (r *videoSourceRepository) UpdateBySourceID(ctx context.Context, sourceID string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	return r.DB(ctx).Model(&model.VideoSource{}).Where("source_id = ?", sourceID).Updates(fields).Error
}

// DelNotIn hard-deletes sources for vid that are not in keepIDs
func (r *videoSourceRepository) DelNotIn(ctx context.Context, vid string, keepIDs []string) error {
	q := r.DB(ctx).Where("vid = ?", vid)
	if len(keepIDs) > 0 {
		q = q.Where("source_id NOT IN ?", keepIDs)
	}
	return q.Delete(&model.VideoSource{}).Error
}
