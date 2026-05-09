package handler

import (
	"fmt"
	"net/http"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/service"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type VideoHandler struct {
	*Handler
	videoService service.VideoService
}

func NewVideoHandler(
	handler *Handler,
	videoService service.VideoService,
) *VideoHandler {
	return &VideoHandler{
		Handler:      handler,
		videoService: videoService,
	}
}

// Upload godoc
// @Summary Batch upload videos
// @Schemes
// @Description Batch upload video files, returns a list of created video VIDs
// @Tags video
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param playlistId formData string false "Playlist ID, if not provided, videos won't be added to any playlist" example(playlistid)
// @Success 200 {object} api.VideoUploadResponse "Returns a list of uploaded video VIDs"
// @Router /api/video/upload [post]
func (h *VideoHandler) Upload(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	playlistID := ctx.PostForm("playlistId")
	log.AddNotice(ctx, "upload_playlist_id", playlistID)
	log.AddNotice(ctx, "upload_creator_id", creatorID)
	form, err := ctx.MultipartForm()
	if err != nil {
		log.Warning(ctx, fmt.Sprintf("parse form error: %s", err.Error()))
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	files := form.File["files"]
	filesCount := len(files)
	if filesCount == 0 {
		log.Warning(ctx, "no files found")
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrNoFilesFound, nil)
		return
	}
	// Synchronous upload only supports one file
	if filesCount > 1 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrTooManyFiles, nil)
	}

	vids, err := h.videoService.Upload(ctx, creatorID, files, playlistID)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("upload video failed: %s", err.Error()))
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, api.VideoUploadResponseData{
		VIDs: vids,
	})
}

// UploadSubtitle godoc
// @Summary Upload video subtitle
// @Schemes
// @Description Upload video subtitle file, supports SRT and VTT formats
// @Tags video
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param vid formData string true "Video ID" example("vid-123456")
// @Param file formData file true "Subtitle file, supports SRT and VTT formats"
// @Success 200 {object} api.Response "Returns the subtitle URL"
// @Router /api/video/upload-subtitle [post]
func (h *VideoHandler) UploadSubtitle(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	vid := ctx.PostForm("vid")
	if vid == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("vid is required"), nil)
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrNoFilesFound, nil)
		return
	}

	file := files[0]
	subtitleURL, err := h.videoService.UploadSubtitle(ctx, creatorID, file, vid)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("upload subtitle failed: %s", err.Error()))
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, types.VideoUrl(subtitleURL))
}

// RegenerateCover
func (h *VideoHandler) RegenerateCover(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	playlistIDs := ctx.Query("playlistids")
	if playlistIDs == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("playlistids is required"), nil)
		return
	}

	covers, err := h.videoService.RegenerateCover(ctx, creatorID, playlistIDs)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("regenerate cover failed: %s", err.Error()))
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, covers)
}

// Upload godoc
// @Summary Replace video, keeping the same video ID
// @Schemes
// @Description Synchronously replace a video
// @Tags video
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param vid query string true "Video ID to replace" example("vid-123456")
// @Success 200 {object} api.VideoReplaceResonse "Replacement successful"
// @Router /api/video/replace [post]
func (h *VideoHandler) Replace(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
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

	vid := ctx.Query("vid")
	nv, nc, err := h.videoService.Replace(ctx, creatorID, vid, file)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("replace video failed: %s", err.Error()))
		api.HandleError(ctx, err, nil)
		return
	}
	coverURL := types.ImageURL(nc)
	videoURL := types.VideoUrl(nv)
	log.AddNotice(ctx, "replace_video_id", vid)
	log.AddNotice(ctx, "replace_video_url", videoURL)
	api.HandleSuccess(ctx, &api.VideoReplaceData{
		VID:            vid,
		Cover:          &coverURL,
		VideoSourceUrl: &videoURL,
	})
}

// Get godoc
// @Summary Get video details
// @Schemes
// @Description Query video details by video ID
// @Tags video
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param vid query string true "Video ID" example("vid-123456")
// @Param need_seo query bool false "Whether SEO information is needed" default(false)
// @Success 200 {object} api.VideoInfo "Returns detailed video information"
// @Router /api/video/get [get]
func (h *VideoHandler) Get(ctx *gin.Context) {
	// creatorID := ctx.GetString("creator_id")
	// if creatorID == "" {
	// 	api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
	// 	return
	// }

	vid := ctx.Query("vid")
	if vid == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("vid is required"), nil)
	}

	video, err := h.videoService.GetVideoAndSeo(ctx, vid)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("get video failed: %s", err.Error()))
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, video)
}

