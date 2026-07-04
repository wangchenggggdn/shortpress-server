package service

import (
	"encoding/json"
	"fmt"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/playlist"
	"shortpress-server/internal/repository/db/site"
	"shortpress-server/internal/repository/db/user"
	"shortpress-server/internal/repository/db/video"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/exp/rand"
)

type ClientPlayerService interface {
	SimpleFeed(ctx *gin.Context, siteID string, userID string, pageSize int) (*api.FeedResponseData, error)
	SimpleFeedV2(ctx *gin.Context, siteID string, userID string, page, pageSize int) (*api.FeedResponseData, error)

	// GetPlaylistInfo Get playlist details
	GetPlaylistInfo(ctx *gin.Context, playlistID string, lang string, needVideo bool, userID string) (*api.PlaylistInfo, error)
	// BatchGetPlaylistInfo Batch get playlist details by IDs
	BatchGetPlaylistInfo(ctx *gin.Context, playlistIDs []string, lang string, needVideo bool, userID string) ([]*api.PlaylistInfo, error)
	// GetVideosByPlaylistID Return all video IDs in a playlist according to playback order
	GetVideosByPlaylistID(ctx *gin.Context, playlistID string) (*api.VideoSortData, *model.Playlist, error)
	GetVideosAndStatusByPlaylistID(ctx *gin.Context, playlistID string, userID string) ([]*api.VideoItem, error)
	GetVideoUnlockMap(ctx *gin.Context, playlistID string, userID string) (map[string]string, error)
	// AnonRegister Anonymous registration
	AnonRegister(ctx *gin.Context, req *api.AnonRegisterRequest) (string, string, error)
	// GetSiteByPath Get site information by path
	GetSiteByPath(ctx *gin.Context, sitePath string) (*model.Site, error)
	GetSiteByHost(ctx *gin.Context, host string) (*model.Site, error)
	// CreateOrUpdatePlayRecord User play records
	CreateOrUpdatePlayRecord(ctx *gin.Context, record *model.UserPlayRecord) error
	GetUserPlayHistory(ctx *gin.Context, userID, siteID string, page, pageSize int) ([]*model.UserPlayRecord, int64, error)
	// GetAllPlaylistIDs Get all playlist IDs for a site with pagination and i18n support
	GetAllPlaylistIDs(ctx *gin.Context, siteID string, page, pageSize int) ([]*api.PlaylistSlugItem, int64, error)
	// GetSitePlist Get published playlists for a site, excluding specific utm_source
	GetSitePlist(ctx *gin.Context, siteID string, page, pageSize, orderType int) ([]*api.PlaylistInfo, int64, error)
}

func NewClientPlayerService(
	service *Service,
	siteRepository site.SiteRepository,
	sitePlaylistsRepository site.SitePlaylistsRepository,
	videoRepository video.VideoRepository,
	playlistRepository playlist.PlaylistRepository,
	playlistI18nRepository playlist.PlaylistI18nRepository,
	playlistSeoRepository playlist.PlaylistSeoRepository,
	playlistVidRepository playlist.PlaylistVidRepository,
	siteSeoRepository site.SiteSeoRepository,
	sitePageTemplateRepository site.SitePageTemplateRepository,
	userRepository user.UserRepository,
	contentUnlockRepository payment.ContentUnlockRepository,
	userPlayRecordsRepository user.UserPlayRecordsRepository,
) ClientPlayerService {
	return &clientPlayerService{
		Service:                    service,
		siteRepository:             siteRepository,
		sitePlaylistsRepository:    sitePlaylistsRepository,
		videoRepository:            videoRepository,
		playlistRepository:         playlistRepository,
		playlistI18nRepository:     playlistI18nRepository,
		playlistVidRepository:      playlistVidRepository,
		PlaylistSeoRepository:      playlistSeoRepository,
		siteSeoRepository:          siteSeoRepository,
		sitePageTemplateRepository: sitePageTemplateRepository,
		contentUnlockRepository:    contentUnlockRepository,
		userRepository:             userRepository,
		userPlayRecordsRepository:  userPlayRecordsRepository,
	}
}

