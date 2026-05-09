package playlist

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

type PlaylistRepository interface {
	db.BaseOperation
	GetByPlaylistID(ctx context.Context, playlistId string) (*model.Playlist, error)
	GetByPlaylistIDWithUtmSource(ctx context.Context, playlistId string, utmSource string) (*model.Playlist, error)
	GetBySiteAndPlaylistID(ctx context.Context, siteID string, playlistId string) (*model.Playlist, error)
	GetByPlaylistIDForUpdate(ctx context.Context, playlistId string) (*model.Playlist, error)
	GetByPlaylistIDS(ctx context.Context, playlistIds []string) ([]*model.Playlist, error)
	GetByPlaylistIDSWithUtmSource(ctx context.Context, playlistIds []string, utmSource string) ([]*model.Playlist, error)
	GetOrderVideos(ctx context.Context, playlistId string) (string, error)
	BatchDelete(ctx context.Context, creatorID string, playlistId []string) error
	ListByPage(ctx context.Context, query *model.PlaylistQuery, page int, pageSize int, orderType int) ([]*model.Playlist, int64, error)
	SearchPlaylistIDsWithI18n(ctx context.Context, siteID, keyword, lang string, status *int, page, pageSize, orderType int) ([]string, int64, error)
	Count(ctx context.Context, creatorID string) (int64, error)
	CountByPlaylistIDs(ctx context.Context, playlistIDs []string) (int64, error)
	UpdateWithVersion(ctx context.Context, playlist *model.Playlist) (int, error)
	GetBySiteAndTag(ctx context.Context, siteID string, tag string) ([]*model.Playlist, error)
}

func NewPlaylistRepository(
	repository *db.Repository,
) PlaylistRepository {
	return &playlistRepository{
		Repository: repository,
	}
}

type playlistRepository struct {
	*db.Repository
}

func (r *playlistRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *playlistRepository) Update(ctx context.Context, entity interface{}) error {
	// Convert to playlist
	playlist := entity.(*model.Playlist)
	return r.DB(ctx).Where("playlist_id = ?", playlist.PlaylistID).Updates(playlist).Error
}

// GetPlaylist query playlist by ID
func (r *playlistRepository) GetByPlaylistID(ctx context.Context, playlistId string) (*model.Playlist, error) {
	var playlist model.Playlist
	err := r.DB(ctx).Where("playlist_id = ? ", playlistId).First(&playlist).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &playlist, nil
}

// GetByPlaylistIDWithUtmSource 根据 playlist ID 和 utm_source 查询
func (r *playlistRepository) GetByPlaylistIDWithUtmSource(ctx context.Context, playlistId string, utmSource string) (*model.Playlist, error) {
	var playlist model.Playlist
	query := r.DB(ctx).Where("playlist_id = ?", playlistId)

	// Header透传''，对应数据库的'none'和''（所有人可见）
	if utmSource == "" {
		query = query.Where("(utm_source = ? OR FIND_IN_SET(?, utm_source) > 0 OR utm_source IS NULL)", "", "none")
	} else {
		// Header透传具体渠道，如'm1'
		query = query.Where("(utm_source = ? OR FIND_IN_SET(?, utm_source) > 0 OR utm_source IS NULL)", "", utmSource)
	}

	err := query.First(&playlist).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &playlist, nil
}

// With row lock
func (r *playlistRepository) GetByPlaylistIDForUpdate(ctx context.Context, playlistId string) (*model.Playlist, error) {
	var playlist model.Playlist
	err := r.DB(ctx).Model(&model.Playlist{}).Where("playlist_id = ? FOR UPDATE", playlistId).Find(&playlist).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &playlist, nil
}

// DeletePlaylist delete playlist (soft delete)
func (r *playlistRepository) BatchDelete(ctx context.Context, creatorID string, playlistIds []string) error {
	return r.DB(ctx).Model(&model.Playlist{}).
		Where("creator_id = ? AND playlist_id IN (?)", creatorID, playlistIds).
		Update("status", model.PlaylistStatusDeleted).Error
}

