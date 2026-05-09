package video

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

type VideoRepository interface {
	db.BaseOperation
	GetByVID(ctx context.Context, vid string) (*model.Video, error)
	GetByVIDs(ctx context.Context, vids []string) ([]*model.Video, error)
	GetVideoAndSeo(ctx context.Context, vid string) (*model.Video, *model.VideoSeo, error)
	Delete(ctx context.Context, vid string) error
	BatchDelete(ctx context.Context, vid []string) error
	ListByPage(ctx context.Context, query *model.VideoQuery, page int, pageSize int, sortType int) ([]*model.Video, int64, error)
	ListByPageV2(ctx context.Context, query *model.VideoQuery, page int, pageSize int, sortType int) ([]*model.Video, int64, error)

	UpdateUploadStatus(ctx context.Context, vid string, status int8) error
	Count(ctx context.Context, query *model.VideoQuery) (int64, error)
}

func NewVideoRepository(
	repository *db.Repository,
) VideoRepository {
	return &videoRepository{
		Repository: repository,
	}
}

type videoRepository struct {
	*db.Repository
}

func (r *videoRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *videoRepository) Update(ctx context.Context, entity interface{}) error {
	switch v := entity.(type) {
	case *model.VideoCore:
		return r.DB(ctx).Model(&model.VideoCore{}).Where("vid = ?", v.VID).Updates(v).Error
	case *model.Video:
		// Only update columns that still exist in videos table
		data := map[string]interface{}{}
		if v.Title != "" {
			data["title"] = v.Title
		}
		if v.Description != "" {
			data["description"] = v.Description
		}
		if v.Tags != "" {
			data["tags"] = v.Tags
		}
		if v.Cover != nil {
			data["cover"] = v.Cover
		}
		if v.Config != nil {
			data["config"] = v.Config
		}
		if v.Status != 0 {
			data["status"] = v.Status
		}
		if v.Subtitles != nil {
			data["subtitles"] = v.Subtitles
		}
		if len(data) == 0 {
			return nil
		}
		return r.DB(ctx).Model(&model.VideoCore{}).Where("vid = ?", v.VID).Updates(data).Error
	default:
		return gorm.ErrInvalidData
	}
}

// Implement FindByID
func (r *videoRepository) FindByID(ctx context.Context, id interface{}) (*model.Video, error) {
	var video model.Video
	err := r.DB(ctx).Where("id = ?", id).First(&video).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *videoRepository) GetByVID(ctx context.Context, vid string) (*model.Video, error) {
	// Join primary source to populate legacy fields
	var video model.Video
	err := r.DB(ctx).Table("videos v").Select(
		"v.id, v.vid, v.tags, v.title, v.description, v.cover, v.status, v.config, v.creator_id, v.created_at, v.updated_at, "+
			"vs.upload_status as upload_status, "+
			"COALESCE(vs.url, vs.local_path) as video_path, "+
			"vs.duration, vs.width, vs.height, 0 as file_size").
		Joins("LEFT JOIN video_sources vs ON vs.vid = v.vid AND vs.status = 1").
		Where("v.vid = ?", vid).
		Order("vs.priority ASC").
		Limit(1).
		Scan(&video).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &video, nil
}

func (r *videoRepository) Delete(ctx context.Context, vid string) error {
	return r.DB(ctx).Model(&model.Video{}).Where("vid = ?", vid).Update("status", model.VideoStatusDeleted).Error
}

func (r *videoRepository) ListByPage(ctx context.Context, query *model.VideoQuery, page int, pageSize int, sortType int) ([]*model.Video, int64, error) {
	db := r.DB(ctx).Table("videos")
	if query.CreatorID != "" {
		db = db.Where("creator_id = ?", query.CreatorID)
	}
	if query.PlaylistID != "" {
		db = db.Joins("INNER JOIN playlist_vid ON videos.vid = playlist_vid.vid").
			Where("playlist_vid.playlist_id = ?", query.PlaylistID)
	}

	if query.Status != nil && *query.Status != -1 {
		db = db.Where("videos.status = ?", *query.Status)
	} else {
		db = db.Where("videos.status != ?", model.VideoStatusDeleted) // Don't return deleted data
	}
	if query.KeyWord != "" {
		db = db.Where("title LIKE ?", "%"+query.KeyWord+"%")
	}
	if query.UploadStatus != nil && *query.UploadStatus != -1 {
		db = db.Joins("LEFT JOIN video_sources vs ON vs.vid = videos.vid").Where("vs.upload_status = ?", *query.UploadStatus)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	// Add sorting
	switch sortType {
	case model.VideoSortByCreatedAtDesc:
		db = db.Order("created_at DESC")
	case model.VideoSortTitleAsc:
		db = db.Order("title ASC")
	case model.VideoSortByCreatedAtAsc:
		db = db.Order("created_at ASC")
	case model.VideoSortByTitleDesc:
		db = db.Order("title DESC")
	default:
		db = db.Order("created_at DESC") // Default sort
	}
	// Pagination query
	var videos []*model.Video
	err := db.Select("videos.id, videos.vid, videos.tags, videos.title, videos.description, videos.cover, videos.status, videos.config, videos.creator_id, videos.created_at, videos.updated_at").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&videos).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
	}
	return videos, total, err
}

