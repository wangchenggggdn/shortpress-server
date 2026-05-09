package handler

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/middleware"
	"shortpress-server/internal/model"
	"shortpress-server/internal/service"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/log"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

var (
	transferCache sync.Map
)

type transferData struct {
	data      string
	expiredAt time.Time
}

func init() {
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			now := time.Now()
			transferCache.Range(func(key, value interface{}) bool {
				if td, ok := value.(transferData); ok {
					if now.After(td.expiredAt) {
						transferCache.Delete(key)
					}
				}
				return true
			})
		}
	}()
}

type ClientPlayerHandler struct {
	*Handler
	clientPlayerService service.ClientPlayerService
	playlistService     service.PlaylistService
	pagesBuilderService service.PagesBuilderService
	conf                *viper.Viper
}

func NewClientPlayerHandler(
	handler *Handler,
	clientPlayerService service.ClientPlayerService,
	playlistService service.PlaylistService,
	pagesBuilderService service.PagesBuilderService,
	conf *viper.Viper,
) *ClientPlayerHandler {
	return &ClientPlayerHandler{
		Handler:             handler,
		clientPlayerService: clientPlayerService,
		playlistService:     playlistService,
		pagesBuilderService: pagesBuilderService,
		conf:                conf,
	}
}

// Feed godoc
// @Summary User Feed Video Data
// @Description Paginate query user feed video data, supports page number and page size parameters
// @Tags client-player
// @Accept json
// @Produce json
// @Param guestId query string true "Guest ID"  example(xxxx-xxxxx-xxxxx)
// @Param playlistId query string false "Playlist ID, if not provided, random or other strategies will be used"  example(xxxx-xxxxx-xxxxx)
// @Param sitePath query string true "Site Path"  example(videopath)
// @Param page query int false "Page number, default is 1" minimum(1) example(1)
// @Param pageSize query int false "Page size, default is 10" minimum(1) maximum(50) example(10)
// @Success 200 {object} api.IDListResponseData[string] "Return Feed Video Data"
// @Router /api/client/feed [get]
func (h *ClientPlayerHandler) Feed(ctx *gin.Context) {
	// Parse pagination parameters
	page := 1
	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	pageSize := 10
	if sizeStr := ctx.Query("pageSize"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 50 {
			pageSize = s
		}
	}

	// guestId := ctx.Query("guestId")
	userID := ctx.GetString("user_id")
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrSiteNotFound, nil)
		return
	}

	// Call Service layer to get feed data
	feedData, err := h.clientPlayerService.SimpleFeedV2(ctx, siteID, userID, page, pageSize)
	if err != nil {
		log.Error(ctx, "get feed data failed, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}
	type Item struct {
		VID        string `json:"videoId"`
		PlaylistID string `json:"playlistId"`
		// Order int `json:"order"`
		Episode int `json:"episode"`
	}

	if feedData == nil {
		log.Warning(ctx, "feed data is empty")
		api.HandleSuccess(ctx, &api.IDListResponseData[Item]{
			Total:   0,
			HasMore: false,
			Items:   []Item{},
		})
		return
	}

	items := make([]Item, 0, len(feedData.Items))
	for _, item := range feedData.Items {
		items = append(items, Item{
			VID:        item.VID,
			PlaylistID: item.PlaylistID,
			Episode:    item.Episode,
		})
	}
	log.AddNotice(ctx, "feed_data_count", len(items))
	api.HandleSuccess(ctx, &api.IDListResponseData[Item]{
		Page:    feedData.Page,
		HasMore: feedData.HasMore,
		Items:   items,
	})
}

// PlaylistInfo godoc
// @Summary Get Playlist Details
// @Description Get Playlist Details
// @Tags client-player
// @Accept json
// @Produce json
// @Param playlistId query string true "Playlist ID" example(playlist-123456)
// @Param needVid query bool false "Whether to get video list" example(true)
// @Success 200 {object} api.PlaylistResponse "Return Playlist Details"
// @Router /api/client/playlist/info [get]
func (h *ClientPlayerHandler) PlaylistInfo(ctx *gin.Context) {
	playlistID := ctx.Query("playlistId")
	if playlistID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("playlistId is required"), nil)
		return
	}
	needVid := false
	if needVidStr := ctx.Query("needVid"); needVidStr != "" {
		var err error
		needVid, err = strconv.ParseBool(needVidStr)
		if err != nil {
			needVid = false // default to false on parsing error
		}
	}
	userID := ctx.GetString("user_id")
	lang := ctx.GetString("lang")

	//TODO Whether to add siteID?
	playlist, err := h.clientPlayerService.GetPlaylistInfo(ctx, playlistID, lang, needVid, userID)
	if err != nil {
		log.Error(ctx, "get playlist info failed, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}
	api.HandleSuccess(ctx, playlist)

}

// BatchPlaylistInfo godoc
// @Summary Batch Get Playlist Details
// @Description Batch get playlist details by IDs
// @Tags client-player
// @Accept json
// @Produce json
// @Param playlistIds query string true "Playlist IDs separated by commas" example("playlist-123456,playlist-123457")
// @Param needVid query bool false "Whether to get video list" example(false)
// @Success 200 {object} api.Response{data=[]api.PlaylistInfo} "Return Batch Playlist Details"
// @Router /api/client/playlist/batch-get [get]
func (h *ClientPlayerHandler) BatchPlaylistInfo(ctx *gin.Context) {
	playlistIdsStr := ctx.Query("playlistIds")
	if playlistIdsStr == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("playlistIds is required"), nil)
		return
	}

	// Split by comma to get playlist IDs
	playlistIDs := strings.Split(playlistIdsStr, ",")
	if len(playlistIDs) == 0 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("playlistIds cannot be empty"), nil)
		return
	}

	// Trim whitespace from each ID
	for i, id := range playlistIDs {
		playlistIDs[i] = strings.TrimSpace(id)
	}

	// Limit the number of playlist IDs to prevent abuse
	if len(playlistIDs) > 50 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("maximum 50 playlist IDs allowed per request"), nil)
		return
	}

	needVid := false
	if needVidStr := ctx.Query("needVid"); needVidStr != "" {
		var err error
		needVid, err = strconv.ParseBool(needVidStr)
		if err != nil {
			needVid = false // default to false on parsing error
		}
	}

	userID := ctx.GetString("user_id")
	lang := ctx.GetString("lang")

	log.AddNotice(ctx, "batch_playlist_ids_count", len(playlistIDs))

	// Call service layer to batch get playlist info
	playlists, err := h.clientPlayerService.BatchGetPlaylistInfo(ctx, playlistIDs, lang, needVid, userID)
	if err != nil {
		log.Error(ctx, "batch get playlist info failed, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	log.AddNotice(ctx, "batch_playlist_result_count", len(playlists))
	api.HandleSuccess(ctx, playlists)
}

// PlaylistVideos godoc
// @Summary Get Ordered Video List Under Playlist
// @Description Get Ordered Video List Under Playlist
// @Tags client-player
// @Accept json
// @Produce json
// @Param playlistId path string true "Playlist ID" example(playlist-123456)
// @Success 200 {object} api.IDListResponseData[api.VideoItem] "Return Ordered Video List Under Playlist"
// @Router /api/client/player/playlist/videos [get]
func (h *ClientPlayerHandler) PlaylistVideos(ctx *gin.Context) {
	playlistID := ctx.Query("playlistId")
	if playlistID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("playlistId is required"), nil)
		return
	}
	userID := ctx.GetString("user_id")

	//TODO Whether to add siteID
	sortVideos, err := h.clientPlayerService.GetVideosAndStatusByPlaylistID(ctx, playlistID, userID)
	if err != nil {
		log.Error(ctx, "get playlist videos failed, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}
	if sortVideos == nil {
		api.HandleSuccess(ctx, &api.IDListResponseData[*api.VideoItem]{
			Total:   0,
			HasMore: false,
			Items:   []*api.VideoItem{},
		})
		log.Warning(ctx, "playlist videos is empty, playlistid:"+playlistID)
		return
	}

	api.HandleSuccess(ctx, &api.IDListResponseData[*api.VideoItem]{
		Total:   len(sortVideos),
		HasMore: false,
		Items:   sortVideos,
	})

}

// AnonRegister Registration
// AnonRegister godoc
// @Summary Anonymous Registration
// @Description Anonymous Registration
// @Tags client-player
// @Accept json
// @Produce json
// @Success 200 {object} api.RegisterResponseData "Return Success"
// @Router /api/client/anonymous/register [post]
func (h *ClientPlayerHandler) AnonRegister(ctx *gin.Context) {
	// Parse parameters
	var req api.AnonRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}
	// siteID := middleware.GetSiteID(ctx)
	// // if siteID == "" {
	// // 	api.HandleError(ctx, http.StatusBadRequest, common.ErrSiteNotFound, nil)
	// // 	return
	// // }
	// // Call Service layer to handle
	token, siteID, err := h.clientPlayerService.AnonRegister(ctx, &req)
	if err != nil {
		log.Error(ctx, "anon register failed, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}
	api.HandleSuccess(ctx, api.RegisterResponseData{
		Token:  token,
		SiteID: siteID,
	})
}

// SiteInfo
// GetSiteInfo godoc
// @Summary Get Site Information
// @Description Get Site Information
// @Tags client-player
// @Accept json
// @Produce json
// @Param sitePath query string true "Site Path"  example(sitepath)
// @Param host query string false "Site Domain"  example(https://example.com)
// @Success 200 {object} api.SiteInfo "Return Site Information"
// @Router /api/client/site/info [get]
func (h *ClientPlayerHandler) SiteInfo(ctx *gin.Context) {
	sitePath := ctx.Query("sitePath")

	if sitePath == "_general" {
		log.Warning(ctx, "site path cannot be '_general'")
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("site path cannot be '_general'"), nil)
		return
	}
	site := &model.Site{}
	log.AddNotice(ctx, "query_path", sitePath)
	if sitePath != "" {
		var err error
		site, err = h.clientPlayerService.GetSiteByPath(ctx, sitePath)
		if err != nil {
			log.Error(ctx, "get site info failed, error: "+err.Error())
			api.HandleError(ctx, err, nil)
			return
		}
	} else { // Support getting host from query parameters or headers
		host := ctx.Query("domain")
		if host == "" {
			host = ctx.Request.Host
		}
		log.AddNotice(ctx, "host", host)
		if host == "" {
			log.Warning(ctx, "site path or host not found in query or header")
			api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("domain is empty"), nil)
			return
		}
		if types.IsOfficialDomain(host) { //如果是子域名按Path获取
			log.AddNotice(ctx, "is_official_domain", true)
			sitePath = types.ExtractSubdomain(host) // Extract subdomain, if needed
			var err error
			log.AddNotice(ctx, "host_path", sitePath)
			site, err = h.clientPlayerService.GetSiteByPath(ctx, sitePath)
			if err != nil {
				log.Error(ctx, "get site info failed, error: "+err.Error())
				api.HandleError(ctx, err, nil)
				return
			}
		} else {
			var err error
			site, err = h.clientPlayerService.GetSiteByHost(ctx, host)
			if err != nil {
				log.Error(ctx, "get site info by host failed, error: "+err.Error())
				api.HandleError(ctx, err, nil)
				return
			}
		}

	}

	// api.SiteInfo
	theme := site.Theme
	result := &api.SiteInfo{
		SiteID: site.SiteID,
		Name:   site.Name,
		OfficialDomain: &types.OfficialDomain{
			Subdomain: site.Path, // Assuming Path is the subdomain
		},
		Domain:            site.Domain,
		Logo:              site.Logo,
		Path:              site.Path,
		TemplateID:        site.TemplateID,
		TemplateName:      site.TemplateName,
		GoogleAnalyticsID: site.GoogleAnalyticsID,
		FacebookPixelID:   site.FacebookPixelID,
		ThinkingDataAppId: site.ThinkingDataAppId,
		Theme:             &theme,
		Seo: &api.SiteSeo{
			Title:       site.SeoTitle,
			Description: site.SeoDescription,
			Keywords:    site.SeoKeywords,
		},
		SiteMultiLang: site.I18n,
		SeoMultiLang:  site.SeoI18n,
	}
	api.HandleSuccess(ctx, result)
}

