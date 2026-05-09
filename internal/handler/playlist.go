package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/service"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
)

type PlaylistHandler struct {
	*Handler
	playlistService service.PlaylistService
	videoService    service.VideoService
	siteService     service.SiteService
}

func NewPlaylistHandler(
	handler *Handler,
	playlistService service.PlaylistService,
	videoService service.VideoService,
	siteService service.SiteService,
) *PlaylistHandler {
	return &PlaylistHandler{
		Handler:         handler,
		playlistService: playlistService,
		siteService:     siteService,
		videoService:    videoService,
	}
}

// Create godoc
// @Summary Create playlist
// @Schemes
// @Description Create a new playlist and its SEO information
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.PlaylistInfo true "Create playlist parameters"
// @Success 200 {object} api.Response "Return created playlist information"
// @Router /api/playlist/create [post]
func (h *PlaylistHandler) Create(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.PlaylistInfo
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	playlistID, err := h.playlistService.CreatePlaylist(ctx, creatorID, req)
	if err != nil {
		log.Error(ctx, "create playlist failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, playlistID)
}

// Get godoc
// @Summary Get playlist information
// @Schemes
// @Description Get detailed information of the specified playlist, including basic information and SEO information
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param playlistId query string true "Playlist ID" example("playlist-123456")
// @Success 200 {object} api.PlaylistResponse "Return detailed playlist information"
// @Router /api/playlist/get [get]
func (h *PlaylistHandler) Get(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	playlistID := ctx.Query("playlistId")
	if playlistID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("playlistId is required"), nil)
		return
	}

	playlist, err := h.playlistService.GetPlaylist(ctx, playlistID)
	if err != nil {
		log.Error(ctx, "get playlist failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, playlist)
}

// BatchGet
// @Summary Batch get playlist information
// @Schemes
// @Description Batch get detailed information of playlists, including basic information and SEO information
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string false "Bearer user token"
// @Param playlistIds query string true "Playlist ID list" example("playlist-123456,playlist-123457")
// @Success 200 {object} api.PlaylistList "Return detailed playlist information"
// @Router /api/playlist/batch-get [get]
func (h *PlaylistHandler) BatchGet(ctx *gin.Context) {
	pStr := ctx.Query("playlistIds")
	if pStr == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrBadRequest, nil)
		return
	}
	playlistIDs := strings.Split(pStr, ",")
	if len(playlistIDs) == 0 {
		api.HandleSuccess(ctx, nil)
		return
	}
	if len(playlistIDs) > 20 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrTooManyPlaylistsGetDetail, nil)
		return
	}

	playlists, err := h.playlistService.BatchGet(ctx, playlistIDs)
	if err != nil {
		log.Error(ctx, "batch get playlists failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}
	playListInfos := make([]*api.PlaylistInfo, 0, len(playlists))
	for _, p := range playlists {
		videoCount := 0
		if p.OrderVids != "" {
			orderData := &api.VideoSortData{}
			err = json.Unmarshal([]byte(p.OrderVids), orderData)
			if err != nil {
				videoCount = 0
			} else {
				videoCount = len(orderData.VIDs)
			}
		}

		playListInfos = append(playListInfos, &api.PlaylistInfo{
			PlaylistID:  p.PlaylistID,
			Title:       p.Title,
			Slug:        p.Slug,
			Description: p.Description,
			Cover:       p.Cover,
			Status:      p.Status,
			VideoCount:  videoCount,
			Version:     p.Version,
			CreatedAt:   p.CreatedAt.Unix(),
			UpdatedAt:   p.UpdatedAt.Unix(),
		})
	}
	log.AddNotice(ctx, "playlist_batch_len", len(playListInfos))
	api.HandleSuccess(ctx, &api.PlaylistList{
		Items: playListInfos,
	})

}

// Modify godoc
// @Summary Modify playlist information
// @Schemes
// @Description Modify specific playlist information, including basic information and SEO information
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.PlaylistInfo true "Modify playlist parameters"
// @Success 200 {object} api.Response "Modification successful"
// @Router /api/playlist/modify [post]
func (h *PlaylistHandler) Modify(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.PlaylistInfo
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	err := h.playlistService.ModifyPlaylist(ctx, req)
	if err != nil {
		log.Error(ctx, "modify playlist failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, nil)
}

// Delete godoc
// @Summary Delete playlist
// @Schemes
// @Description Delete one or more playlists
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.PlaylistDeleteRequest true "Delete playlist parameters"
// @Success 200 {object} api.Response "Deletion successful"
// @Router /api/playlist/delete [post]
func (h *PlaylistHandler) Delete(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.PlaylistDeleteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	err := h.playlistService.DeletePlaylists(ctx, creatorID, req.PlaylistIDs)
	if err != nil {
		log.Error(ctx, "delete playlists failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, nil)
}

// Videos godoc
// @Summary Get videos in playlist
// @Schemes
// @Description Get videos in the playlist with pagination, supports filtering by status
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param playlistId query string true "Playlist ID" example("playlist-123456")
// @Param status query int false "Video status, -1 for all" example(1)
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Items per page, default 20" minimum(1) maximum(100) example(20)
// @Success 200 {object} api.IDListResponseData[string] "Return video list IDs in playlist"
// @Router /api/playlist/videos [get]
func (h *PlaylistHandler) Videos(ctx *gin.Context) {
	playlistID := ctx.Query("playlistId")
	if playlistID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrBadRequest, nil)
		return
	}

	// 从 header 中获取 UTM 来源
	utmSource := ctx.GetHeader("Utm-Source0")

	// 验证 playlist 是否存在并符合 UTM 条件
	playlist, err := h.playlistService.VerifyPlaylistAccess(ctx, playlistID, utmSource)
	if err != nil {
		log.Error(ctx, "verify playlist failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	if playlist == nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusNotFound, common.ErrPlaylistNotFound, nil)
		return
	}

	page := 1
	pageSize := 20
	var pStatus *int
	if p := ctx.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := ctx.Query("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}
	if sParam := ctx.Query("status"); sParam != "" {
		if v, err := strconv.Atoi(sParam); err == nil {
			pStatus = &v
		}
	}
	query := &model.VideoQuery{
		PlaylistID: playlistID,
		Status:     pStatus,
	}
	videos, total, err := h.videoService.List(ctx, query, page, pageSize, model.PlaylistSortByCreatedAtDesc)
	if err != nil {
		log.Error(ctx, "list playlist videos failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	items := make([]string, 0, len(videos))
	for _, v := range videos {
		items = append(items, v.VID)
	}

	responseData := api.IDListResponseData[string]{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}
	api.HandleSuccess(ctx, responseData)
}

// VideosV2 godoc
// @Summary Get videos in playlist V2
// @Schemes
// @Description Get videos in the playlist with pagination, supports filtering by status
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param playlistId query string true "Playlist ID" example("playlist-123456")
// @Param status query int false "Video status, -1 for all" example(1)
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Items per page, default 20" minimum(1) maximum(100) example(20)
// @Success 200 {object} api.IDListResponseData[string] "Return video list IDs in playlist"
// @Router /api/playlist/videos [get]
func (h *PlaylistHandler) VideosV2(ctx *gin.Context) {
	h.VideosOrder(ctx)
}

// AddVideos godoc
// @Summary Add videos to playlist
// @Schemes
// @Description Add multiple videos to the specified playlist
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.PlaylistAddVideosRequest true "Add videos parameters"
// @Success 200 {object} api.Response "Addition successful"
// @Router /api/playlist/add-videos [post]
func (h *PlaylistHandler) AddVideos(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.PlaylistAddVideosRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	err := h.playlistService.AddVideos(ctx, req.PlaylistID, req.VIDs)
	if err != nil {
		log.Error(ctx, "add videos to playlist failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, nil)
}

// RemoveVideos godoc
// @Summary Remove videos from playlist
// @Schemes
// @Description Remove multiple videos from specified playlist
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.PlaylistDelVideosRequest true "Remove videos parameters"
// @Success 200 {object} api.Response "Removal successful"
// @Router /api/playlist/remove-videos [post]
func (h *PlaylistHandler) RemoveVideos(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.PlaylistDelVideosRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	err := h.playlistService.RemoveVideos(ctx, req.PlaylistID, req.VIDs)
	if err != nil {
		log.Error(ctx, "delete videos from playlist failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, nil)
}

// List godoc
// @Summary Query playlists under site
// @Schemes
// @Description Paginate query all playlist information under specified site
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param orderType query int false "Sort type (0:Create time descending 1:Name sorting)" example(0)
// @Param status query int false "Playlist status, if not provided queries all" example(1)
// @Param siteId query string false "Site ID, if not provided queries all sites" example("123e4567-e89b-12d3-a456-426614174000")
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Items per page, default 10" minimum(1) maximum(20) example(10)
// @Success 200 {object} api.IDListResponseData[string] "Return playlists information under site"
// @Router /api/playlist/list [get]
func (h *PlaylistHandler) List(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	siteID := ctx.Query("siteId")
	page := 1
	pageSize := 10
	orderType := 0
	status := -1
	if p := ctx.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := ctx.Query("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}
	if ot := ctx.Query("orderType"); ot != "" {
		if v, err := strconv.Atoi(ot); err == nil {
			orderType = v
		}
	}
	if s := ctx.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			status = v
		}
	}

	var playlists []*model.Playlist
	var total int64
	var err error
	var ps *int
	if status != -1 {
		ps = &status
	}
	query := &model.PlaylistQuery{
		CreatorID: creatorID,
		Status:    ps,
		SiteID:    siteID,
	}
	playlists, total, err = h.playlistService.List(ctx, query, page, pageSize, orderType)
	if err != nil {
		log.Error(ctx, "list playlists failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	items := make([]string, 0, len(playlists))
	for _, p := range playlists {
		items = append(items, p.PlaylistID)
	}
	responseData := api.IDListResponseData[string]{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}
	api.HandleSuccess(ctx, responseData)
}

// Search godoc
// @Summary Search playlists
// @Schemes
// @Description Search videos by keyword, supports pagination
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param orderType query int false "Sort type (0:Create time descending 1:Name sorting)" example(1)
// @Param siteId query string false "Site ID, if not provided queries all sites" example("123e4567-e89b-12d3-a456-426614174000")
// @Param excludeSiteId query string false "Exclude site ID, if set, query results will not include playlists from this site, cannot be the same as siteID" example("123e4567-e89b-12d3-a456-426614174000")
// @Param status query int false "Video playlist status" example(1)
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Items per page, default 10" minimum(1) maximum(20) example(20)
// @Param keyword query string false "Search keyword" example(playlist)
// @Success 200 {object} api.IDListResponseData[string] "Return search results"
// @Router /api/playlist/search [get]
func (h *PlaylistHandler) Search(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	siteID := ctx.Query("siteId")
	excludeSiteID := ctx.Query("excludeSiteId")
	keyword := ctx.Query("keyword")
	page := 1
	pageSize := 10
	orderType := 0
	status := -1
	if p := ctx.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := ctx.Query("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}
	if ot := ctx.Query("orderType"); ot != "" {
		if v, err := strconv.Atoi(ot); err == nil {
			orderType = v
		}
	}
	if s := ctx.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			status = v
		}
	}

	var ps *int
	if status != -1 {
		ps = &status
	}
	query := &model.PlaylistQuery{
		CreatorID:     creatorID,
		Status:        ps,
		SiteID:        siteID,
		ExcludeSiteId: excludeSiteID,
		Keyword:       keyword,
	}
	playlists, total, err := h.playlistService.List(ctx, query, page, pageSize, orderType)
	if err != nil {
		log.Error(ctx, "search playlists failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	items := make([]string, 0, len(playlists))
	for _, p := range playlists {
		items = append(items, p.PlaylistID)
	}

	responseData := api.IDListResponseData[string]{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}

	api.HandleSuccess(ctx, responseData)
}

// VideosUpdateOrder godoc
// @Summary Video order in playlist
// @Schemes
// @Description Order of videos in playlist
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.PlaylistVideosOrder true "Create playlist parameters"
// @Success 200 {object} api.Response "Return created playlist information"
// @Router /api/playlist/videos/update-order [post]
func (h *PlaylistHandler) VideosUpdateOrder(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.PlaylistVideosOrder
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	err := h.playlistService.UpdateVideosOrder(ctx, req)
	if err != nil {
		log.Error(ctx, "update videos order failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, nil)
}

// VideosOrder
// @Summary Get video order in playlist
// @Schemes
// @Description Get the order of videos in playlist
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param playlistId query string true "Playlist ID" example("playlist-123456")
// @Success 200 {object} api.PlaylistVideosOrder "Return video list IDs in playlist"
// @Router /api/playlist/videos/order [get]
func (h *PlaylistHandler) VideosOrder(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	playlistID := ctx.Query("playlistId")
	if playlistID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("playlistId is required"), nil)
		return
	}

	ordrData, err := h.playlistService.GetVideosOrder(ctx, playlistID)
	if err != nil {
		log.Error(ctx, "get playlist videos order failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, ordrData)
}

// ChangeAccessType godoc
// @Summary Modify playlist access type and payment settings
// @Description Modify playlist access type (free, paid, member-exclusive) and price settings
// @Tags playlist
// @Accept json
// @Produce json
// @Param request body api.PlaylistAccessChangeRequest true "Access type and payment settings"
// @Success 200 {object} api.Response "Operation successful"
// @Router /api/playlist/change/access/type [post]
func (h *PlaylistHandler) ChangeAccessType(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.PlaylistAccessChangeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	err := h.playlistService.ChangeAccessType(ctx, creatorID, req)
	if err != nil {
		log.Error(ctx, "change playlist access type failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, nil)
}

// TranslatePlaylist godoc
// @Summary Translate playlist to multiple languages
// @Description Translate playlist information (title, description, tags, SEO) to multiple languages using AI translation service. Supports re-translation to update existing translations.
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.PlaylistTranslateRequest true "Translate playlist parameters"
// @Success 200 {object} api.PlaylistTranslateResponse "Return translation results"
// @Router /api/playlist/i18n/create [post]
func (h *PlaylistHandler) TranslatePlaylist(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.PlaylistTranslateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	result, err := h.playlistService.TranslatePlaylist(ctx, req)
	if err != nil {
		log.Error(ctx, "translate playlist failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, result)
}

// BatchModifyI18n godoc
// @Summary Batch modify playlist i18n information
// @Schemes
// @Description Batch modify i18n information for multiple playlists, supports creating or updating translations for fields like title, description, tags, and SEO content
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.PlaylistI18nModifyRequest true "Batch modify i18n parameters"
// @Success 200 {object} api.Response "Modification successful"
// @Router /api/playlist/i18n/batch-modify [post]
func (h *PlaylistHandler) BatchModifyI18n(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.PlaylistI18nModifyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// Validate request
	if len(req.Data) == 0 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("request body cannot be empty"), nil)
		return
	}

	if len(req.Data) > 100 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("too many items, maximum 100 per request"), nil)
		return
	}

	err := h.playlistService.BatchModifyPlaylistI18n(ctx, req.Data)
	if err != nil {
		log.Error(ctx, "batch modify playlist i18n failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, nil)
}

// GetPlaylistI18n godoc
// @Summary Get playlist i18n information
// @Schemes
// @Description Get all i18n translation information for a specific playlist, including translated title, description, tags, and SEO content in different languages
// @Tags playlist
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param playlistId query string true "Playlist ID" example("playlist-123456")
// @Success 200 {object} api.PlaylistI18nResponse "Return detailed playlist i18n information"
// @Router /api/playlist/i18n [get]
func (h *PlaylistHandler) GetPlaylistI18n(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	playlistID := ctx.Query("playlistId")
	if playlistID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("playlistId is required"), nil)
		return
	}

	result, err := h.playlistService.GetPlaylistI18n(ctx, playlistID)
	if err != nil {
		log.Error(ctx, "get playlist i18n failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, result)
}
