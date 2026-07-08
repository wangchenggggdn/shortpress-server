package handler

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/service"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type SiteHandler struct {
	*Handler
	siteService service.SiteService
}

func NewSiteHandler(
	handler *Handler,
	siteService service.SiteService,
) *SiteHandler {
	return &SiteHandler{
		Handler:     handler,
		siteService: siteService,
	}
}

// Get godoc
// @Summary Query site and SEO information
// @Schemes
// @Description Query basic site information and corresponding SEO information by site ID
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param siteId query string true "Unique site ID" example("123e4567-e89b-12d3-a456-426614174000")
// @Success 200 {object} api.SiteGetResponse "Returns site and SEO information"
// @Router /api/site/get [get]
func (h *SiteHandler) Get(ctx *gin.Context) {
	siteId := ctx.Query("siteId")
	if siteId == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}
	site, err := h.siteService.GetSiteAndSeo(ctx, siteId)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, site)
}

// BatchGet godoc
// @Summary Batch get site information
// @Schemes
// @Description Batch query brief site information by site ID
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string false "Bearer user token"
// @Param siteIds query string true "List of site IDs, comma-separated" example("site-123456,site-123457")
// @Success 200 {object} api.VideoListResponseData "Returns a list of brief site information"
// @Router /api/site/batch-get [get]
func (h *SiteHandler) BatchGet(ctx *gin.Context) {

	// 2. Get parameters
	vids := ctx.Query("siteIds")
	if vids == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteIds is required"), nil)
		return
	}

	// 3. Get video information
	vidList := strings.Split(vids, ",")
	log.AddNotice(ctx, "req_vids_len", len(vidList))
	if len(vidList) > 20 {
		api.HandleError(ctx, common.ErrTooManyVideosGetDetail, nil)
		return
	}
	sites, err := h.siteService.GetSites(ctx, vidList)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}
	if len(sites) == 0 {
		log.Warning(ctx, "no sites found for the given IDs")
		api.HandleSuccess(ctx, api.VideoListResponseData{
			Items: []*api.VideoInfo{},
		})
		return
	}

	log.AddNotice(ctx, "resp_len", len(sites))

	items := make([]*api.SiteInfo, 0, len(sites))
	for _, s := range sites {
		theme := s.Theme
		items = append(items, &api.SiteInfo{
			OfficialDomain: &types.OfficialDomain{
				Subdomain: s.Path,
			},
			SiteID:       s.SiteID,
			Domain:       s.Domain,
			Path:         s.Path,
			Redirect:     s.Redirect,
			Name:         s.Name,
			Logo:         s.Logo,
			TemplateID:   s.TemplateID,
			TemplateName: s.TemplateName, // 从 Service 层获取的 template name
			Status:       s.Status,
			Theme:        &theme,
		})
	}

	responseData := api.SiteListResponse{
		Items: items,
	}
	api.HandleSuccess(ctx, responseData)
}

// Create godoc
// @Summary Create site and default SEO information
// @Schemes
// @Description Create a site and its default SEO information. Optionally pass templateId to initialize pages from a template and save into site_builder_data.
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.SiteInfo true "Create site parameters"
// @Success 200 {object} api.SiteCreateResponse "Successfully created site response"
// @Router /api/site/create [post]
func (h *SiteHandler) Create(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.SiteInfo
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// if req.Domain == "" {
	//     req.Domain = DefultSiteDomain
	// }

	req.Path = strings.ToLower(req.Path)
	if err := h.validateSitePath(req.Path); err != nil {
		log.Error(ctx, "invalid site path: "+err.Error())
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	siteID, err := h.siteService.CreateSiteAndSeo(ctx, &req, creatorID)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}
	resp := api.SiteCreateResponse{
		Data: siteID,
	}
	api.HandleSuccess(ctx, resp)
}

func (h *SiteHandler) validateSitePath(path string) error {
	// Check if path is at least 6 characters long
	if len(path) < 6 || len(path) > 30 {
		return fmt.Errorf("path length must be at least 6 characters")
	}
	// Validate path contains only letters, numbers, and hyphens
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(path) {
		return fmt.Errorf("path can only contain letters, numbers, and hyphens")
	}
	return nil
}