// SitePages godoc
// @Summary Get Published Site Pages
// @Description Get the current published site pages data from site_current_published table for public consumption
// @Tags client-player
// @Accept json
// @Produce json
// @Success 200 {object} api.Response{data=api.GetSitePagesResponse} "Returns published site pages data with version and published timestamp"
// @Router /api/client/site/pages [get]
func (h *ClientPlayerHandler) SitePages(ctx *gin.Context) {
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		siteID = ctx.Query("siteId")
		if siteID == "" {
			api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("site ID is required"), nil)
			return
		}

	}

	response, err := h.pagesBuilderService.GetSitePages(ctx, siteID)
	if err != nil {
		log.Error(ctx, "get site pages failed, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, response)
}

// NewRelease godoc
// @Summary Get New Release Playlists
// @Description Get the latest published playlists for a site, returns playlist ID, name, cover, and creation time
// @Tags client-player
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @param siteId query string false "Site ID" example("site-123")
// @Success 200 {object} api.NewReleasePlaylistsResponse "Returns new release playlists"
// @Router /api/client/site/new-release [get]
func (h *ClientPlayerHandler) NewRelease(ctx *gin.Context) {
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		siteID = ctx.GetString("siteId")
		if siteID == "" {
			api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("site ID is required"), nil)
			return
		}
	}

	// Get new release playlists from service
	playlists, err := h.playlistService.GetNewReleasePlaylistsBySiteID(ctx, siteID)
	if err != nil {
		log.Error(ctx, "get new release playlists failed, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	// Return the response
	response := api.NewReleasePlaylistsResponse{
		Response: api.Response{
			Code: 0,
			Info: "ok",
		},
		Data: playlists,
	}

	api.HandleSuccess(ctx, response.Data)
}

// PlaylistSearch godoc
// @Summary Search videos on the client side
// @Description Search videos by keyword for a given site, with pagination and sorting
// @Tags client-player
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param keyword query string true "Search keyword" example("tutorial")
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Items per page, default 10" minimum(1) maximum(50) example(10)
// @Param sortType query int false "Sort type (0:created time desc 1:name sort)" example(0)
// @Success 200 {object} api.IDListResponseData[string] "Returns video list in playlist"
// @Router /api/client/playlist/search [get]
func (h *ClientPlayerHandler) PlaylistSearch(ctx *gin.Context) {
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("site ID is required"), nil)
		return
	}

	keyword := ctx.Query("keyword")
	if keyword == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("keyword parameter is required"), nil)
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))
	sortType, _ := strconv.Atoi(ctx.DefaultQuery("sortType", "0")) // Default to sort by creation time descending
	lang := ctx.GetString("lang")
	if lang == "" {
		lang = "zh-TW"
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 { // Limit page size
		pageSize = 50
	}

	// Assuming playlistService.List signature is List(ctx, query, page, pageSize, sortType)
	playlists, total, err := h.playlistService.Search(ctx, siteID, keyword, page, pageSize, sortType, lang)
	if err != nil {
		log.Error(ctx, "search playlists failed, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	responseData := api.IDListResponseData[string]{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		Items:    playlists,
	}
	api.HandleSuccess(ctx, responseData)
}

// PlaylistRelatedRecommend godoc
// @Summary Get Related Playlists
// @Description Get related playlists based on a given playlist ID
// @Tags client-player
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param playlistID query string true "Playlist ID" example("playlist-123456")
// @Success 200 {object} api.IDListResponseData[string] "Returns related playlists"
// @Router /api/client/playlist/related [get]
func (h *ClientPlayerHandler) PlaylistRelatedRecommend(ctx *gin.Context) {
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("site ID is required"), nil)
		return
	}

	playlistID := ctx.Query("playlistId")
	if playlistID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("playlist ID is required"), nil)
		return
	}

	// Get related playlists from service
	relatedPlaylists, err := h.playlistService.GetRecAfterPlaylists(ctx, siteID, playlistID)
	if err != nil {
		log.Error(ctx, "get related playlists failed, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	// Prepare response
	// responseData := api.IDListResponseData[string]{
	// 	Total:    int(len(relatedPlaylists)),
	// 	Page:     1,
	// 	PageSize: len(relatedPlaylists),
	// 	Items:    relatedPlaylists,
	// }
	api.HandleSuccess(ctx, relatedPlaylists)
}

// ReportRecords godoc
// @Summary Report Video Playback Record
// @Description Report user video playback progress and related information
// @Tags client-player
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param data body api.PlaybackReportRequest true "Playback record information"
// @Success 200 {object} api.Response{} "Return report result"
// @Router /api/client/video/playback/records [post]
func (h *ClientPlayerHandler) ReportRecords(ctx *gin.Context) {
	// 从认证中间件获取用户ID
	userID := ctx.GetString("user_id")
	if userID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, fmt.Errorf("user not authenticated"), nil)
		return
	}

	// 从中间件获取站点ID
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("site ID cannot be empty"), nil)
		return
	}

	// 绑定并校验请求参数
	var req api.PlaybackReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// 检查必要参数
	if req.VID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("video ID is required"), nil)
		return
	}

	// 创建播放记录
	record := &model.UserPlayRecord{
		UserID:        userID,
		SiteID:        siteID,
		VID:           req.VID,
		PlaylistID:    req.PlaylistID,
		EpisodeNumber: req.EpisodeNumber,
		Progress:      req.Progress,
		Duration:      req.Duration,
		VideoTitle:    req.VideoTitle,
		PlaylistTitle: req.PlaylistTitle,
		Cover:         req.Cover,
	}

	// 保存或更新播放记录
	err := h.clientPlayerService.CreateOrUpdatePlayRecord(ctx, record)
	if err != nil {
		log.Error(ctx, "failed to create or update user play record, "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	// 返回成功响应
	api.HandleSuccess(ctx, nil)
}

// GetWatchHistory godoc
// @Summary Get user's video viewing history
// @Description Returns the video viewing history for the authenticated user
// @Tags client-player
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Items per page, default 10" minimum(1) maximum(50) example(10)
// @Success 200 {object} api.Response{data=api.VideoHistoryResponse} "Returns user's viewing history"
// @Router /api/client/video/history [get]
func (h *ClientPlayerHandler) GetWatchHistory(ctx *gin.Context) {
	// Get user ID from authentication context
	userID := ctx.GetString("user_id")
	if userID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, fmt.Errorf("user not authenticated"), nil)
		return
	}

	// Get site ID from request context
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("site ID cannot be empty"), nil)
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	// Retrieve play records from repository
	records, total, err := h.clientPlayerService.GetUserPlayHistory(ctx, userID, siteID, page, pageSize)
	if err != nil {
		log.Error(ctx, "failed to get user play records, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	// Convert play records to response items
	items := make([]*api.VideoHistoryItem, 0, len(records))
	for _, record := range records {
		items = append(items, &api.VideoHistoryItem{
			VID:           record.VID,
			Title:         record.VideoTitle,
			PlaylistID:    record.PlaylistID,
			PlaylistTitle: record.PlaylistTitle,
			EpisodeNumber: record.EpisodeNumber,
			Progress:      record.Progress,
			Duration:      record.Duration,
			Cover:         record.Cover,
			LastPlayedAt:  record.LastPlayedAt.Unix(),
		})
	}

	// Prepare and return response
	response := &api.VideoHistoryResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}
	log.AddNotice(ctx, "video_history_count", len(items))

	api.HandleSuccess(ctx, response)
}