type clientPlayerService struct {
	*Service
	siteRepository             site.SiteRepository
	sitePlaylistsRepository    site.SitePlaylistsRepository
	videoRepository            video.VideoRepository
	playlistRepository         playlist.PlaylistRepository
	playlistI18nRepository     playlist.PlaylistI18nRepository
	playlistVidRepository      playlist.PlaylistVidRepository
	PlaylistSeoRepository      playlist.PlaylistSeoRepository
	siteSeoRepository          site.SiteSeoRepository
	sitePageTemplateRepository site.SitePageTemplateRepository
	contentUnlockRepository    payment.ContentUnlockRepository
	userRepository             user.UserRepository
	userPlayRecordsRepository  user.UserPlayRecordsRepository
}

func (s *clientPlayerService) SimpleFeed(ctx *gin.Context, siteID string, userID string, pageSize int) (*api.FeedResponseData, error) {

	site, err := s.siteRepository.GetBySiteID(ctx, siteID)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, common.ErrSiteNotFound
	}

	// Get playlists under the site, no pagination for now.
	playlists, err := s.sitePlaylistsRepository.GetAllPlaylistBySiteID(ctx, site.SiteID, model.PlaylistStatusPublished)
	if err != nil {
		return nil, err
	}
	if len(playlists) == 0 {
		return nil, nil
	}

	MAX_VIDEO_COUNT := 1000
	type Item struct {
		VID                      string
		Order                    int // Indicates the order of the video in the current playlist
		PlaylistID               string
		PlaylistAccessType       int
		PlaylistSingleVideoPrice int
		PlaylistFreeVideos       int
	}
	allvids := make([]*Item, MAX_VIDEO_COUNT)
	counter := 0
	for _, playlist := range playlists {
		vidData, _, err := s.GetVideosByPlaylistID(ctx, playlist.PlaylistID)
		if err != nil {
			continue
		}
		if vidData == nil {
			continue
		}
		order := 1
		for _, vid := range vidData.VIDs {
			if counter >= MAX_VIDEO_COUNT {
				break
			}
			allvids[counter] = &Item{
				VID:                      vid,
				Order:                    order,
				PlaylistID:               playlist.PlaylistID,
				PlaylistAccessType:       playlist.AccessType,
				PlaylistSingleVideoPrice: playlist.SingleVideoPrice,
				PlaylistFreeVideos:       playlist.FreeVideos,
			}
			counter++
			order++

		}
		if counter >= MAX_VIDEO_COUNT {
			break
		}
	}
	// Filter out nil entries and create actual slice of valid items
	validItems := []*Item{}
	for _, item := range allvids {
		if item == nil {
			continue
		}
		validItems = append(validItems, item)
	}

	// If no valid items, return empty result
	if len(validItems) == 0 {
		return &api.FeedResponseData{
			HasMore: false,
			Items:   []*api.FeedItem{},
		}, nil
	}

	rand.Seed(uint64(time.Now().UnixNano()))
	rand.Shuffle(len(validItems), func(i, j int) {
		validItems[i], validItems[j] = validItems[j], validItems[i]
	})
	// Limit to pageSize or available count
	count := pageSize
	if count > len(validItems) {
		count = len(validItems)
	}

	selectedItems := make([]*Item, count)
	idx := 0
	var user *model.User
	if userID != "" {
		user, err = s.userRepository.GetByUserID(ctx, userID)
		if err != nil {
			log.Warning(ctx, "get user by userID error: "+err.Error())
			return nil, err
		}
	}
	for _, item := range validItems {
		if item == nil {
			continue
		}
		if idx >= count {
			break
		}
		// If user is a premium member, add all items
		if user != nil && user.IsValidPremiumMember() {
			selectedItems[idx] = item
			idx++
			continue
		}
		if item.PlaylistAccessType == 1 || item.PlaylistAccessType == 0 { // Free 0 is for compatibility with old data
			selectedItems[idx] = item
			idx++
		} else {
			if item.Order <= item.PlaylistFreeVideos {
				selectedItems[idx] = item
				idx++
			}
			//TODO Add after user purchase

		}
	}

	// Get the selected items

	// Fetch video details for selected items
	items := make([]*api.FeedItem, 0, count)
	for _, item := range selectedItems {
		if item == nil {
			continue
		}
		items = append(items, &api.FeedItem{
			VID:        item.VID,
			PlaylistID: item.PlaylistID,
			Episode:    item.Order,
		})
	}

	return &api.FeedResponseData{
		HasMore: false,
		Items:   items,
	}, nil
}