// BatchGetPlaylists batch query playlists
func (r *playlistRepository) GetByPlaylistIDS(ctx context.Context, playlistIds []string) ([]*model.Playlist, error) {
	if len(playlistIds) == 0 {
		return nil, nil
	}
	var playlists []*model.Playlist
	err := r.DB(ctx).Where("playlist_id IN (?) AND status != ?", playlistIds, model.PlaylistStatusDeleted).Find(&playlists).Error
	if err != nil {
		return nil, err
	}
	return playlists, nil
}

// GetByPlaylistIDSWithUtmSource 根据 playlist IDs 和 utm_source 批量查询
func (r *playlistRepository) GetByPlaylistIDSWithUtmSource(ctx context.Context, playlistIds []string, utmSource string) ([]*model.Playlist, error) {
	if len(playlistIds) == 0 {
		return nil, nil
	}
	var playlists []*model.Playlist
	query := r.DB(ctx).Where("playlist_id IN (?) AND status != ?", playlistIds, model.PlaylistStatusDeleted)

	// Header透传''，对应数据库的'none'和''（所有人可见）
	if utmSource == "" {
		query = query.Where("(utm_source = ? OR FIND_IN_SET(?, utm_source) > 0 OR utm_source IS NULL)", "", "none")
	} else {
		query = query.Where("(utm_source = ? OR FIND_IN_SET(?, utm_source) > 0 OR utm_source IS NULL)", "", utmSource)
	}

	err := query.Find(&playlists).Error
	if err != nil {
		return nil, err
	}
	return playlists, nil
}

// ListPlaylistsWithPage paginated query of playlists
func (r *playlistRepository) ListPlaylistsWithPage(ctx context.Context, creatorID string, page, pageSize, status int, orderType int) ([]*model.Playlist, int64, error) {
	var playlists []*model.Playlist
	var total int64

	// Build base query
	query := r.DB(ctx).Model(&model.Playlist{})

	// Add status filter
	if status > 0 {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status != ?", model.PlaylistStatusDeleted) // Don't return deleted data
	}
	query = query.Where("creator_id = ?", creatorID)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Add pagination and sorting
	offset := (page - 1) * pageSize
	query = query.Offset(offset).Limit(pageSize)

	// Add sort conditions based on sort type
	switch orderType {
	case 1: // Sort by title
		query = query.Order("title ASC")
	default: // Default sort by creation time descending
		query = query.Order("created_at DESC")
	}

	// Execute query
	if err := query.Find(&playlists).Error; err != nil {
		return nil, 0, err
	}

	return playlists, total, nil
}

