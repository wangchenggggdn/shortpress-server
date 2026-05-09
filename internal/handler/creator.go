package handler

import (
	"net/http"

	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/service"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type CreatorHandler struct {
	*Handler
	creatorService service.CreatorService
	conf           *viper.Viper
}

func NewCreatorHandler(
	handler *Handler,
	creatorService service.CreatorService,
	conf *viper.Viper,
) *CreatorHandler {
	return &CreatorHandler{
		Handler:        handler,
		creatorService: creatorService,
		conf:           conf,
	}
}

func (h *CreatorHandler) GetCreator(ctx *gin.Context) {

}

// Register godoc
// @Summary Register a creator
// @Schemes
// @Description Currently only supports email registration
// @Tags creator
// @Accept json
// @Produce json
// @Param request body api.RegisterCreatorRequest true "params"
// @Success 200 {object} api.Response
// @Router /api/creator/register [post]
func (h *CreatorHandler) CreatorRegister(ctx *gin.Context) {
	var req api.RegisterCreatorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}
	err := h.creatorService.RegisterCreator(ctx, &req)
	if err != nil {
		log.Error(ctx, "creator register failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}

// ReseCreatorPwd godoc
// @Summary Reset creator password
// @Schemes
// @Description Reset creator password via email.
// @Tags creator
// @Accept json
// @Produce json
// @Param request body api.ResetCreatorPwdRequest true "Reset password parameters"
// @Success 200 {object} api.Response
// @Router /api/creator/resetpwd [post]
func (h *CreatorHandler) CreatorRestPwd(ctx *gin.Context) {
	api.HandleSuccess(ctx, "will be implemented")
}

// CreatorLogin godoc
// @Summary Creator login
// @Schemes
// @Description Creator login interface
// @Tags creator
// @Accept json
// @Produce json
// @Param request body api.CreatorLoginRequest true "Creator login parameters"
// @Success 200 {object} api.Response "Returns login response containing accessToken"
// @Router /api/creator/login [post]
func (h *CreatorHandler) CreatorLogin(ctx *gin.Context) {
	var req api.CreatorLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	token, err := h.creatorService.LoginCreator(ctx, &req)
	if err != nil {
		log.Error(ctx, "creator login failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}
	api.HandleSuccess(ctx, &api.CreatorLoginData{
		AccessToken: token,
	})
}

// CreatorProfile godoc
// @Summary Get creator's profile information
// @Schemes
// @Description Get detailed creator profile information based on creator ID, including nickname, avatar URL, creation time, etc.
// @Tags creator
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Success 200 {object} api.CreatorProfileResponse "Returns creator profile information"
// @Router /api/creator/profile [get]
func (h *CreatorHandler) CreatorProfile(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}
	creator, err := h.creatorService.GetCreator(ctx, creatorID)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}
	if creator == nil {
		log.Error(ctx, "creator not found, creator_id: "+creatorID)
		api.HandleError(ctx, common.ErrCreatorNotFound, nil)
		return
	}

	profiles, err := h.creatorService.GetCreatorProfile(ctx, creatorID)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}
	if profiles == nil {
		log.Error(ctx, "creator profile not found, creator_id: "+creatorID)
		api.HandleError(ctx, common.ErrCreatorNotFound, nil)
		return
	}

	api.HandleSuccess(ctx, api.CreatorProfileData{
		Email:            creator.Email,
		Nickname:         profiles.Nickname,
		AvatarURL:        profiles.AvatarURL,
		Guides:           h.creatorService.GetGuides(ctx, creatorID),
		DefultSiteDomain: h.conf.GetString("domain.hosting"),
		CreatedAt:        profiles.CreatedAt,
		UpdatedAt:        profiles.UpdatedAt,
	})
}

// UploadImg godoc
// @Summary Upload image
// @Schemes
// @Description Creator image upload interface, supports jpg, png, gif and other common image formats
// @Tags creator
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param file formData file true "Image file to upload"
// @Success 200 {object} api.Response{data=string} "Returns the path of the successfully uploaded image"
// @Router /api/creator/upload-file [post]
func (h *CreatorHandler) UploadFile(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}
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
	fileUrl, err := h.creatorService.UploadImg(ctx, creatorID, file)
	if err != nil {
		api.HandleError(ctx, err, nil)
		return
	}
	api.HandleSuccess(ctx, fileUrl)
}

// Stats godoc
// @Summary Get creator's total videos, playlists, and sites count
// @Schemes
// @Description Get creator's total videos, playlists, and sites count
// @Tags creator
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Success 200 {object} api.CreatorStatsResponseData "Returns creator video statistics"
// @Router /api/creator/stats [get]
func (h *CreatorHandler) Stats(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}
	siteCount, playlistCount, videoCount := h.creatorService.GetStatsCount(ctx, creatorID)
	api.HandleSuccess(ctx, api.CreatorStatsResponseData{
		SiteCount:     siteCount,
		PlaylistCount: playlistCount,
		VideoCount:    videoCount,
	})
}

// Complete guides update
// UpdateGuides godoc
// @Summary Complete guides
// @Schemes
// @Description Complete guides
// @Tags creator
// @Accept json
// @Produce json
// @Security Bearer
// @Param Authorization header string true "Bearer user token"
// @Param request body api.CompleteGuidesRequest true "Creator login parameters"
// @Success 200 {object} api.Response "Returns successful result of completing guides"
// @Router /api/creator/complete-guides [post]
func (h *CreatorHandler) CompleteGuides(ctx *gin.Context) {
	creatorID := ctx.GetString("creator_id")
	if creatorID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}
	var req api.CompleteGuidesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}
	err := h.creatorService.CompleteGuides(ctx, creatorID, req.Guides)
	if err != nil {
		log.Error(ctx, "complete guides failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}
	api.HandleSuccess(ctx, nil)
}