func (s *clientPlayerService) GetPlaylistInfo(ctx *gin.Context, playlistID string, lang string, needVideo bool, userID string) (*api.PlaylistInfo, error) {
	log.AddNotice(ctx, "playlist_id", playlistID)

	// 先从 i18n 表查找对应多语言内容
	var playlistI18n *model.PlaylistI18n
	var err error
	if lang != "" {
		playlistI18n, err = s.playlistI18nRepository.Get(ctx, playlistID, lang)
		if err != nil {
			log.Warning(ctx, "get playlist i18n error: "+err.Error())
		}
	}

	// 查找 playlist 表，客户端固定使用 WithUtmSource 过滤
	var playlist *model.Playlist
	utmSource := ctx.GetHeader("Utm-Source0")
	if userID != "" {
		if user, err := s.userRepository.GetByUserID(ctx, userID); err == nil && user != nil {
			utmSource = user.Referer
		}
	}
	playlist, err = s.playlistRepository.GetByPlaylistIDWithUtmSource(ctx, playlistID, utmSource)

	if err != nil {
		return nil, err
	}
	if playlist == nil {
		return nil, nil
	}

	// 确定最终使用的 title, description, slug 和 SEO 内容
	var title, description, slug, seoTitle, seoDescription, seoKeywords string
	if playlistI18n != nil {
		// 使用 i18n 表的多语言内容
		title = playlistI18n.Title
		description = playlistI18n.Description
		slug = playlistI18n.Slug
		seoTitle = playlistI18n.SeoTitle
		seoDescription = playlistI18n.SeoDescription
		seoKeywords = playlistI18n.SeoKeywords
		log.AddNotice(ctx, "i18n_language", lang)
	} else {
		// 使用 playlist 表的原始内容
		title = playlist.Title
		description = playlist.Description
		slug = playlist.Slug
		// 如果 i18n 没有找到,尝试从 playlist_seo 表获取 SEO 信息
		seo, err := s.PlaylistSeoRepository.FindSeo(ctx, playlistID)
		if err != nil {
			return nil, err
		}
		if seo != nil {
			seoTitle = seo.Title
			seoDescription = seo.Description
			seoKeywords = seo.Keywords
		}
	}

	videoCount := 0
	orderData := &api.VideoSortData{}
	if playlist.OrderVids != "" {
		err = json.Unmarshal([]byte(playlist.OrderVids), orderData)
		if err != nil {
			log.Error(ctx, err.Error())
			return nil, err
		}
		videoCount = len(orderData.VIDs)
	}
	var videos []*api.VideoItem
	// File video data
	if needVideo {
		videos, err = s.getVideoStatusByPlaylist(ctx, playlist, orderData, userID)
		if err != nil {
			log.Error(ctx, err)
			return nil, err
		}
	}
	log.AddNotice(ctx, "title", title)
	log.AddNotice(ctx, "playlist_video_count", videoCount)

	// Handle FreeVideos pointer conversion
	freeVideos := 0
	if playlist.FreeVideos != nil {
		freeVideos = *playlist.FreeVideos
	}

	return &api.PlaylistInfo{
		PlaylistID:       playlist.PlaylistID,
		Title:            title,
		Slug:             slug,
		Description:      description,
		Cover:            playlist.Cover,
		AccessType:       playlist.AccessType,
		FreeVideos:       freeVideos,
		SingleVideoPrice: playlist.SingleVideoPrice,
		Seo: &api.PlaylistSeo{
			Title:       seoTitle,
			Description: seoDescription,
			Keywords:    seoKeywords,
		},
		VideoCount: videoCount,
		Videos:     videos,
	}, nil
}

// BatchGetPlaylistInfo batch get playlist details by IDs
func (s *clientPlayerService) BatchGetPlaylistInfo(ctx *gin.Context, playlistIDs []string, lang string, needVideo bool, userID string) ([]*api.PlaylistInfo, error) {
	if len(playlistIDs) == 0 {
		return []*api.PlaylistInfo{}, nil
	}

	log.AddNotice(ctx, "playlist_ids_len", len(playlistIDs))

	// 1. Batch get playlists from database
	var playlists []*model.Playlist
	utmSource := ctx.GetHeader("Utm-Source0")
	if userID != "" {
		if user, err := s.userRepository.GetByUserID(ctx, userID); err == nil && user != nil {
			utmSource = user.Referer
		}
	}
	var err error
	playlists, err = s.playlistRepository.GetByPlaylistIDSWithUtmSource(ctx, playlistIDs, utmSource)

	if err != nil {
		return nil, err
	}

	if len(playlists) == 0 {
		return []*api.PlaylistInfo{}, nil
	}

	playlistMap := make(map[string]*model.Playlist, len(playlists))
	for _, p := range playlists {
		playlistMap[p.PlaylistID] = p
	}

	return s.buildPlaylistInfoList(ctx, playlistIDs, playlistMap, lang, needVideo, false, userID)
}

