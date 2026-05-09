package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/playlist"
	"shortpress-server/internal/repository/db/video"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/log"
	"shortpress-server/pkg/translate"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

const SortNumberStep = 1000

type PlaylistService interface {
	GetPlaylist(ctx *gin.Context, id string) (*api.PlaylistInfo, error)
	VerifyPlaylistAccess(ctx *gin.Context, id string, utmSource string) (*model.Playlist, error)
	CreatePlaylist(ctx *gin.Context, creatorID string, req api.PlaylistInfo) (string, error)
	ModifyPlaylist(ctx *gin.Context, req api.PlaylistInfo) error
	DeletePlaylists(ctx *gin.Context, creatorID string, playlistIds []string) error
	AddVideos(ctx *gin.Context, playlistId string, videoIds []string) error
	RemoveVideos(ctx *gin.Context, playlistId string, videoIds []string) error
	List(ctx *gin.Context, query *model.PlaylistQuery, page, pageSize int, orderType int) ([]*model.Playlist, int64, error)
	Search(ctx *gin.Context, siteID string, keyword string, page int, pageSize int, orderType int, lang string) ([]string, int64, error)
	BatchGet(ctx *gin.Context, playlistIds []string) ([]*model.Playlist, error)
	UpdateVideosOrder(ctx *gin.Context, req api.PlaylistVideosOrder) error
	GetVideosOrder(ctx *gin.Context, playlistId string) (*api.PlaylistVideosOrder, error)
	AppendVidsToOrder(ctx *gin.Context, playlistID string, vids []string) error
	ChangeAccessType(ctx *gin.Context, creatorID string, req api.PlaylistAccessChangeRequest) error
	GetSimpleRelatedPlaylists(ctx *gin.Context, siteID string, playlistID string) ([]string, error)
	GetRecAfterPlaylists(ctx *gin.Context, siteID string, playlistID string) ([]string, error)
	GetNewReleasePlaylistsBySiteID(ctx *gin.Context, siteID string) ([]*api.NewReleasePlaylistItem, error)
	TranslatePlaylist(ctx *gin.Context, req api.PlaylistTranslateRequest) ([]*api.PlaylistI18nItem, error)
	GetPlaylistI18n(ctx *gin.Context, id string) ([]*api.PlaylistI18nItem, error)
	BatchModifyPlaylistI18n(ctx *gin.Context, req []*api.PlaylistI18nItem) error
}

func NewPlaylistService(
	service *Service,
	playlistRepository playlist.PlaylistRepository,
	playlistSeoRepository playlist.PlaylistSeoRepository,
	playlistVidRepository playlist.PlaylistVidRepository,
	playlistI18nRepository playlist.PlaylistI18nRepository,
	videoRepository video.VideoRepository,
	conf *viper.Viper,
) PlaylistService {
	llm := types.GetLLMConfig()
	targetLang := types.GetTranslatorLang()
	return &playlistService{
		Service:                service,
		playlistRepository:     playlistRepository,
		playlistSeoRepository:  playlistSeoRepository,
		playlistVidRepository:  playlistVidRepository,
		playlistI18nRepository: playlistI18nRepository,
		videoRepository:        videoRepository,
		conf:                   conf,
		translator:             translate.NewTranslator(llm.BaseURL, llm.Model, llm.APIKey, targetLang),
	}
}

type playlistService struct {
	*Service
	playlistRepository     playlist.PlaylistRepository
	playlistSeoRepository  playlist.PlaylistSeoRepository
	playlistVidRepository  playlist.PlaylistVidRepository
	playlistI18nRepository playlist.PlaylistI18nRepository
	videoRepository        video.VideoRepository
	conf                   *viper.Viper
	translator             translate.Translator
}