// PathValid godoc
// @Summary Validate site path availability
// @Schemes
// @Description Check if a site path is valid and available for use
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param path query string true "Site path to validate" example("my-site")
// @Success 200 {object} api.Response "Path is valid and available"
// @Router /api/site/path/valid [get]
func (h *SiteHandler) PathValid(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}
	path := ctx.Query("path")
	path = strings.ToLower(path)
	if err := h.validateSitePath(path); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}
	// Check if path already exists
	exists, err := h.siteService.PathExists(ctx, path)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}
	if exists {
		api.HandleError(ctx, common.ErrSiteAlreadyExist, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// Modify godoc
// @Summary Modify site and SEO information
// @Schemes
// @Description Modify site information and SEO configuration, siteId is a required field.
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.SiteInfo true "Modify site parameters"
// @Success 200 {object} api.Response "Returns whether modification was successful"
// @Router /api/site/modify [post]
func (h *SiteHandler) Modify(ctx *gin.Context) {
	// 1. Get and validate user authentication information
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// 2. Parse request parameters
	var req api.SiteInfo
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// 3. Parameter validation
	if req.SiteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	// 4. Call service layer for modification
	err := h.siteService.ModifySiteAndSeo(ctx, &req)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}

	// 5. Return success response
	api.HandleSuccess(ctx, nil)
}

// AddPlaylists godoc
// @Summary Add playlists to site
// @Schemes
// @Description Add one or more video playlists to the target site.
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.SiteAddPlaylistsRequest true "Add playlists parameters"
// @Success 200 {object} api.Response "Successfully added playlists response"
// @Router /api/site/add-playlists [post]
func (h *SiteHandler) AddPlaylists(ctx *gin.Context) {
	// 1. Validate user authentication information
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// 2. Parse request parameters
	var req api.SiteAddPlaylistsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// 3. Parameter validation
	if req.SiteID == "" || len(req.PlaylistIDs) == 0 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId and playlistIds are required"), nil)
		return
	}

	// 4. Call service layer to add playlists
	err := h.siteService.AddPlaylists(ctx, req.SiteID, req.PlaylistIDs)
	if err != nil {
		log.Error(ctx, "add playlists failed, err: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	// 5. Return success response
	api.HandleSuccess(ctx, nil)
}

// RemovePlaylists godoc
// @Summary Remove playlists from site
// @Schemes
// @Description Remove one or more video playlists from the target site.
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.SiteDelPlaylistsRequest true "Remove playlists parameters"
// @Success 200 {object} api.Response "Successfully removed playlists response"
// @Router /api/site/remove-playlists [post]
func (h *SiteHandler) RemovePlaylists(ctx *gin.Context) {
	// 1. Validate user authentication information
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// 2. Parse request parameters
	var req api.SiteDelPlaylistsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// 3. Parameter validation
	if req.SiteID == "" || len(req.PlaylistIDs) == 0 {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId and playlistIds are required"), nil)
		return
	}

	// 4. Call service layer to remove playlists
	err := h.siteService.RemovePlaylists(ctx, req.SiteID, req.PlaylistIDs)
	if err != nil {
		log.Error(ctx, "delete playlists failed"+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	// 5. Return success response
	api.HandleSuccess(ctx, nil)
}

// List godoc
// @Summary Get user's site list
// @Schemes
// @Description Paginate query for all site information owned by the current user, including basic site information and status
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Success 200 {object} api.IDListResponseData[string] "Returns site list data"
// @Failure 401 {object} api.Response "Unauthorized"
// @Failure 500 {object} api.Response "Internal server error"
// @Router /api/site/list [get]
func (h *SiteHandler) List(ctx *gin.Context) {
	// 1. Validate user authentication information
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// 2. Call service layer to get data
	sites, err := h.siteService.GetSitesByCreator(ctx, creatorID)
	if err != nil {
		log.Error(ctx, "get creator sites failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	if sites == nil {
		api.HandleSuccess(ctx, api.IDListResponseData[string]{
			Items: []string{},
		})
		return
	}

	dataArrary := make([]string, 0, len(sites))
	for _, s := range sites {
		dataArrary = append(dataArrary, s.SiteID)
	}
	if len(dataArrary) == 0 {
		api.HandleSuccess(ctx, api.IDListResponseData[string]{
			Items: []string{},
		})
		return
	}
	log.AddNotice(ctx, "site_count", len(dataArrary))
	api.HandleSuccess(ctx, api.IDListResponseData[string]{
		Items: dataArrary})
}

// UserList godoc
// @Summary Get users for a specific site
// @Schemes
// @Description Retrieve users associated with a site with pagination, filtering, and sorting
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param siteId query string true "Site ID" example("site-123456")
// @Param page query int false "Page number" default(1) minimum(1)
// @Param pageSize query int false "Page size" default(20) minimum(1) maximum(100)
// @Param query query string false "Search term for email or nickname" example("john")
// @Param sortType query int false "Sort type: 0=login time desc, 1=created time desc, 2=login time asc, 3=created time asc" default(1)
// @Param status query int false "Status filter: -1=all, 1=inactive, 2=active, 3=disabled, 4=deleted" default(-1)
// @Success 200 {object} api.UserListResponseData "Returns list of users"
// @Failure 400 {object} api.Response "Bad request"
// @Failure 401 {object} api.Response "Unauthorized"
// @Router /api/site/user/list [get]
func (h *SiteHandler) UserList(ctx *gin.Context) {
	// 1. Validate user authentication
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	siteID := ctx.Query("siteId")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	page := 1
	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := ctx.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	sortType := 1 // Default to created time desc
	if sortTypeStr := ctx.Query("sortType"); sortTypeStr != "" {
		if st, err := strconv.Atoi(sortTypeStr); err == nil && st >= 0 && st <= 3 {
			sortType = st
		}
	}

	status := int8(-1) // Default to all statuses
	if statusStr := ctx.Query("status"); statusStr != "" {
		if st, err := strconv.ParseInt(statusStr, 10, 8); err == nil {
			status = int8(st)
		}
	}

	query := &model.UserQuery{
		SiteID:     siteID,
		SearchTerm: ctx.Query("query"),
		Status:     status,
	}

	users, total, err := h.siteService.GetSiteUsers(ctx, query, page, pageSize, sortType)
	if err != nil {
		log.Error(ctx, "get site users failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	responseData := api.UserListResponseData{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    users,
	}

	api.HandleSuccess(ctx, responseData)
}

// UserInfo godoc
// @Summary Get user information by email
// @Schemes
// @Description Retrieve user information for a specific email within a site
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param siteId query string true "Site ID" example("site-123456")
// @Param email query string true "User email" example("user@example.com")
// @Success 200 {object} api.UserInfo "Returns user information"
// @Router /api/site/user/info [get]
func (h *SiteHandler) UserInfo(ctx *gin.Context) {
	// 1. Validate user authentication
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// 2. Get and validate request parameters
	siteID := ctx.Query("siteId")
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteId is required"), nil)
		return
	}

	email := strings.TrimSpace(ctx.Query("email"))
	userID := strings.TrimSpace(ctx.Query("userId"))
	if userID == "" && email == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("userId or email is required"), nil)
		return
	}

	userInfo, err := h.siteService.GetSiteUserInfo(ctx, siteID, userID, email)
	if err != nil {
		if errors.Is(err, common.ErrUserNotFound) {
			api.HandleError(ctx, common.ErrUserNotFound, nil)
			return
		}
		log.Error(ctx, "get user info failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, userInfo)
}

// UserChangeStatus godoc
// @Summary Change user status
// @Schemes
// @Description Change a user's status (activate, forbidden, delete) within a site. If deleted, the user is truly removed.
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.UserChangeStatusRequest true "Parameters for changing user status"
// @Success 200 {object} api.Response "Status changed successfully"
// @Failure 400 {object} api.Response "Bad request"
// @Failure 401 {object} api.Response "Unauthorized"
// @Failure 404 {object} api.Response "User not found"
// @Failure 500 {object} api.Response "Internal server error"
// @Router /api/site/user/change/status [post]
func (h *SiteHandler) UserChangeStatus(ctx *gin.Context) {
	// 1. Validate user authentication
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	// 2. Parse request body into the request struct
	var req api.UserChangeStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// 3. Validate status value: allowed are 2=activate, 3=forbidden, 127=delete
	if req.Status != model.UserStatusActive && req.Status != model.UserStatusDisabled && req.Status != model.UserStatusDeleted {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("invalid status value"), nil)
	}

	// 4. Call service to change user status
	err := h.siteService.ChangeUserStatus(ctx, req.SiteID, req.Email, req.Status)
	if err != nil {
		log.Error(ctx, "change user status failed: "+err.Error())
		return
	}

	// 5. Return success response
	api.HandleSuccess(ctx, nil)
}

// UserResetPassword godoc
// @Summary Reset site user password
// @Schemes
// @Description Generate a new password for a site user and return it to the creator.
// @Tags site
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.UserResetPasswordRequest true "Parameters for resetting user password"
// @Success 200 {object} api.UserResetPasswordResponseData "Password reset successfully"
// @Failure 400 {object} api.Response "Bad request"
// @Failure 401 {object} api.Response "Unauthorized"
// @Failure 404 {object} api.Response "User not found"
// @Failure 500 {object} api.Response "Internal server error"
// @Router /api/site/user/reset/password [post]
func (h *SiteHandler) UserResetPassword(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	var req api.UserResetPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	result, err := h.siteService.ResetUserPassword(ctx, req.SiteID, req.Email)
	if err != nil {
		if errors.Is(err, common.ErrUserNotFound) {
			api.HandleError(ctx, common.ErrUserNotFound, nil)
			return
		}
		log.Error(ctx, "reset user password failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, result)
}