// BatchGet
// @Summary Batch get brief video information
// @Schemes
// @Description Query brief video information by batch video IDs (note quantity control...)
// @Tags video
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string false "Bearer user token"
// @Param vids query string true "Video ID list, comma separated" example("vid-123456,vid-123457")
// @Success 200 {object} api.VideoListResponseData "Returns brief video information list"
// @Router /api/video/batch-get [get]
func (h *VideoHandler) BatchGet(ctx *gin.Context) {
	vids := ctx.Query("vids")
	if vids == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, common.ErrBadRequest, nil)
		return
	}

	vidList := strings.Split(vids, ",")
	filtered := vidList[:0]
	for _, vid := range vidList {
		if vid != "" {
			filtered = append(filtered, vid)
		}
	}
	vidList = filtered
	videos, err := h.videoService.GetVideoByVIDs(ctx, vidList)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("get videos failed: %s", err.Error()))
		api.HandleError(ctx, err, nil)
		return
	}
	if len(videos) == 0 {
		api.HandleSuccess(ctx, api.VideoListResponseData{
			Items: []*api.VideoInfo{},
		})
		return
	}
	if len(videos) > 20 {
		h.logger.Warn("too many videos", zap.Int("count", len(videos)))
		api.HandleError(ctx, common.ErrTooManyVideosGetDetail, nil)
		return
	}

	videoInfos := make([]*api.VideoInfo, 0, len(videos))
	for _, v := range videos {
		info := &api.VideoInfo{
			VID:         v.VID,
			Title:       v.Title,
			Description: v.Description,
			Tags:        v.Tags,
			Cover:       v.Cover,
			Status:      v.Status,
			CreatedAt:   v.CreatedAt.Unix(),
			UpdatedAt:   v.UpdatedAt.Unix(),
			Subtitles:   v.Subtitles,
			Config:      v.Config,
		}
		// Populate sources for each video (N+1; acceptable for small batches)
		if sources, err := h.videoService.ListSources(ctx, v.VID); err == nil {
			info.Sources = sources
		} else {
			h.logger.Warn("list sources failed", zap.String("vid", v.VID), zap.Error(err))
		}
		videoInfos = append(videoInfos, info)
	}

	responseData := api.VideoListResponseData{
		Items: videoInfos,
	}

	api.HandleSuccess(ctx, responseData)
}

// Modify godoc
// @Summary Modify video information
// @Schemes
// @Description Modify basic information of a specified video
// @Tags video
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.VideoInfo true "Video modification parameters"
// @Success 200 {object} api.Response "Modification successful"
// @Router /api/video/modify [post]
func (h *VideoHandler) Modify(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.VideoInfo
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	err := h.videoService.ModifyVideo(ctx, req)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("modify video failed: %s", err.Error()))
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, nil)
}

// Delete godoc
// @Summary Delete videos
// @Schemes
// @Description Batch delete specified videos
// @Tags video
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.VideoDeleteRequest true "Video deletion parameters"
// @Success 200 {object} api.Response "Deletion successful"
// @Router /api/video/delete [post]
func (h *VideoHandler) Delete(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.VideoDeleteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}
	log.AddNotice(ctx, "delete_videos_count", len(req.VIDs))
	err := h.videoService.DeleteVideos(ctx, req.VIDs)

	if err != nil {
		log.Error(ctx, fmt.Sprintf("delete videos failed: %s", err.Error()))
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, nil)
}

