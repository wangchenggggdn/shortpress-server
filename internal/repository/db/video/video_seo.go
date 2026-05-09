package video

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VideoSeoRepository interface {
	db.BaseOperation
	GetByVID(ctx context.Context, vid string) (*model.VideoSeo, error)
	Save(ctx context.Context, seo *model.VideoSeo) error
	Delete(ctx context.Context, vid string) error
}

type videoSeoRepository struct {
	*db.Repository
}

func NewVideoSeoRepository(repository *db.Repository) VideoSeoRepository {
	return &videoSeoRepository{
		Repository: repository,
	}
}

func (r *videoSeoRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *videoSeoRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Updates(entity).Error
}

func (r *videoSeoRepository) GetByVID(ctx context.Context, vid string) (*model.VideoSeo, error) {
	var seo model.VideoSeo
	err := r.DB(ctx).Where("vid = ?", vid).First(&seo).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &seo, nil
}

func (r *videoSeoRepository) Save(ctx context.Context, seo *model.VideoSeo) error {
	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "vid"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "description", "keywords", "updated_at"}),
	}).Create(seo).Error
}

func (r *videoSeoRepository) Delete(ctx context.Context, vid string) error {
	return r.DB(ctx).Where("vid = ?", vid).Delete(&model.VideoSeo{}).Error
}
