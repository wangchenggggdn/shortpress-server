package playlist

import (
	"context"
	"fmt"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm/clause"
)

// SortNumberStep is the increment value used for sorting videos in a playlist
const SortNumberStep = 1000.0

type PlaylistVidRepository interface {
	db.BaseOperation
	Delete(ctx context.Context, playlistID, vid string) error
	BatchDelete(ctx context.Context, playlistID string, vids []string) error
	BatchDeleteByVids(ctx context.Context, vids []string) error
	BatchAdd(ctx context.Context, records []*model.PlaylistVid) error
	GetVideosByPlaylistID(ctx context.Context, playlistID string) ([]*model.VideoSView, int64, error)
	Count(ctx context.Context, playlistID string) (int64, error)
	UpdateSortNumber(ctx context.Context, playlistID string, vid string, sortNumber float64) error
}

type playlistVidRepository struct {
	*db.Repository
}

func NewPlaylistVidRepository(repository *db.Repository) PlaylistVidRepository {
	return &playlistVidRepository{
		Repository: repository,
	}
}

func (r *playlistVidRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *playlistVidRepository) Update(ctx context.Context, entity interface{}) error {
	playlistVid := entity.(*model.PlaylistVid)
	return r.DB(ctx).Where("playlist_id = ? AND vid = ?", playlistVid.PlaylistID, playlistVid.VID).Updates(playlistVid).Error
}

func (r *playlistVidRepository) BatchAdd(ctx context.Context, records []*model.PlaylistVid) error {
	if len(records) == 0 {
		return fmt.Errorf("empty records")
	}
	// playlistID := records[0].PlaylistID
	// var maxOrder int
	// if err := r.DB(ctx).Model(&model.PlaylistVid{}).
	//     Where("playlist_id = ?", playlistID)
	// }
	// for i, record := range records {
	//     record.SortOrder = maxOrder + i + 1
	// }

	return r.DB(ctx).Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "playlist_id"}, {Name: "vid"}},
			DoNothing: true,
		},
	).CreateInBatches(records, len(records)).Error

}

func (r *playlistVidRepository) Delete(ctx context.Context, playlistID, vid string) error {
	return r.DB(ctx).Where("playlist_id = ? AND vid = ?", playlistID, vid).Delete(&model.PlaylistVid{}).Error
}

func (r *playlistVidRepository) BatchDelete(ctx context.Context, playlistID string, vids []string) error {
	return r.DB(ctx).Where("playlist_id = ? AND vid IN (?)", playlistID, vids).Delete(&model.PlaylistVid{}).Error
}

func (r *playlistVidRepository) GetOrderNumber(ctx context.Context, playlistID string) (int64, error) {
	var maxOrder int64
	err := r.DB(ctx).Model(&model.PlaylistVid{}).
		Where("playlist_id = ?", playlistID).
		Select("COALESCE(MAX(`order`), 0)").
		Scan(&maxOrder).Error
	return maxOrder, err
}

func (r *playlistVidRepository) GetVideosByPlaylistID(ctx context.Context, playlistID string) ([]*model.VideoSView, int64, error) {
	var videos []*model.VideoSView
	var total int64

	// First get count of total records
	if err := r.DB(ctx).Model(&model.PlaylistVid{}).Table("playlist_vid").
		Where("playlist_id = ?", playlistID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.DB(ctx).Model(&model.VideoSView{}).Table("playlist_vid").
		Select("videos.vid, videos.title, videos.description, videos.cover, videos.video_path, playlist_vid.`order` as sort_order").
		Joins("JOIN videos ON playlist_vid.vid = videos.vid").
		Where("playlist_vid.playlist_id = ?", playlistID).
		Order("playlist_vid.`order` ASC").
		Find(&videos).Error

	if err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

func (r *playlistVidRepository) BatchDeleteByVids(ctx context.Context, vids []string) error {
	return r.DB(ctx).Where("vid IN (?)", vids).Delete(&model.PlaylistVid{}).Error
}

func (r *playlistVidRepository) Count(ctx context.Context, playlistID string) (int64, error) {
	var total int64
	if err := r.DB(ctx).Model(&model.PlaylistVid{}).
		Where("playlist_id = ?", playlistID).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *playlistVidRepository) UpdateSortNumber(ctx context.Context, playlistID string, vid string, sortNumber float64) error {
	// Update sort_number for the specific video in the playlist
	return r.DB(ctx).Table("playlist_vid").
		Where("playlist_id = ? AND vid = ?", playlistID, vid).
		Update("sort_number", sortNumber).Error
}
