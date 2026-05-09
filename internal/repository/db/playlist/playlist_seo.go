package playlist

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PlaylistSeoRepository interface {
	db.BaseOperation
	GetSeo(ctx context.Context, playlistID string) (*model.PlaylistSeo, error)
	Save(ctx context.Context, seo *model.PlaylistSeo) error
	FindSeo(ctx context.Context, playlistID string) (*model.PlaylistSeo, error)
}

type playlistSeoRepository struct {
	*db.Repository
}

func NewPlaylistSeoRepository(repository *db.Repository) PlaylistSeoRepository {
	return &playlistSeoRepository{
		Repository: repository,
	}
}

func (r *playlistSeoRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *playlistSeoRepository) Update(ctx context.Context, entity interface{}) error {
	// Convert to PlaylistSeo
	playlistSeo := entity.(*model.PlaylistSeo)
	return r.DB(ctx).Where("playlist_id = ?", playlistSeo.PlaylistID).Updates(playlistSeo).Error
}

// GetSeo Get playlist SEO information
func (r *playlistSeoRepository) GetSeo(ctx context.Context, playlistID string) (*model.PlaylistSeo, error) {
	var seo model.PlaylistSeo
	err := r.DB(ctx).Where("playlist_id = ?", playlistID).First(&seo).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &seo, nil
}

// FindSeo Find playlist SEO information
func (r *playlistSeoRepository) FindSeo(ctx context.Context, playlistID string) (*model.PlaylistSeo, error) {
	var seo model.PlaylistSeo
	err := r.DB(ctx).Where("playlist_id = ?", playlistID).First(&seo).Error
	if err != nil {
		return nil, err
	}
	return &seo, nil
}

func (r *playlistSeoRepository) Save(ctx context.Context, seo *model.PlaylistSeo) error {
	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "playlist_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "description", "keywords", "updated_at"}),
	}).Create(seo).Error
}