func (r *playlistRepository) ListByPage(ctx context.Context, query *model.PlaylistQuery, page int, pageSize int, orderType int) ([]*model.Playlist, int64, error) {

	db := r.DB(ctx).Model(&model.Playlist{})
	var total int64

	if query.CreatorID != "" {
		db = db.Where("creator_id = ?", query.CreatorID)
	}
	if query.Status != nil && *query.Status != -1 {
		db = db.Where("playlists.status = ?", query.Status)
	} else {
		db = db.Where("playlists.status != ?", model.PlaylistStatusDeleted) // Don't return deleted data
	}
	if query.SiteID != "" {
		db = db.Joins("INNER JOIN site_playlists ON playlists.playlist_id = site_playlists.playlist_id").
			Where("site_playlists.site_id = ?", query.SiteID)
	}

	if query.ExcludeSiteId != "" {
		db = db.Where("NOT EXISTS (SELECT 1 FROM site_playlists WHERE site_playlists.playlist_id = playlists.playlist_id AND site_playlists.site_id = ?)", query.ExcludeSiteId)
	}

	if query.Keyword != "" {
		db = db.Where("title LIKE ? OR description LIKE ? OR tags LIKE ?", "%"+query.Keyword+"%", "%"+query.Keyword+"%", "%"+query.Keyword+"%")
	}

	// UTM 来源强制过滤
	if query.FilterUtm {
		if query.UtmSource == "" {
			db = db.Where("(utm_source = ? OR FIND_IN_SET(?, utm_source) > 0 OR utm_source IS NULL)", "", "none")
		} else {
			db = db.Where("(utm_source = ? OR FIND_IN_SET(?, utm_source) > 0 OR utm_source IS NULL)", "", query.UtmSource)
		}
	} else if query.UtmSource != "" {
		// 对于未开启强制过滤（如Creator后台）保留原本逻辑
		db = db.Where("utm_source LIKE ?", "%"+query.UtmSource+"%")
	}

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Add sorting
	switch orderType {
	case model.PlaylistSortByCreatedAtDesc:
		db = db.Order("created_at DESC")
	case model.PlaylistSortByTitleAsc:
		db = db.Order("title ASC")
	case model.PlaylistSortByCreatedAtAsc:
		db = db.Order("created_at ASC")
	case model.PlaylistSortByTitleDesc:
		db = db.Order("title DESC")
	default:
		db = db.Order("created_at DESC") // Default sort
	}

	// Add pagination and sorting
	var playlists []*model.Playlist
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&playlists).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
	}
	return playlists, total, nil
}