func (s *clientPlayerService) buildPlaylistInfoList(ctx *gin.Context, playlistIDs []string, playlistMap map[string]*model.Playlist, lang string, needVideo bool, includeVideoConfig bool, userID string) ([]*api.PlaylistInfo, error) {
	orderDataMap := make(map[string]*api.VideoSortData, len(playlistIDs))
	for _, playlistID := range playlistIDs {
		playlist := playlistMap[playlistID]
		if playlist == nil || playlist.OrderVids == "" {
			continue
		}
		orderData := &api.VideoSortData{}
		if err := json.Unmarshal([]byte(playlist.OrderVids), orderData); err != nil {
			log.Error(ctx, "unmarshal order vids error for "+playlistID+": "+err.Error())
			continue
		}
		orderDataMap[playlistID] = orderData
	}

	videoMap := make(map[string]*model.Video)
	if needVideo || includeVideoConfig {
		allVids := make([]string, 0)
		for _, orderData := range orderDataMap {
			allVids = append(allVids, orderData.VIDs...)
		}
		if len(allVids) > 0 {
			vs, err := s.videoRepository.GetByVIDs(ctx, allVids)
			if err != nil {
				return nil, err
			}
			for _, v := range vs {
				videoMap[v.VID] = v
			}
		}
	}
	i18nMap := make(map[string]*model.PlaylistI18n)
	if lang != "" {
		for _, playlistID := range playlistIDs {
			if _, exists := playlistMap[playlistID]; exists {
				playlistI18n, err := s.playlistI18nRepository.Get(ctx, playlistID, lang)
				if err != nil {
					log.Warning(ctx, "get playlist i18n error for "+playlistID+": "+err.Error())
				}
				if playlistI18n != nil {
					i18nMap[playlistID] = playlistI18n
				}
			}
		}
	}

	seoMap := make(map[string]*model.PlaylistSeo)
	for _, playlistID := range playlistIDs {
		if _, exists := playlistMap[playlistID]; exists {
			if _, hasI18n := i18nMap[playlistID]; hasI18n {
				continue
			}

			seo, err := s.PlaylistSeoRepository.FindSeo(ctx, playlistID)
			if err != nil {
				log.Warning(ctx, "get playlist seo error for "+playlistID+": "+err.Error())
			}
			if seo != nil {
				seoMap[playlistID] = seo
			}
		}
	}

	result := make([]*api.PlaylistInfo, 0, len(playlistIDs))
	for _, playlistID := range playlistIDs {
		playlist, exists := playlistMap[playlistID]
		if !exists {
			continue
		}

		var title, description, slug, seoTitle, seoDescription, seoKeywords string
		if playlistI18n, hasI18n := i18nMap[playlistID]; hasI18n {
			title = playlistI18n.Title
			description = playlistI18n.Description
			slug = playlistI18n.Slug
			seoTitle = playlistI18n.SeoTitle
			seoDescription = playlistI18n.SeoDescription
			seoKeywords = playlistI18n.SeoKeywords
		} else {
			title = playlist.Title
			description = playlist.Description
			slug = playlist.Slug
			if seo, hasSeo := seoMap[playlistID]; hasSeo {
				seoTitle = seo.Title
				seoDescription = seo.Description
				seoKeywords = seo.Keywords
			}
		}

		videoCount := 0
		orderData := orderDataMap[playlistID]
		if orderData != nil {
			videoCount = len(orderData.VIDs)
		}

		var videos []*api.VideoItem
		if needVideo {
			var err error
			videos, err = s.getVideoStatusByPlaylist(ctx, playlist, orderData, userID)
			if err != nil {
				log.Error(ctx, "get videos error for "+playlistID+": "+err.Error())
			}
		} else if includeVideoConfig {
			videos = buildVideoItemsWithConfig(orderData, videoMap)
		}

		freeVideos := 0
		if playlist.FreeVideos != nil {
			freeVideos = *playlist.FreeVideos
		}

		result = append(result, &api.PlaylistInfo{
			PlaylistID:       playlist.PlaylistID,
			Title:            title,
			Slug:             slug,
			Description:      description,
			Tags:             playlist.Tags,
			Cover:            playlist.Cover,
			Status:           playlist.Status,
			Version:          playlist.Version,
			AccessType:       playlist.AccessType,
			FreeVideos:       freeVideos,
			SingleVideoPrice: playlist.SingleVideoPrice,
			UtmSource:        playlist.UtmSource,
			Seo: &api.PlaylistSeo{
				Title:       seoTitle,
				Description: seoDescription,
				Keywords:    seoKeywords,
			},
			VideoCount: videoCount,
			Videos:     videos,
			CreatedAt:  playlist.CreatedAt.Unix(),
			UpdatedAt:  playlist.UpdatedAt.Unix(),
		})
	}

	log.AddNotice(ctx, "result_count", len(result))
	return result, nil
}

