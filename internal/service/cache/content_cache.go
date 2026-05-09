package cache

import (
	"context"
	"time"

	"shortpress-server/internal/model"
)

const (
	VideoCachePrefix = "video"
	VideoCacheExpire = 1 * time.Hour
)

type ContentCacheService struct {
	*CacheService
}

func NewContentCacheService(cacheService *CacheService) *ContentCacheService {
	return &ContentCacheService{
		CacheService: cacheService,
	}
}

func (s *ContentCacheService) SetVideo(ctx context.Context, videoID string, video *model.Video) error {
	key := s.BuildKey(VideoCachePrefix, videoID)
	return s.SetCache(ctx, key, video, VideoCacheExpire)
}

func (s *ContentCacheService) GetVideo(ctx context.Context, videoID string) (*model.Video, error) {
	key := s.BuildKey(VideoCachePrefix, videoID)
	var video model.Video
	err := s.GetCache(ctx, key, &video)
	if err != nil {
		return nil, err
	}
	return &video, nil
}