func (r *videoRepository) ListByPageV2(ctx context.Context, query *model.VideoQuery, page int, pageSize int, sortType int) ([]*model.Video, int64, error) {
	db := r.DB(ctx).Table("videos")
	if query.CreatorID != "" {
		db = db.Where("creator_id = ?", query.CreatorID)
	}
	if query.PlaylistID != "" {
		db = db.Joins("INNER JOIN playlist_vid ON videos.vid = playlist_vid.vid").
			Where("playlist_vid.playlist_id = ?", query.PlaylistID)
	}

	if query.Status != nil && *query.Status != -1 {
		db = db.Where("videos.status = ?", *query.Status)
	} else {
		db = db.Where("videos.status != ?", model.VideoStatusDeleted) // Don't return deleted data
	}
	if query.KeyWord != "" {
		db = db.Where("title LIKE ? OR tags LIKE ? OR description LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}
	//TODO 如果vid 够多，mysql 是否可能会过长？
	if len(query.ExcludeVids) > 0 {
		db = db.Where("vid NOT IN ?", query.ExcludeVids)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	// Add sorting
	switch sortType {
	case model.VideoSortByCreatedAtDesc:
		db = db.Order("created_at DESC")
	case model.VideoSortTitleAsc:
		db = db.Order("title ASC")
	case model.VideoSortByCreatedAtAsc:
		db = db.Order("created_at ASC")
	case model.VideoSortByTitleDesc:
		db = db.Order("title DESC")
	default:
		db = db.Order("created_at DESC") // Default sort
	}
	// Pagination query
	var videos []*model.Video
	err := db.Select("videos.vid, videos.status").Offset((page - 1) * pageSize).Limit(pageSize).Find(&videos).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, nil
		}
	}
	return videos, total, err
}

// UpdateUploadStatus updates video upload status
func (r *videoRepository) UpdateUploadStatus(ctx context.Context, vid string, status int8) error {
	// Update upload status for local provider source by default
	return r.DB(ctx).Model(&model.VideoSource{}).
		Where("vid = ? AND provider = ?", vid, "local").
		Update("upload_status", status).
		Error
}

func (r *videoRepository) GetVideoAndSeo(ctx context.Context, vid string) (*model.Video, *model.VideoSeo, error) {
	// Query video and SEO information with primary source
	var video model.Video
	var seo model.VideoSeo
	err := r.DB(ctx).Table("videos v").
		Select("v.id, v.vid, v.tags, v.title, v.description, v.cover, v.status, v.config, v.creator_id, v.created_at, v.updated_at, "+
			"vs.upload_status as upload_status, COALESCE(vs.url, vs.local_path) as video_path, vs.duration, vs.width, vs.height").
		Joins("LEFT JOIN video_seo ON v.vid = video_seo.vid").
		Joins("LEFT JOIN video_sources vs ON vs.vid = v.vid AND vs.status = 1").
		Where("v.vid = ?", vid).
		Order("vs.priority ASC").
		Limit(1).
		Scan(&video).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return &video, &seo, nil

}

func (r *videoRepository) Count(ctx context.Context, query *model.VideoQuery) (int64, error) {
	var count int64
	db := r.DB(ctx).Table("videos")
	if query.CreatorID != "" {
		db = db.Where("creator_id = ?", query.CreatorID)
	}
	if query.PlaylistID != "" {
		db = db.Joins("INNER JOIN playlist_vid ON videos.vid = playlist_vid.vid").
			Where("playlist_vid.playlist_id = ?", query.PlaylistID)
	}
	if query.Status != nil && *query.Status != -1 {
		db = db.Where("videos.status = ?", *query.Status)
	} else {
		db = db.Where("videos.status != ?", model.VideoStatusDeleted) // Don't return deleted data
	}
	if query.UploadStatus != nil && *query.UploadStatus != -1 {
		db = db.Joins("LEFT JOIN video_sources vs ON vs.vid = videos.vid").Where("vs.upload_status = ?", *query.UploadStatus)
	}

	if err := db.Count(&count).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func (r *videoRepository) GetByVIDs(ctx context.Context, vids []string) ([]*model.Video, error) {
	var videos []*model.Video
	if len(vids) == 0 {
		return videos, nil
	}
	err := r.DB(ctx).Table("videos v").
		Select("v.vid, v.title, v.tags, v.description, v.cover, v.status, v.created_at, v.updated_at, v.config, v.subtitles, "+
			"vs.upload_status as upload_status, COALESCE(vs.url, vs.local_path) as video_path, vs.duration, vs.width, vs.height").
		Joins("LEFT JOIN video_sources vs ON vs.vid = v.vid AND vs.status = 1").
		Where("v.vid IN ?", vids).
		Scan(&videos).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return videos, err
}

func (r *videoRepository) BatchDelete(ctx context.Context, vids []string) error {
	if len(vids) == 0 {
		return nil
	}
	return r.DB(ctx).Model(&model.Video{}).Where("vid IN ?", vids).Update("status", model.VideoStatusDeleted).Error
}