func buildVideoItemsWithConfig(orderData *api.VideoSortData, videoMap map[string]*model.Video) []*api.VideoItem {
	if orderData == nil || len(orderData.VIDs) == 0 {
		return nil
	}

	result := make([]*api.VideoItem, 0, len(orderData.VIDs))
	for _, vid := range orderData.VIDs {
		item := &api.VideoItem{VID: vid}
		if video, ok := videoMap[vid]; ok {
			item.Status = int(video.Status)
			item.Config = video.Config
			item.Cover = video.Cover
			item.LocalPath = localPathFromCover(video.Cover)
		}
		result = append(result, item)
	}
	return result
}

// localPathFromCover derives playback URL by replacing the cover file extension with .mp4.
func localPathFromCover(cover *types.ImageURL) *types.ImageURL {
	if cover == nil || *cover == "" {
		return nil
	}
	path := string(*cover)
	if i := strings.LastIndex(path, "."); i >= 0 && !strings.Contains(path[i:], "/") {
		path = path[:i] + ".mp4"
	} else {
		path = path + ".mp4"
	}
	localPath := types.ImageURL(path)
	return &localPath
}

func (s *clientPlayerService) GetVideosByPlaylistID(ctx *gin.Context, playlistID string) (*api.VideoSortData, *model.Playlist, error) {

	var palylistInfo *model.Playlist
	var err error
	utmSource := ctx.GetHeader("Utm-Source0")
	userID := ctx.GetString("user_id")
	if userID != "" {
		if user, err := s.userRepository.GetByUserID(ctx, userID); err == nil && user != nil {
			utmSource = user.Referer
		}
	}
	palylistInfo, err = s.playlistRepository.GetByPlaylistIDWithUtmSource(ctx, playlistID, utmSource)

	if err != nil {
		return nil, nil, err
	}
	if palylistInfo == nil {
		return nil, nil, nil
	}
	if palylistInfo.OrderVids == "" {
		log.AddNotice(ctx, "vid_len", 0)
		log.Warning(ctx, "playlist by order no data, playlisid:"+playlistID)
		return nil, nil, nil
	}

	orderData := &api.VideoSortData{}
	err = json.Unmarshal([]byte(palylistInfo.OrderVids), orderData)
	if err != nil {
		log.Error(ctx, "unmarshal order vids error:"+err.Error())
		return nil, nil, err
	}
	total := len(orderData.VIDs)
	if total == 0 {
		log.AddNotice(ctx, "vid_len", 0)
		return nil, nil, nil
	}
	log.AddNotice(ctx, "vid_len", total)

	return orderData, palylistInfo, nil
}

