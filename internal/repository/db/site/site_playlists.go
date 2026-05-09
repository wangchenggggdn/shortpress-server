package site

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm/clause"
)

type SitePlaylistsRepository interface {
	db.BaseOperation
	Delete(ctx context.Context, siteID, playlistID string) error
	// Temporary full retrieval TODO: Need to add caching later
	GetAllPlaylistBySiteID(ctx context.Context, siteID string, status int8) ([]*model.SitePlaylistsView, error)
	GetPlaylistByPage(ctx context.Context, siteID string, status int8, page, pageSize int) ([]*model.SitePlaylistsView, int64, error)
	GetPlaylistByPageWithUtmSource(ctx context.Context, siteID string, status int8, page, pageSize int, utmSource string) ([]*model.SitePlaylistsView, int64, error)
	BatchCreate(ctx context.Context, sitePlaylists []*model.SitePlaylists) error
	BatchDelete(ctx context.Context, siteID string, playlistIDs []string) error
}

type sitePlaylistsRepository struct {
	*db.Repository
}

func NewSitePlaylistRepository(repository *db.Repository) SitePlaylistsRepository {
	return &sitePlaylistsRepository{
		Repository: repository,
	}
}

func (r *sitePlaylistsRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *sitePlaylistsRepository) Update(ctx context.Context, entity interface{}) error {
	// Convert to SitePlaylists
	sitePlaylist := entity.(*model.SitePlaylists)
	return r.DB(ctx).Where("site_id = ? AND playlist_id = ?", sitePlaylist.SiteID, sitePlaylist.PlaylistID).Updates(sitePlaylist).Error
}

// func (r *sitePlaylistsRepository) Update(ctx context.Context, sitePlaylist *model.SitePlaylists) error {
// 	return r.DB(ctx).Where("site_id = ? AND playlist_id = ?", sitePlaylist.SiteID, sitePlaylist.PlaylistID).
// 		Updates(sitePlaylist).Error
// }

func (r *sitePlaylistsRepository) Delete(ctx context.Context, siteID, playlistID string) error {
	return r.DB(ctx).Model(&model.SitePlaylists{}).
		Where("site_id = ? AND playlist_id = ?", siteID, playlistID).
		Update("status", 1).Error
}

func (r *sitePlaylistsRepository) GetAllPlaylistBySiteID(ctx context.Context, siteID string, status int8) ([]*model.SitePlaylistsView, error) {
	var sitePlaylists []*model.SitePlaylistsView

	// Using Table() to explicitly specify the table name, and proper column names
	err := r.DB(ctx).
		Select("site_playlists.*, playlists.access_type, playlists.single_video_price, playlists.free_videos, playlists.title, playlists.slug,playlists.order_vids").
		Table("site_playlists").
		Joins("JOIN playlists ON site_playlists.playlist_id = playlists.playlist_id").
		Where("site_playlists.site_id = ? AND playlists.status = ?", siteID, status).
		Order("site_playlists.created_at DESC").
		Find(&sitePlaylists).Error

	if err != nil {
		return nil, err
	}
	return sitePlaylists, nil
}

// BatchCreate batch create site playlist associations
func (r *sitePlaylistsRepository) BatchCreate(ctx context.Context, sitePlaylists []*model.SitePlaylists) error {
	if len(sitePlaylists) == 0 {
		return nil
	}
	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "site_id"}, {Name: "playlist_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
	}).Create(&sitePlaylists).Error

	// return r.DB(ctx).Create(&sitePlaylists).Error
}

// BatchDelete batch delete site playlist associations
func (r *sitePlaylistsRepository) BatchDelete(ctx context.Context, siteID string, playlistIDs []string) error {
	if len(playlistIDs) == 0 {
		return nil
	}
	return r.DB(ctx).Where("site_id = ? AND playlist_id IN (?)", siteID, playlistIDs).
		Delete(&model.SitePlaylists{}).Error
}

// GetPlaylistByPage retrieves playlists for a site with pagination
// Returns playlists data and total count
func (r *sitePlaylistsRepository) GetPlaylistByPage(ctx context.Context, siteID string, status int8, page, pageSize int) ([]*model.SitePlaylistsView, int64, error) {
	var sitePlaylists []*model.SitePlaylistsView
	var total int64

	// Build query
	db := r.DB(ctx).
		Table("site_playlists").
		Select("site_playlists.*, playlists.access_type, playlists.single_video_price, playlists.free_videos, playlists.title, playlists.slug, playlists.cover, playlists.title, playlists.order_vids").
		Joins("JOIN playlists ON site_playlists.playlist_id = playlists.playlist_id").
		Where("site_playlists.site_id = ? AND playlists.status = ?", siteID, status)

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Handle pagination parameters
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = int(total) // Return all if pageSize is not specified
	}
	if pageSize > 100 {
		pageSize = 100 // Limit max page size
	}

	// Get paginated data
	err := db.
		Order("site_playlists.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&sitePlaylists).Error

	if err != nil {
		return nil, 0, err
	}

	return sitePlaylists, total, nil
}

// GetPlaylistByPageWithUtmSource retrieves playlists for a site with pagination and utm_source filtering
// Returns playlists data and total count
func (r *sitePlaylistsRepository) GetPlaylistByPageWithUtmSource(ctx context.Context, siteID string, status int8, page, pageSize int, utmSource string) ([]*model.SitePlaylistsView, int64, error) {
	var sitePlaylists []*model.SitePlaylistsView
	var total int64

	// Build query
	db := r.DB(ctx).
		Table("site_playlists").
		Select("site_playlists.*, playlists.access_type, playlists.single_video_price, playlists.free_videos, playlists.title, playlists.slug, playlists.cover, playlists.title, playlists.order_vids").
		Joins("JOIN playlists ON site_playlists.playlist_id = playlists.playlist_id").
		Where("site_playlists.site_id = ? AND playlists.status = ?", siteID, status)

	// Apply utm_source filter
	if utmSource == "" {
		// Header透传''，对应数据库的'none'和''（所有人可见）
		db = db.Where("(playlists.utm_source = ? OR FIND_IN_SET(?, playlists.utm_source) > 0 OR playlists.utm_source IS NULL)", "", "none")
	} else {
		// Header透传具体渠道，如'm1'
		db = db.Where("(playlists.utm_source = ? OR FIND_IN_SET(?, playlists.utm_source) > 0 OR playlists.utm_source IS NULL)", "", utmSource)
	}

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Handle pagination parameters
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = int(total) // Return all if pageSize is not specified
	}
	if pageSize > 100 {
		pageSize = 100 // Limit max page size
	}

	// Get paginated data
	err := db.
		Order("site_playlists.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&sitePlaylists).Error

	if err != nil {
		return nil, 0, err
	}

	return sitePlaylists, total, nil
}