// Search godoc
// @Summary Display all videos of current creator
// @Schemes
// @Description Search videos by keyword, supports pagination
// @Tags video
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param sortType query int false "Sort type (0:created time desc 1:name sort)" example(1)
// @Param status query int false "Video status" example(1)
// @Param uploadStatus query int false "Video upload status, comma separated, supports multiple status combinations" example(1,2)
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Page size, default 10" minimum(1) maximum(20) example(20)
// @Success 200 {object} api.IDListResponseData[string] "Returns video list in playlist"
// @Router /api/video/list [get]
func (h *VideoHandler) List(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	page := 1
	pageSize := 20
	var pStatus *int
	sortType := 0
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
	if statusP := ctx.Query("status"); statusP != "" {
		if v, err := strconv.Atoi(statusP); err == nil {
			statusValue := v
			pStatus = &statusValue
		}
	}
	if sortTypeP := ctx.Query("sortType"); sortTypeP != "" {
		if v, err := strconv.Atoi(sortTypeP); err == nil {
			sortType = v
		}
	}

	query := &model.VideoQuery{
		CreatorID: creatorID,
		Status:    pStatus,
	}
	videos, total, err := h.videoService.List(ctx, query, page, pageSize, sortType)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("list videos failed: %s", err.Error()))
		api.HandleError(ctx, err, nil)
		return
	}

	items := make([]string, len(videos))
	for i, v := range videos {
		items[i] = v.VID
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
// @Summary Search videos
// @Schemes
// @Description Search videos by keyword, supports pagination
// @Tags video
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param playlistId query string false "Playlist ID, if not provided, search all videos" example(playlistid)
// @Param excludePlaylistId query string false "Exclude playlist id, if not provided, search all videos" example(playlistid)
// @Param orderType query int false "Sort type (0:created time desc 1:name sort)" example(1)
// @Param status query int false "Video status" example(1)
// @Param uploadStatus query int false "Video upload status, comma separated, supports multiple status combinations" example(1,2)
// @Param page query int false "Page number, default 1" minimum(1) example(1)
// @Param pageSize query int false "Page size, default 10" minimum(1) maximum(20) example(20)
// @Param keyword query string true "Search keyword" example(tutorial)
// @Success 200 {object} api.IDListResponseData[string] "Returns search results"
// @Router /api/video/search [get]
func (h *VideoHandler) Search(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	page := 1
	pageSize := 20
	var pStatus *int
	keyword := ctx.Query("keyword")
	playlistID := ctx.Query("playlistId")
	excludePlaylistID := ctx.Query("excludePlaylistId")
	sortType := 0
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
			statusValue := v
			pStatus = &statusValue
		}
	}
	if sortTypeP := ctx.Query("orderType"); sortTypeP != "" {
		if v, err := strconv.Atoi(sortTypeP); err == nil {
			sortType = v
		}
	}

	query := &model.VideoQuery{
		CreatorID:         creatorID,
		Status:            pStatus,
		KeyWord:           keyword,
		PlaylistID:        playlistID,
		ExcludePlaylistID: excludePlaylistID,
	}
	videos, total, err := h.videoService.List(ctx, query, page, pageSize, sortType)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("search videos failed: %s", err.Error()))
		api.HandleError(ctx, err, nil)
		return
	}

	items := make([]string, 0, len(videos))
	for _, v := range videos {
		items = append(items, v.VID)
	}

	hasMore := page*pageSize < int(total)

	api.HandleSuccess(ctx, api.IDListResponseData[string]{
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
		HasMore:  hasMore,
		Items:    items,
	})
}

// AddSources godoc
// @Summary Add network sources for a video
// @Schemes
// @Description Add one or multiple non-local sources for a video
// @Tags video
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.VideoAddSourcesRequest true "Add sources request"
// @Success 200 {object} api.VideoAddSourcesResponse "Added successfully"
// @Router /api/video/sources/add [post]
func (h *VideoHandler) AddSources(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}
	var req api.VideoAddSourcesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}
	if len(req.Sources) == 0 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("sources is empty"), nil)
		return
	}
	if err := h.videoService.AddSources(ctx, req.VID, req.Sources); err != nil {
		api.HandleError(ctx, err, nil)
		return
	}
	api.HandleSuccess(ctx, struct {
		VID string `json:"vid"`
	}{VID: req.VID})
}

// InternalGetVideoConfig godoc
// @Summary Internal API: Get video config
// @Description Internal service-to-service API for getting video configuration. No authentication required.
// @Tags internal-video
// @Accept json
// @Produce json
// @Param vid query string true "Video ID"
// @Success 200 {object} api.InternalGetVideoConfigResponse "Config retrieved successfully"
// @Failure 400 {object} api.Response "Invalid request"
// @Failure 404 {object} api.Response "Video not found"
// @Router /api/internal/video/config [get]
func (h *VideoHandler) InternalGetVideoConfig(ctx *gin.Context) {
	vid := ctx.Query("vid")
	if vid == "" {
		api.HandleErrorWithHttpCode(ctx, 400, fmt.Errorf("vid is required"), nil)
		return
	}

	// Directly query video from database
	video, err := h.videoService.GetVideoByVIDOnly(ctx, vid)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("get video failed: %s", err.Error()))
		api.HandleError(ctx, err, nil)
		return
	}

	if video == nil {
		api.HandleErrorWithHttpCode(ctx, 404, common.ErrVideoNotFound, nil)
		return
	}

	// Return only VID and Config
	response := &api.InternalGetVideoConfigResponse{
		VID:    video.VID,
		Config: video.Config,
	}

	api.HandleSuccess(ctx, response)
}