func (s *clientPlayerService) AnonRegister(ctx *gin.Context, req *api.AnonRegisterRequest) (string, string, error) {

	// Anonymous registration compatible with non-empty path
	site := &model.Site{}
	if req != nil && req.SitePath != "" {
		var err error
		site, err = s.siteRepository.GetByPath(ctx, req.SitePath)
		if err != nil {
			return "", "", err
		}
		if site == nil {
			return "", "", common.ErrSiteNotFound
		}
	} else {
		var err error
		site, err = s.siteRepository.GetByDomain(ctx, req.Host)
		if err != nil {
			return "", "", err
		}
		if site == nil {
			return "", "", common.ErrSiteNotFound
		}
	}

	guestID := uuid.New().String()
	token, err := s.jwt.GenClientUserToken(guestID, time.Now().AddDate(100, 0, 0)) //匿名登录有效期100年
	if err != nil {
		log.Error(ctx, "gen token error:"+err.Error())
		return "", "", err
	}
	return token, site.SiteID, nil
}

func (s *clientPlayerService) GetSiteByPath(ctx *gin.Context, sitePath string) (*model.Site, error) {
	site, err := s.siteRepository.GetByPath(ctx, sitePath)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, common.ErrSiteNotFound
	}
	// TODO Need to be consistent with B-side in the future
	seo, err := s.siteSeoRepository.GetBySiteID(ctx, site.SiteID)
	if err != nil {
		return nil, err
	}
	if seo != nil {
		site.SeoDescription = seo.Description
		site.SeoKeywords = seo.Keywords
		site.SeoTitle = seo.Title
		site.SeoI18n = seo.I18n
	}

	// Get template name if template_id exists
	if site.TemplateID != nil {
		template, err := s.sitePageTemplateRepository.GetByTemplateID(ctx, *site.TemplateID)
		if err != nil {
			return nil, err
		}
		if template != nil {
			site.TemplateName = &template.Name
		}
	}

	log.AddNotice(ctx, "site_name", site.Name)
	return site, nil
}

// CreateOrUpdatePlayRecord creates or updates a user play record
func (s *clientPlayerService) CreateOrUpdatePlayRecord(ctx *gin.Context, record *model.UserPlayRecord) error {
	return s.userPlayRecordsRepository.CreateOrUpdate(ctx, record)
}

// GetUserPlayHistory retrieves user play history with pagination
func (s *clientPlayerService) GetUserPlayHistory(ctx *gin.Context, userID, siteID string, page, pageSize int) ([]*model.UserPlayRecord, int64, error) {
	return s.userPlayRecordsRepository.GetByUserIDAndSiteID(ctx, userID, siteID, page, pageSize)
}

func (s *clientPlayerService) GetVideoUnlockMap(ctx *gin.Context, playlistID string, userID string) (map[string]string, error) {

	unlocks, err := s.contentUnlockRepository.GetByPlaylistID(ctx, userID, playlistID)
	if err != nil {
		return nil, err
	}

	unlockedMap := make(map[string]string, len(unlocks))
	for _, unlock := range unlocks {
		// Both individual video unlocks and full playlist unlocks
		if unlock.ContentType == "video" {
			unlockedMap[unlock.ContentID] = unlock.PlaylistID
		} else {
			return nil, fmt.Errorf("playlist unlock not supported")
		}
	}

	return unlockedMap, nil
}