func (s *playlistService) VerifyPlaylistAccess(ctx *gin.Context, id string, utmSource string) (*model.Playlist, error) {
	var playlist *model.Playlist
	var err error

	// 强制使用带 UTM 的校验方法（针对客户端访问）
	playlist, err = s.playlistRepository.GetByPlaylistIDWithUtmSource(ctx, id, utmSource)

	if err != nil {
		return nil, err
	}
	if playlist == nil {
		return nil, common.ErrPlaylistNotFound
	}

	return playlist, nil
}

func (s *playlistService) GetPlaylist(ctx *gin.Context, id string) (*api.PlaylistInfo, error) {
	// 从 header 中获取 UTM 来源
	utmSource := ctx.GetHeader("Utm-Source0")

	// 1. Get basic playlist information with UTM source filter
	var playlist *model.Playlist
	var err error

	if utmSource != "" {
		playlist, err = s.playlistRepository.GetByPlaylistIDWithUtmSource(ctx, id, utmSource)
	} else {
		playlist, err = s.playlistRepository.GetByPlaylistID(ctx, id)
	}

	if err != nil {
		return nil, err
	}
	if playlist == nil {
		return nil, common.ErrPlaylistNotFound
	}
	videoCount := 0
	if playlist.OrderVids != "" {
		orderData := &api.VideoSortData{}
		err = json.Unmarshal([]byte(playlist.OrderVids), orderData)
		if err != nil {
			log.Error(ctx, "unmarshal order vids error, "+err.Error())
			return nil, err
		}
		videoCount = len(orderData.VIDs)
	}

	// 2. Construct response data
	fv := 0
	if playlist.FreeVideos != nil {
		fv = *playlist.FreeVideos
	}
	response := &api.PlaylistInfo{
		PlaylistID:       playlist.PlaylistID,
		Title:            playlist.Title,
		Slug:             playlist.Slug,
		Description:      playlist.Description,
		Tags:             playlist.Tags,
		Cover:            playlist.Cover,
		Status:           playlist.Status,
		VideoCount:       videoCount,
		AccessType:       playlist.AccessType,
		SingleVideoPrice: playlist.SingleVideoPrice,
		FreeVideos:       fv,
		Version:          playlist.Version,
		UtmSource:        playlist.UtmSource, // UTM 来源
		CreatedAt:        playlist.CreatedAt.Unix(),
		UpdatedAt:        playlist.UpdatedAt.Unix(),
	}

	// 3. Get and add SEO information
	seo, err := s.playlistSeoRepository.GetSeo(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if seo != nil {
		response.Seo = &api.PlaylistSeo{
			Title:       seo.Title,
			Description: seo.Description,
			Keywords:    seo.Keywords,
		}
	}
	log.AddNotice(ctx, "title", playlist.Title)
	log.AddNotice(ctx, "video_cunt", videoCount)
	return response, nil
}

// formatSlug converts a title to a valid slug:
// 1. Convert to lowercase
// 2. Replace spaces with hyphens
// 3. Remove special characters (except spaces and hyphens)
// 4. Support multilingual characters
func formatSlug(title string) string {
	// Convert to lowercase
	title = strings.ToLower(title)

	var result strings.Builder
	var lastCharWasHyphen bool

	for _, char := range title {
		if char == ' ' {
			// Convert space to hyphen, but avoid consecutive hyphens
			if !lastCharWasHyphen {
				result.WriteRune('-')
				lastCharWasHyphen = true
			}
		} else if char == '-' {
			// Keep existing hyphens, but avoid consecutive hyphens
			if !lastCharWasHyphen {
				result.WriteRune('-')
				lastCharWasHyphen = true
			}
		} else if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char > 127 {
			// Keep lowercase letters, numbers, and non-ASCII characters (for multilingual support)
			result.WriteRune(char)
			lastCharWasHyphen = false
		}
		// Remove special characters (punctuation, symbols, etc.)
	}

	return result.String()
}

func (s *playlistService) CreatePlaylist(ctx *gin.Context, creatorID string, req api.PlaylistInfo) (string, error) {
	// Generate playlist ID
	playlistID := uuid.NewString()

	// Construct playlist data
	if req.Status == 0 {
		req.Status = int(model.PlaylistStatusPublished)
	}
	playlist := &model.Playlist{
		PlaylistID:  playlistID,
		CreatorID:   creatorID,
		Title:       req.Title,
		Slug:        formatSlug(req.Title),
		Description: req.Description,
		Tags:        req.Tags,
		Cover:       req.Cover,
		Status:      req.Status,
		Version:     1,             // Version starts from 1
		UtmSource:   req.UtmSource, // UTM 来源
	}

	// Create playlist and SEO records in transaction
	err := s.tx.Transaction(ctx, func(ctx context.Context) error {
		// 1. Create playlist
		if err := s.playlistRepository.Create(ctx, playlist); err != nil {
			return err
		}

		seo := &model.PlaylistSeo{
			PlaylistID: playlistID,
		}

		// 设置标题
		if req.Seo != nil && req.Seo.Title != "" {
			seo.Title = req.Seo.Title
		}

		// 设置关键词
		if req.Seo != nil && req.Seo.Keywords != "" {
			seo.Keywords = req.Seo.Keywords
		}

		// 设置描述
		if req.Seo != nil && req.Seo.Description != "" {
			seo.Description = req.Seo.Description
		}

		if err := s.playlistSeoRepository.Create(ctx, seo); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return "", err
	}
	log.AddNotice(ctx, "playlist_id", playlistID)
	log.AddNotice(ctx, "playlist_title", req.Title)
	return playlistID, nil
}

func (s *playlistService) ModifyPlaylist(ctx *gin.Context, req api.PlaylistInfo) error {
	playlist := &model.Playlist{
		PlaylistID:  req.PlaylistID,
		Title:       req.Title,
		Slug:        formatSlug(req.Title),
		Description: req.Description,
		Tags:        req.Tags,
		Cover:       req.Cover,
		Status:      req.Status,
		UtmSource:   req.UtmSource, // UTM 来源
	}
	// Not using transaction processing for now
	if req.Seo != nil {
		playlistSeo := &model.PlaylistSeo{
			PlaylistID:  req.PlaylistID,
			Title:       req.Seo.Title,
			Description: req.Seo.Description,
			Keywords:    req.Seo.Keywords,
		}
		if err := s.playlistSeoRepository.Save(ctx, playlistSeo); err != nil {
			return err
		}
	}
	return s.playlistRepository.Update(ctx, playlist)
}

func (s *playlistService) DeletePlaylists(ctx *gin.Context, creatorID string, playlistIds []string) error {
	log.AddNotice(ctx, "playlist_ids", strings.Join(playlistIds, ","))
	if err := s.playlistRepository.BatchDelete(ctx, creatorID, playlistIds); err != nil {
		return err
	}
	return nil
}

func (s *playlistService) AddVideos(ctx *gin.Context, playlistID string, videoIDs []string) error {
	// 1. Check if playlist exists
	log.AddNotice(ctx, "playlist_id", playlistID)
	log.AddNotice(ctx, "video_ids_len", len(videoIDs))
	playlist, err := s.playlistRepository.GetByPlaylistID(ctx, playlistID)
	if err != nil {
		return err
	}
	if playlist == nil {
		return common.ErrPlaylistNotFound
	}

	err = s.tx.Transaction(ctx, func(ctx context.Context) error {
		//     // Construct records to add
		var records []*model.PlaylistVid
		for _, vid := range videoIDs {
			records = append(records, &model.PlaylistVid{
				PlaylistID: playlistID,
				VID:        vid,
			})
		}
		// Batch create records
		err = s.playlistVidRepository.BatchAdd(ctx, records)
		if err != nil {
			return err
		}

		count, err := s.playlistVidRepository.Count(ctx, playlistID)
		if err != nil {
			return err
		}

		// Update playlist video count
		upData := &model.Playlist{
			PlaylistID: playlistID,
			VideoCount: int(count),
		}
		err = s.playlistRepository.Update(ctx, upData)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (s *playlistService) RemoveVideos(ctx *gin.Context, playlistID string, videoIDs []string) error {
	// 1. Check if playlist exists
	log.AddNotice(ctx, "playlist_id", playlistID)
	log.AddNotice(ctx, "video_ids_len", len(videoIDs))
	playlist, err := s.playlistRepository.GetByPlaylistID(ctx, playlistID)
	if err != nil {
		return err
	}
	if playlist == nil {
		return common.ErrPlaylistNotFound
	}

	// 2. Delete video association records in transaction
	err = s.tx.Transaction(ctx, func(ctx context.Context) error {
		err := s.playlistVidRepository.BatchDelete(ctx, playlistID, videoIDs)
		if err != nil {
			return err
		}
		count, err := s.playlistVidRepository.Count(ctx, playlistID)
		if err != nil {
			return err
		}

		// Update playlist video count
		upData := &model.Playlist{
			PlaylistID: playlistID,
			VideoCount: int(count),
		}
		err = s.playlistRepository.Update(ctx, upData)
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

func (s *playlistService) List(ctx *gin.Context, query *model.PlaylistQuery, page int, pageSize int, orderType int) ([]*model.Playlist, int64, error) {
	// 从 header 中获取 UTM 来源
	utmSource := ctx.GetHeader("Utm-Source0")
	if utmSource != "" {
		// 如果 query 为空，创建一个新的
		if query == nil {
			query = &model.PlaylistQuery{}
		}
		query.UtmSource = utmSource
	}
	return s.playlistRepository.ListByPage(ctx, query, page, pageSize, orderType)
}

func (s *playlistService) Search(ctx *gin.Context, siteID string, keyword string, page int, pageSize int, orderType int, lang string) ([]string, int64, error) {
	status := int(model.PlaylistStatusPublished)

	fmt.Println(lang)

	if lang == "" || keyword == "" {
		// 从 header 中获取 UTM 来源
		utmSource := ctx.GetHeader("Utm-Source0")

		queryParams := &model.PlaylistQuery{
			SiteID:    siteID,
			Status:    &status,
			Keyword:   keyword,
			UtmSource: utmSource,
			FilterUtm: true, // 强制给客户端加过滤限制
		}

		playlists, total, err := s.playlistRepository.ListByPage(ctx, queryParams, page, pageSize, orderType)
		if err != nil {
			return nil, 0, err
		}

		playlistIDs := make([]string, 0, len(playlists))
		for _, p := range playlists {
			playlistIDs = append(playlistIDs, p.PlaylistID)
		}

		return playlistIDs, total, nil
	}

	// 如果提供了语言参数和关键词，使用支持 i18n 的搜索
	// 注意：SearchPlaylistIDsWithI18n 暂不支持 UTM 过滤，如果需要可以后续添加
	// Repository 层只负责数据查询，不包含业务逻辑
	return s.playlistRepository.SearchPlaylistIDsWithI18n(
		ctx,
		siteID,
		keyword,
		lang,
		&status,
		page,
		pageSize,
		orderType,
	)
}

func (s *playlistService) BatchGet(ctx *gin.Context, playlistIds []string) ([]*model.Playlist, error) {
	// 从 header 中获取 UTM 来源
	utmSource := ctx.GetHeader("Utm-Source0")

	var result []*model.Playlist
	var err error

	if utmSource != "" {
		result, err = s.playlistRepository.GetByPlaylistIDSWithUtmSource(ctx, playlistIds, utmSource)
	} else {
		result, err = s.playlistRepository.GetByPlaylistIDS(ctx, playlistIds)
	}

	if err != nil {
		return nil, err
	}
	// Ensure returned playlists are in the same order as playlistIds
	playlistMap := make(map[string]*model.Playlist)
	for _, p := range result {
		playlistMap[p.PlaylistID] = p
	}
	orderedResult := make([]*model.Playlist, 0, len(playlistIds))
	for _, id := range playlistIds {
		if p, exists := playlistMap[id]; exists {
			orderedResult = append(orderedResult, p)
		}
	}
	return orderedResult, nil

}

func (s *playlistService) UpdateVideosOrder(ctx *gin.Context, req api.PlaylistVideosOrder) error {
	// Marshal the sort data to JSON
	sortData, err := json.Marshal(req.SortData)
	if err != nil {
		return err
	}

	// Update the playlist with the new order
	orderPlaylist := &model.Playlist{
		PlaylistID: req.PlaylistID,
		OrderVids:  string(sortData),
		// Use version from retrieved playlist for optimistic concurrency control
		Version: req.Version,
	}

	rowAffected, err := s.playlistRepository.UpdateWithVersion(ctx, orderPlaylist)
	if err != nil {
		return err
	}
	if rowAffected == 0 {
		log.Warning(ctx, "playlist updated by other request, playlistID: "+req.PlaylistID)
		return common.ErrPlaylistUpdated
	}
	return nil
}

func (s *playlistService) GetVideosOrder(ctx *gin.Context, playlistId string) (*api.PlaylistVideosOrder, error) {
	playlist, err := s.playlistRepository.GetByPlaylistID(ctx, playlistId)
	if err != nil {
		return nil, err
	}
	if playlist == nil {
		return nil, common.ErrPlaylistNotFound
	}
	if playlist.OrderVids == "" {
		return &api.PlaylistVideosOrder{
			PlaylistID: playlistId,
			Version:    playlist.Version,
			SortData: &api.VideoSortData{
				VIDs: []string{},
			},
		}, nil
	}
	// Parse JSON string
	orderData := &api.VideoSortData{}
	err = json.Unmarshal([]byte(playlist.OrderVids), orderData)
	if err != nil {
		return nil, err
	}
	return &api.PlaylistVideosOrder{
		PlaylistID: playlistId,
		Version:    playlist.Version,
		SortData:   orderData,
	}, nil
}

func (s *playlistService) AppendVidsToOrder(ctx *gin.Context, playlistID string, vids []string) error {

	err := s.tx.Transaction(ctx, func(ctx context.Context) error {
		playlist, err := s.playlistRepository.GetByPlaylistIDForUpdate(ctx, playlistID)
		if err != nil {
			return err
		}
		if playlist == nil {
			return common.ErrPlaylistNotFound
		}
		orderData := &api.VideoSortData{}
		if playlist.OrderVids == "" {
			orderData.VIDs = []string{}
		} else {
			err = json.Unmarshal([]byte(playlist.OrderVids), orderData)
			if err != nil {
				return err
			}
		}
		orderData.VIDs = append(orderData.VIDs, vids...)
		newSortData, err := json.Marshal(orderData)
		if err != nil {
			return err
		}
		row, err := s.playlistRepository.UpdateWithVersion(ctx, &model.Playlist{
			PlaylistID: playlistID,
			OrderVids:  string(newSortData),
			Version:    playlist.Version,
		})
		if err != nil {
			return err
		}
		if row == 0 {
			return common.ErrPlaylistUpdated
		}

		return nil
	})

	return err
}

// ChangeAccessType updates the access settings for a playlist
func (s *playlistService) ChangeAccessType(ctx *gin.Context, creatorID string, req api.PlaylistAccessChangeRequest) error {
	// Get the playlist
	log.AddNotice(ctx, "playlist_id", req.PlaylistID)
	log.AddNotice(ctx, "access_type", req.AccessType)
	playlist, err := s.playlistRepository.GetByPlaylistID(ctx, req.PlaylistID)
	if err != nil {
		return err
	}

	if playlist == nil {
		return common.ErrPlaylistNotFound
	}

	// Verify creator has access to this playlist
	if playlist.CreatorID != creatorID {
		log.Warning(ctx, "creatorID not match, creatorID: "+creatorID+", playlistID: "+req.PlaylistID)
		return common.ErrUnauthorized
	}

	// Validate the request
	if req.AccessType < 1 || req.AccessType > 3 {
		return common.ErrBadRequest
	}

	// If it's a paid playlist, ensure pricing is set properly
	if req.AccessType == 2 {
		// Paid access type must have at least one pricing option
		if req.SingleVideoPrice <= 0 {
			return errors.New("paid playlists must have either coinPrice or singleVideoPrice")
		}
	} else {
		// For free or member-only playlists, reset prices to zero
		if req.AccessType == 1 || req.AccessType == 3 {
			req.SingleVideoPrice = 0
		}
	}

	// Update the playlist access settings
	playlist.AccessType = req.AccessType
	playlist.SingleVideoPrice = req.SingleVideoPrice
	if req.FreeVideos != nil {
		playlist.FreeVideos = req.FreeVideos
	}

	// Save the updated playlist
	return s.playlistRepository.Update(ctx, playlist)
}

// GetRecAfterPlaylists retrieves related playlists based on tags
func (s *playlistService) GetRecAfterPlaylists(ctx *gin.Context, siteID string, playlistID string) ([]string, error) {

	recPlaylists, err := s.GetSimpleRelatedPlaylists(ctx, siteID, playlistID)
	if err != nil {
		return nil, err
	}
	if len(recPlaylists) != 0 {
		return recPlaylists, nil // No related playlists found
	}
	// If no related playlists found, return new release playlists
	log.AddNotice(ctx, "use_new_release_playlists", "true")
	publishedStatus := int(model.PlaylistStatusPublished)
	query := &model.PlaylistQuery{
		SiteID: siteID,
		Status: &publishedStatus, // Only published playlists
	}

	playlists, _, err := s.playlistRepository.ListByPage(ctx, query, 1, 10, model.PlaylistSortByCreatedAtDesc)
	if err != nil {
		return []string{}, err
	}
	if len(playlists) == 0 {
		log.Warning(ctx, "no new release playlists found , siteID: "+siteID)
		return []string{}, fmt.Errorf("no new release playlists found for site %s", siteID)
	}
	for _, playlist := range playlists {
		if playlist.PlaylistID == playlistID {
			continue // Skip the original playlist
		}
		recPlaylists = append(recPlaylists, playlist.PlaylistID)
	}
	return recPlaylists, nil
}

func (s *playlistService) GetSimpleRelatedPlaylists(ctx *gin.Context, siteID string, playlistID string) ([]string, error) {
	// Get the playlist
	playlist, err := s.playlistRepository.GetBySiteAndPlaylistID(ctx, siteID, playlistID)
	if err != nil {
		return nil, err
	}
	if playlist == nil {
		return nil, common.ErrPlaylistNotFound
	}
	if playlist.Tags == "" {
		log.Warning(ctx, "playlist has no tags, playlistID: "+playlistID)
		return []string{}, nil // No tags, no related playlists
	}

	// 1. Split tags and trim, take at most 5 tags
	allTags := strings.Split(playlist.Tags, ",")
	var tags []string
	for i, tag := range allTags {
		if i >= 5 { // Take at most 5 tags
			break
		}
		trimmedTag := strings.TrimSpace(tag)
		if trimmedTag != "" {
			tags = append(tags, trimmedTag)
		}
	}

	if len(tags) == 0 {
		log.Warning(ctx, "playlist has no valid tags after trimming, playlistID: "+playlistID)
		return []string{}, nil
	}

	// 2. Map to store playlist and its matching tag count
	playlistTagCount := make(map[string]int)
	playlistDetails := make(map[string]*model.Playlist)

	// Search for playlists by each tag
	for _, tag := range tags {
		relatedPlaylists, err := s.playlistRepository.GetBySiteAndTag(ctx, siteID, tag)
		if err != nil {
			return nil, err
		}

		for _, relatedPlaylist := range relatedPlaylists {
			if relatedPlaylist.PlaylistID == playlistID {
				continue // Skip the original playlist
			}

			playlistTagCount[relatedPlaylist.PlaylistID]++
			playlistDetails[relatedPlaylist.PlaylistID] = relatedPlaylist
		}
	}

	// 3. Convert map to slice for sorting
	type PlaylistScore struct {
		PlaylistID string
		TagCount   int
		Playlist   *model.Playlist
	}

	var scoredPlaylists []PlaylistScore
	for playlistID, tagCount := range playlistTagCount {
		scoredPlaylists = append(scoredPlaylists, PlaylistScore{
			PlaylistID: playlistID,
			TagCount:   tagCount,
			Playlist:   playlistDetails[playlistID],
		})
	}

	// 4. Sort by tag count (descending), then by creation time (descending for tie-breaking)
	sort.Slice(scoredPlaylists, func(i, j int) bool {
		if scoredPlaylists[i].TagCount != scoredPlaylists[j].TagCount {
			return scoredPlaylists[i].TagCount > scoredPlaylists[j].TagCount
		}
		// If tag count is the same, sort by creation time (newer first)
		return scoredPlaylists[i].Playlist.CreatedAt.After(scoredPlaylists[j].Playlist.CreatedAt)
	})

	// 5. Extract playlist IDs, limit to 10
	var retPlaylists []string
	maxResults := 10
	for i, scored := range scoredPlaylists {
		if i >= maxResults {
			break
		}
		retPlaylists = append(retPlaylists, scored.PlaylistID)
	}

	return retPlaylists, nil
}

// GetNewReleasePlaylistsBySiteID gets the latest published playlists for a site
func (s *playlistService) GetNewReleasePlaylistsBySiteID(ctx *gin.Context, siteID string) ([]*api.NewReleasePlaylistItem, error) {
	// Query for published playlists in this site, order by created_at DESC, limit 10
	publishedStatus := int(model.PlaylistStatusPublished)
	query := &model.PlaylistQuery{
		SiteID: siteID,
		Status: &publishedStatus, // Only published playlists
	}

	playlists, _, err := s.playlistRepository.ListByPage(ctx, query, 1, 10, model.PlaylistSortByCreatedAtDesc)
	if err != nil {
		return nil, err
	}

	if len(playlists) == 0 {
		log.Warning(ctx, "no new release playlists found , siteID: "+siteID)
		return []*api.NewReleasePlaylistItem{}, nil
	}

	// Convert to response format
	items := make([]*api.NewReleasePlaylistItem, 0, len(playlists))
	for _, playlist := range playlists {
		items = append(items, &api.NewReleasePlaylistItem{
			PlaylistID: playlist.PlaylistID,
			Title:      playlist.Title,
			Slug:       playlist.Slug,
			Cover:      playlist.Cover,
			CreatedAt:  playlist.CreatedAt.Unix(),
		})
	}
	log.AddNotice(ctx, "playlist_count", len(items))

	return items, nil
}

func (s *playlistService) TranslatePlaylist(ctx *gin.Context, req api.PlaylistTranslateRequest) ([]*api.PlaylistI18nItem, error) {
	playlist, err := s.playlistRepository.GetByPlaylistID(ctx, req.PlaylistID)
	if err != nil {
		return nil, err
	}
	seo, err := s.playlistSeoRepository.GetSeo(ctx, req.PlaylistID)
	if err != nil {
		return nil, err
	}
	translateReq := translate.PlaylistTranslateReq{
		PlaylistI18nItem: translate.PlaylistI18nItem{
			Title:          playlist.Title,
			Description:    playlist.Description,
			Tags:           playlist.Tags,
			SeoTitle:       seo.Title,
			SeoDescription: seo.Description,
			SeoKeywords:    seo.Keywords,
		},
	}

	translateResp, err := s.translator.TranslatePlaylist(translateReq)
	if err != nil {
		log.Error(ctx, err)
		if strings.Contains(err.Error(), "handle translate error") {
			return nil, common.ErrHandleTranslateFailed
		}
		return nil, common.ErrValidateTranslateResult
	}
	var playlistI18nRecords []*model.PlaylistI18n
	var result []*api.PlaylistI18nItem
	for _, lang := range translateResp {
		// TODO: 后续需要优化prompt，防止ai擅自补充空字段
		seoDescription := s.translateField(lang.SeoDescription, translateReq.SeoDescription)
		seoKeywords := s.translateField(lang.SeoKeywords, translateReq.SeoKeywords)
		seoTitle := s.translateField(lang.SeoTitle, translateReq.SeoTitle)
		tags := s.translateField(lang.Tags, translateReq.Tags)

		playlistI18nRecords = append(playlistI18nRecords, &model.PlaylistI18n{
			PlaylistID:     req.PlaylistID,
			Language:       lang.Language,
			Title:          lang.Title,
			Slug:           formatSlug(lang.Title),
			Description:    lang.Description,
			Tags:           tags,
			SeoTitle:       seoTitle,
			SeoDescription: seoDescription,
			SeoKeywords:    seoKeywords,
		})
		// Build response result
		result = append(result, &api.PlaylistI18nItem{
			PlaylistID:     req.PlaylistID,
			Language:       lang.Language,
			Title:          lang.Title,
			Description:    lang.Description,
			Tags:           lang.Tags,
			SeoTitle:       lang.SeoTitle,
			SeoDescription: lang.SeoDescription,
			SeoKeywords:    lang.SeoKeywords,
		})
	}
	if err = s.playlistI18nRepository.BatchSaveOrUpdate(ctx, playlistI18nRecords); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *playlistService) translateField(value, original string) string {
	if original == "" {
		return ""
	}
	return value
}

func (s *playlistService) GetPlaylistI18n(ctx *gin.Context, id string) ([]*api.PlaylistI18nItem, error) {
	playlistI18nRecords, err := s.playlistI18nRepository.List(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(playlistI18nRecords) == 0 {
		log.Warning(ctx, "no i18n playlists found , playlistID: "+id)
		return []*api.PlaylistI18nItem{}, nil
	}
	var result []*api.PlaylistI18nItem
	for _, record := range playlistI18nRecords {
		result = append(result, &api.PlaylistI18nItem{
			PlaylistID:     record.PlaylistID,
			Language:       record.Language,
			Title:          record.Title,
			Description:    record.Description,
			Tags:           record.Tags,
			SeoTitle:       record.SeoTitle,
			SeoDescription: record.SeoDescription,
			SeoKeywords:    record.SeoKeywords,
		})
	}
	return result, nil
}

// BatchModifyPlaylistI18n batch modifies playlist i18n information
func (s *playlistService) BatchModifyPlaylistI18n(ctx *gin.Context, req []*api.PlaylistI18nItem) error {
	if len(req) == 0 {
		return nil
	}

	// Convert API requests to model objects
	var playlistI18nRecords []*model.PlaylistI18n
	for _, item := range req {
		if item.PlaylistID == "" {
			log.Warning(ctx, "playlist_id is empty, skipping")
			continue
		}
		if item.Language == "" {
			log.Warning(ctx, "language is empty for playlist "+item.PlaylistID+", skipping")
			continue
		}

		playlistI18nRecords = append(playlistI18nRecords, &model.PlaylistI18n{
			PlaylistID:     item.PlaylistID,
			Language:       item.Language,
			Title:          item.Title,
			Slug:           formatSlug(item.Title),
			Description:    item.Description,
			Tags:           item.Tags,
			SeoTitle:       item.SeoTitle,
			SeoDescription: item.SeoDescription,
			SeoKeywords:    item.SeoKeywords,
		})
	}

	// Batch save or update using existing repository method
	if err := s.playlistI18nRepository.BatchSaveOrUpdate(ctx, playlistI18nRecords); err != nil {
		log.Error(ctx, "batch save/update playlist i18n failed: "+err.Error())
		return err
	}

	log.AddNotice(ctx, "batch_modify_i18n_success", len(playlistI18nRecords))
	return nil
}