func (r *playlistRepository) Count(ctx context.Context, creatorID string) (int64, error) {
	var count int64
	if err := r.DB(ctx).Model(&model.Playlist{}).
		Where("creator_id = ? AND status != ?", creatorID, model.PlaylistStatusDeleted).
		Count(&count).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func (r *playlistRepository) GetOrderVideos(ctx context.Context, playlistId string) (string, error) {
	var orderVideos string
	err := r.DB(ctx).Model(&model.Playlist{}).
		Where("playlist_id = ? ", playlistId).
		Select("order_vids").Scan(&orderVideos).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return orderVideos, nil
}

func (r *playlistRepository) UpdateWithVersion(ctx context.Context, playlist *model.Playlist) (int, error) {
	// Run the update with version checking
	result := r.DB(ctx).Model(&model.Playlist{}).
		Where("playlist_id = ? AND version = ?", playlist.PlaylistID, playlist.Version).
		Updates(map[string]interface{}{
			"order_vids": playlist.OrderVids,
			"version":    gorm.Expr("version + 1"), // Increment version
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

func (r *playlistRepository) CountByPlaylistIDs(ctx context.Context, playlistIDs []string) (int64, error) {
	var count int64
	if err := r.DB(ctx).Model(&model.Playlist{}).
		Where("playlist_id IN (?)", playlistIDs).
		Count(&count).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func (r *playlistRepository) GetBySiteAndTag(ctx context.Context, siteID string, tag string) ([]*model.Playlist, error) {
	var playlists []*model.Playlist
	err := r.DB(ctx).
		Joins("INNER JOIN site_playlists ON playlists.playlist_id = site_playlists.playlist_id").
		Where("site_playlists.site_id = ? AND playlists.tags LIKE ? AND playlists.status != ? order by playlists.created_at desc limit 10", siteID, "%"+tag+"%", model.PlaylistStatusDeleted).
		Find(&playlists).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return playlists, nil
}

func (r *playlistRepository) GetBySiteAndPlaylistID(ctx context.Context, siteID string, playlistId string) (*model.Playlist, error) {
	var playlist model.Playlist
	err := r.DB(ctx).
		Joins("INNER JOIN site_playlists ON playlists.playlist_id = site_playlists.playlist_id").
		Where("site_playlists.site_id = ? AND playlists.playlist_id = ? AND playlists.status != ?", siteID, playlistId, model.PlaylistStatusDeleted).
		First(&playlist).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &playlist, nil
}

// SearchPlaylistIDsWithI18n 搜索支持多语言的 playlistID
// 已翻译：在 i18n 表中搜索（限定语言）
// 未翻译：在 playlist 表中搜索（但该 playlistID 在 i18n 表中没有任何记录）
func (r *playlistRepository) SearchPlaylistIDsWithI18n(
	ctx context.Context,
	siteID, keyword, lang string,
	status *int,
	page, pageSize, orderType int,
) ([]string, int64, error) {

	var playlistIDs []string
	var total int64

	keywordPattern := "%" + keyword + "%"

	// 构建 UNION 查询
	unionQuery := r.DB(ctx).Raw(`
		SELECT playlist_id FROM (
			SELECT DISTINCT
				p.playlist_id,
				p.created_at,
				p.title
			FROM playlists p
			INNER JOIN site_playlists sp ON p.playlist_id = sp.playlist_id
			INNER JOIN playlist_i18n i18n ON p.playlist_id = i18n.playlist_id
			WHERE
				sp.site_id = ?
				AND p.status != ?
				AND i18n.language = ?
				AND (i18n.title LIKE ? OR i18n.description LIKE ?)

			UNION

			SELECT
				p.playlist_id,
				p.created_at,
				p.title
			FROM playlists p
			INNER JOIN site_playlists sp ON p.playlist_id = sp.playlist_id
			LEFT JOIN playlist_i18n i18n ON p.playlist_id = i18n.playlist_id
			WHERE
				sp.site_id = ?
				AND p.status != ?
				AND i18n.id IS NULL
				AND (p.title LIKE ? OR p.description LIKE ?)
		) AS combined
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`,
		siteID, model.PlaylistStatusDeleted, lang, keywordPattern, keywordPattern,
		siteID, model.PlaylistStatusDeleted, keywordPattern, keywordPattern,
		pageSize, (page-1)*pageSize,
	)

	// 获取总数
	countQuery := r.DB(ctx).Raw(`
		SELECT COUNT(*) FROM (
			SELECT DISTINCT p.playlist_id
			FROM playlists p
			INNER JOIN site_playlists sp ON p.playlist_id = sp.playlist_id
			INNER JOIN playlist_i18n i18n ON p.playlist_id = i18n.playlist_id
			WHERE
				sp.site_id = ?
				AND p.status != ?
				AND i18n.language = ?
				AND (i18n.title LIKE ? OR i18n.description LIKE ?)

			UNION

			SELECT p.playlist_id
			FROM playlists p
			INNER JOIN site_playlists sp ON p.playlist_id = sp.playlist_id
			LEFT JOIN playlist_i18n i18n ON p.playlist_id = i18n.playlist_id
			WHERE
				sp.site_id = ?
				AND p.status != ?
				AND i18n.id IS NULL
				AND (p.title LIKE ? OR p.description LIKE ?)
		) AS combined
	`,
		siteID, model.PlaylistStatusDeleted, lang, keywordPattern, keywordPattern,
		siteID, model.PlaylistStatusDeleted, keywordPattern, keywordPattern,
	).Count(&total)

	if countQuery.Error != nil {
		return nil, 0, countQuery.Error
	}

	// 执行主查询
	err := unionQuery.Scan(&playlistIDs).Error
	if err != nil {
		return nil, 0, err
	}

	return playlistIDs, total, nil
}

// applyOrdering 应用排序
func (r *playlistRepository) applyOrdering(query *gorm.DB, orderType int) *gorm.DB {
	switch orderType {
	case 0: // PlaylistSortByCreatedAtDesc
		query = query.Order("created_at DESC")
	case 1: // PlaylistSortByCreatedAtAsc
		query = query.Order("created_at ASC")
	case 2: // PlaylistSortByTitleAsc
		query = query.Order("title ASC")
	case 3: // PlaylistSortByTitleDesc
		query = query.Order("title DESC")
	default:
		query = query.Order("created_at DESC")
	}
	return query
}