func (s *clientPlayerService) GetVideosAndStatusByPlaylistID(ctx *gin.Context, playlistID string, userID string) ([]*api.VideoItem, error) {
	vids, playlistinfo, err := s.GetVideosByPlaylistID(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if vids == nil {
		return nil, nil
	}
	if playlistinfo == nil {
		return nil, fmt.Errorf("playlist not found")
	}
	return s.getVideoStatusByPlaylist(ctx, playlistinfo, vids, userID)

}

func (s *clientPlayerService) getVideoStatusByPlaylist(ctx *gin.Context, playlistInfo *model.Playlist, vids *api.VideoSortData, userID string) ([]*api.VideoItem, error) {
	unlockMap, err := s.GetVideoUnlockMap(ctx, playlistInfo.PlaylistID, userID)
	if err != nil {
		return nil, err
	}
	result := make([]*api.VideoItem, 0, len(vids.VIDs))
	order := 1

	// Generate map status
	vs, err := s.videoRepository.GetByVIDs(ctx, vids.VIDs)
	if err != nil {
		return nil, err
	}
	if vs == nil {
		log.Warning(ctx, "video not found, playlistID: "+playlistInfo.PlaylistID)
		return nil, nil
	}
	videoStatusMap := make(map[string]int)
	videoConfigMap := make(map[string]json.RawMessage)
	for _, v := range vs {
		videoStatusMap[v.VID] = int(v.Status)
		if len(v.Config) > 0 {
			videoConfigMap[v.VID] = v.Config
		}
	}
	var user *model.User
	if userID != "" {
		user, err = s.userRepository.GetByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	for _, vid := range vids.VIDs {
		item := &api.VideoItem{
			VID:    vid,
			Status: videoStatusMap[vid],
			Config: videoConfigMap[vid],
		}

		// Unlock status
		if playlistInfo.AccessType == 1 || playlistInfo.AccessType == 0 { // Free 0 is for compatibility with old data
			item.UnLockStatus = 1
		} else {
			if user != nil && user.IsValidPremiumMember() {
				item.UnLockStatus = 1

			} else {
				if order > *playlistInfo.FreeVideos {
					item.UnLockStatus = 2 // 2 means unlock is required
					if userID != "" {
						if _, ok := unlockMap[vid]; ok {
							item.UnLockStatus = 1 // 1 means unlocked
						} else {
							item.UnLockStatus = 2 // 2 means unlock is required
						}
					}
				} else {
					item.UnLockStatus = 1
				}
			}

		}
		result = append(result, item)
		order++
	}
	return result, nil
}

func (s *clientPlayerService) GetSiteByHost(ctx *gin.Context, host string) (*model.Site, error) {
	// Strip http:// or https:// prefixes from host if present
	if strings.HasPrefix(host, "http://") {
		host = host[7:]
	} else if strings.HasPrefix(host, "https://") {
		host = host[8:]
	}
	// Also remove any trailing path or query parameters
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	site, err := s.siteRepository.GetByDomain(ctx, host)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, common.ErrSiteNotFound
	}
	// TODO Need to be consistent with B-side in the future
	seo, err := s.siteSeoRepository.GetBySiteID(ctx, site.SiteID)
	if err != nil {
		return nil, err
	}
	if seo != nil {
		site.SeoDescription = seo.Description
		site.SeoKeywords = seo.Keywords
		site.SeoTitle = seo.Title
		site.SeoI18n = seo.I18n
	}

	// Get template name if template_id exists
	if site.TemplateID != nil {
		template, err := s.sitePageTemplateRepository.GetByTemplateID(ctx, *site.TemplateID)
		if err != nil {
			return nil, err
		}
		if template != nil {
			site.TemplateName = &template.Name
		}
	}

	log.AddNotice(ctx, "site_name", site.Name)
	return site, nil
}

func (s *clientPlayerService) GetAllPlaylistIDs(ctx *gin.Context, siteID string, page, pageSize int) ([]*api.PlaylistSlugItem, int64, error) {
	// Validate site exists
	site, err := s.siteRepository.GetBySiteID(ctx, siteID)
	if err != nil {
		return nil, 0, err
	}
	if site == nil {
		return nil, 0, common.ErrSiteNotFound
	}

	// Get utm_source from header or user referer
	utmSource := ctx.GetHeader("Utm-Source0")
	userID := ctx.GetString("user_id")
	if userID != "" {
		if user, err := s.userRepository.GetByUserID(ctx, userID); err == nil && user != nil {
			utmSource = user.Referer
		}
	}

	// Get playlists with pagination and utm_source filtering from repository
	playlists, total, err := s.sitePlaylistsRepository.GetPlaylistByPageWithUtmSource(ctx, siteID, model.PlaylistStatusPublished, page, pageSize, utmSource)
	if err != nil {
		log.Error(ctx, "get playlists failed, error: "+err.Error())
		return nil, 0, err
	}

	// If no playlists found, return empty result
	if len(playlists) == 0 {
		return []*api.PlaylistSlugItem{}, total, nil
	}

	// Collect all playlist IDs for batch query
	playlistIDs := make([]string, 0, len(playlists))
	for _, playlist := range playlists {
		playlistIDs = append(playlistIDs, playlist.PlaylistID)
	}

	// Batch query all i18n data at once
	allI18nData, err := s.playlistI18nRepository.ListByPlaylistIDs(ctx, playlistIDs)
	if err != nil {
		log.Error(ctx, "batch query i18n failed, error: "+err.Error())
		// Continue without i18n data
		allI18nData = []*model.PlaylistI18n{}
	}

	// Build a map for quick lookup: playlistID -> []SlugI18n
	i18nMap := make(map[string][]api.SlugI18n)
	for _, i18n := range allI18nData {
		if i18n.Slug != "" {
			i18nMap[i18n.PlaylistID] = append(i18nMap[i18n.PlaylistID], api.SlugI18n{
				Lang: i18n.Language,
				Slug: i18n.Slug,
			})
		}
	}

	// Extract playlist data with i18n slugs
	items := make([]*api.PlaylistSlugItem, 0, len(playlists))
	for _, playlist := range playlists {
		var vids []string
		if playlist.OrderVids != "" {
			orderData := &api.VideoSortData{}
			err = json.Unmarshal([]byte(playlist.OrderVids), orderData)
			if err != nil {
				vids = []string{}
			} else {
				vids = orderData.VIDs
			}
		}

		// Get i18n slugs from map (O(1) lookup)
		slugs := i18nMap[playlist.PlaylistID]

		// Convert cover string to ImageURL type
		var cover *types.ImageURL
		if playlist.Cover != "" {
			imgURL := types.ImageURL(playlist.Cover)
			cover = &imgURL
		}

		items = append(items, &api.PlaylistSlugItem{
			PlaylistID: playlist.PlaylistID,
			Slug:       playlist.Slug,
			Title:      playlist.Title,
			Cover:      cover,
			Slugs:      slugs,
			Vids:       vids,
		})
	}

	log.AddNotice(ctx, "playlist_count", len(items))
	return items, total, nil
}

func (s *clientPlayerService) GetSitePlist(ctx *gin.Context, siteID string, page, pageSize, orderType int) ([]*api.PlaylistInfo, int64, error) {
	site, err := s.siteRepository.GetBySiteID(ctx, siteID)
	if err != nil {
		return nil, 0, err
	}
	if site == nil {
		return nil, 0, common.ErrSiteNotFound
	}

	status := int(model.PlaylistStatusPublished)
	query := &model.PlaylistQuery{
		SiteID:           siteID,
		Status:           &status,
		ExcludeUtmSource: "m1",
	}

	playlists, total, err := s.playlistRepository.ListByPage(ctx, query, page, pageSize, orderType)
	if err != nil {
		return nil, 0, err
	}
	if len(playlists) == 0 {
		return []*api.PlaylistInfo{}, total, nil
	}

	playlistIDs := make([]string, 0, len(playlists))
	playlistMap := make(map[string]*model.Playlist, len(playlists))
	for _, p := range playlists {
		playlistIDs = append(playlistIDs, p.PlaylistID)
		playlistMap[p.PlaylistID] = p
	}

	lang := ctx.GetString("lang")
	userID := ctx.GetString("user_id")
	items, err := s.buildPlaylistInfoList(ctx, playlistIDs, playlistMap, lang, false, true, userID)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// SimpleFeedV2 provides a simplified feed with more efficient data handling
func (s *clientPlayerService) SimpleFeedV2(ctx *gin.Context, siteID string, userID string, page, pageSize int) (*api.FeedResponseData, error) {
	// Validate site exists
	site, err := s.siteRepository.GetBySiteID(ctx, siteID)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, common.ErrSiteNotFound
	}

	// Get all playlists for the site
	playlists, total, err := s.sitePlaylistsRepository.GetPlaylistByPage(ctx, site.SiteID, model.PlaylistStatusPublished, page, pageSize)
	if err != nil {
		return nil, err
	}
	playlistLen := len(playlists)
	log.AddNotice(ctx, "playlist_count", playlistLen)
	if playlistLen == 0 {
		return &api.FeedResponseData{HasMore: false, Items: []*api.FeedItem{}}, nil
	}

	// The first episode of all playlist IDs
	videoItems := make([]*api.FeedItem, 0, pageSize)
	for _, playlist := range playlists {
		orderData, _, err := s.GetVideosByPlaylistID(ctx, playlist.PlaylistID)
		if err != nil || orderData == nil || len(orderData.VIDs) == 0 {
			continue
		}
		for _, vid := range orderData.VIDs {
			videoItems = append(videoItems, &api.FeedItem{
				VID:        vid,
				Episode:    1,
				PlaylistID: playlist.PlaylistID,
			})
			break // Only take the first video of the playlist
		}
	}

	return &api.FeedResponseData{
		Page:    page,
		HasMore: int64(page*pageSize) < total,
		Items:   videoItems,
	}, nil
}