// AllPlaylistID godoc
// @Summary Get All Playlist IDs for a Site
// @Description Get all published playlist IDs for the specified site with pagination support
// @Tags client-player
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Page size (default: 20)"
// @Success 200 {object} api.IDListResponseData[api.PlaylistSlugItem] "Returns paginated playlist slug data with i18n support"
// @Router /api/client/site/playlists [get]
func (h *ClientPlayerHandler) AllPlaylistID(ctx *gin.Context) {
	// Get site ID from request context
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("site ID is required"), nil)
		return
	}

	// Get pagination parameters from query string
	// Default to page 1 and pageSize 20
	page := 1
	pageSize := 20

	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := ctx.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	// Get all playlist IDs from service
	items, total, err := h.clientPlayerService.GetAllPlaylistIDs(ctx, siteID, page, pageSize)
	if err != nil {
		log.Error(ctx, "get all playlist IDs failed, error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	// Assemble pagination response
	result := &api.IDListResponseData[*api.PlaylistSlugItem]{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		HasMore:  int64(page*pageSize) < total,
		Items:    items,
	}

	api.HandleSuccess(ctx, result)
}

// UploadFile godoc
// @Summary Upload file without authentication
// @Description Upload an image file without authentication (for client users)
// @Tags client-player
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image file to upload"
// @Success 200 {object} api.Response{data=string} "Returns the path of the successfully uploaded image"
// @Router /api/client/upload [post]
func (h *ClientPlayerHandler) UploadFile(ctx *gin.Context) {
	form, err := ctx.MultipartForm()
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}
	files := form.File["file"]
	if len(files) == 0 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrNoFilesFound, nil)
		return
	}
	if len(files) > 1 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrTooManyFiles, nil)
		return
	}
	file := files[0]

	// Generate a unique ID for anonymous uploads
	anonymousID := "client_" + uuid.NewString()

	// Storage path: base_path/res/user_upload/md5sum(anonymous_id)/xxx.jpg
	md5Hash := md5.Sum([]byte(anonymousID))
	fileName := uuid.NewString() + filepath.Ext(file.Filename)
	imgPath := "res/user_upload/" + hex.EncodeToString(md5Hash[:]) + "/"
	fullPath := h.conf.GetString("storage.local.path") + "/" + imgPath
	imgFullName := fullPath + "/" + fileName

	// Check if directory exists, if not create it
	if _, err := os.Stat(filepath.Dir(imgFullName)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(imgFullName), 0755); err != nil {
			log.Error(ctx, "create directory failed: "+err.Error())
			api.HandleError(ctx, err, nil)
			return
		}
	}

	// Create destination file
	dst, err := os.Create(imgFullName)
	if err != nil {
		log.Error(ctx, "create file failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}
	defer dst.Close()

	// Open source file
	src, err := file.Open()
	if err != nil {
		log.Error(ctx, "file open error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}
	defer src.Close()

	// Copy file
	if _, err = dst.ReadFrom(src); err != nil {
		log.Error(ctx, "file copy error: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	// Return relative path
	imgUrl := types.ImageURL(imgPath + fileName)
	api.HandleSuccess(ctx, imgUrl)
}

// TransferSave godoc
// @Summary Save transfer data
// @Description Save browser data to transfer to another browser, returns a sync_id
// @Tags client-player
// @Accept json
// @Produce json
// @Param data body map[string]interface{} true "Data to transfer"
// @Success 200 {object} api.Response{data=map[string]string} "Returns sync_id"
// @Router /api/client/transfer/save [post]
func (h *ClientPlayerHandler) TransferSave(ctx *gin.Context) {
	var req struct {
		Data string `json:"data" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("TransferSave ShouldBindJSON failed. Possible oversized payload (413) or malformed format (500)",
			zap.Error(err),
			zap.Int64("content_length", ctx.Request.ContentLength),
		)
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}
	transferKey := "sync_" + uuid.NewString()

	h.logger.WithContext(ctx).Info("TransferSave request successful",
		zap.String("sync_id", transferKey),
		zap.Int("payload_length", len(req.Data)),
	)
	transferCache.Store(transferKey, transferData{
		data:      req.Data,
		expiredAt: time.Now().Add(10 * time.Minute),
	})
	api.HandleSuccess(ctx, map[string]string{
		"sync_id": transferKey,
	})
}

// TransferGet godoc
// @Summary Get transfer data
// @Description Get previously saved browser data using sync_id
// @Tags client-player
// @Accept json
// @Produce json
// @Param data body map[string]string true "Request body containing sync_id"
// @Success 200 {object} api.Response{data=map[string]string} "Returns data"
// @Router /api/client/transfer/get [post]
func (h *ClientPlayerHandler) TransferGet(ctx *gin.Context) {
	var req struct {
		SyncID string `json:"sync_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("TransferGet ShouldBindJSON failed", zap.Error(err))
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	syncID := req.SyncID
	if syncID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("sync_id is required"), nil)
		return
	}
	if val, ok := transferCache.Load(syncID); ok {
		if td, valid := val.(transferData); valid {
			if time.Now().Before(td.expiredAt) {
				// One time only, delete after getting
				transferCache.Delete(syncID)
				h.logger.WithContext(ctx).Info("TransferGet hit cache successfully",
					zap.String("sync_id", syncID),
					zap.Int("payload_length", len(td.data)),
				)
				api.HandleSuccess(ctx, map[string]string{
					"data": td.data,
				})
				return
			} else {
				h.logger.WithContext(ctx).Warn("TransferGet cache expired",
					zap.String("sync_id", syncID),
					zap.Time("expired_at", td.expiredAt),
				)
				transferCache.Delete(syncID)
			}
		}
	} else {
		h.logger.WithContext(ctx).Warn("TransferGet cache not found",
			zap.String("sync_id", syncID),
		)
	}

	api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("invalid or expired sync_id"), nil)
}
