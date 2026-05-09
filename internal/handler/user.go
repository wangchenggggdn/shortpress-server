package handler

import (
	"fmt"
	"net/http"

	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/middleware"
	"shortpress-server/internal/service"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related requests
type UserHandler struct {
	*Handler
	userService service.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	handler *Handler,
	userService service.UserService,
) *UserHandler {
	return &UserHandler{
		Handler:     handler,
		userService: userService,
	}
}

// Register godoc
// @Summary Register a new user
// @Schemes
// @Description Register a new user with email and password
// @Tags user
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param request body api.UserRegisterRequest true "Registration parameters"
// @Success 200 {object} api.RegisterResponseData
// @Router /api/user/register [post]
func (h *UserHandler) Register(ctx *gin.Context) {
	var req api.UserRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// Get site ID from context set by middleware
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteID is required"), nil)
		return
	}
	log.AddNotice(ctx, "email", req.Email)

	token, err := h.userService.RegisterUser(ctx, &req, siteID)
	if err != nil {
		log.Error(ctx, "user registration failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	// Get app version from header
	appVersion := ctx.GetHeader("x-app-version")
	if appVersion == "" {
		appVersion = ctx.GetHeader("X-App-Version")
	}

	api.HandleSuccess(ctx, api.RegisterResponseData{
		Token:  token,
		SiteID: siteID,
		Ver:    appVersion,
	})
}

// Login godoc
// @Summary User login
// @Schemes
// @Description User login with email and password
// @Tags user
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param request body api.UserLoginRequest true "Login parameters"
// @Success 200 {object} api.UserLoginResponse "Returns access token"
// @Router /api/user/login [post]
func (h *UserHandler) Login(ctx *gin.Context) {
	var req api.UserLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// Get site ID from context set by middleware
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteID is required"), nil)
		return
	}

	token, ver, err := h.userService.LoginUser(ctx, &req, siteID)
	if err != nil {
		log.Error(ctx, "user login failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	resp := api.UserLoginResponse{
		Response: api.Response{Code: 0, Info: "success"},
		Data: api.UserLoginData{
			AccessToken: token,
			Ver:         ver,
		},
	}
	ctx.JSON(http.StatusOK, resp)
}

// LoginByOauth2 godoc
// @Summary User login with oauth2
// @Schemes
// @Description User login with oauth2
// @Tags user
// @Accept json
// @Produce json
// @Param X-Site-Id header string true "Site ID"
// @Param request body api.UserLoginByAuthRequest true "Login parameters"
// @Success 200 {object} api.UserLoginResponse "Returns access token"
// @Router /api/user/login/oauth2 [post]
func (h *UserHandler) LoginByOauth2(ctx *gin.Context) {
	var req api.UserLoginByAuthRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	// Get site ID from context set by middleware
	siteID := middleware.GetSiteID(ctx)
	if siteID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, fmt.Errorf("siteID is required"), nil)
		return
	}

	token, ver, err := h.userService.LoginByOAuth2(ctx, &req, siteID)
	if err != nil {
		log.Error(ctx, "user login failed: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	resp := api.UserLoginResponse{
		Response: api.Response{Code: 0, Info: "success"},
		Data: api.UserLoginData{
			AccessToken: token,
			Ver:         ver,
		},
	}
	ctx.JSON(http.StatusOK, resp)
}

// GetProfile godoc
// @Summary Get user profile
// @Schemes
// @Description Get user profile information
// @Tags user
// @Accept json
// @Produce json
// @Security Bearer
// @Param X-Site-Id header string true "Site ID"
// @Param Authorization header string true "Bearer user token"
// @Success 200 {object} api.UserProfileResponse "Returns user profile"
// @Router /api/user/profile [get]
func (h *UserHandler) GetProfile(ctx *gin.Context) {
	userID := ctx.GetString("user_id")
	if userID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	profile, err := h.userService.GetUserProfile(ctx, userID)
	if err != nil {
		log.Error(ctx, "failed to get user profile: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, profile)
}

// ProfileModify godoc
// @Summary Modify user profile
// @Schemes
// @Description Modify user profile information (nickname and autoUnlock)
// @Tags user
// @Accept json
// @Produce json
// @Security Bearer
// @Param X-Site-Id header string true "Site ID"
// @Param Authorization header string true "Bearer user token"
// @Param request body api.UserProfileModifyRequest true "Profile modification parameters"
// @Success 200 {object} api.UserProfileResponse "Returns updated user profile"
// @Router /api/user/profile-modify [post]
func (h *UserHandler) ProfileModify(ctx *gin.Context) {
	var req api.UserProfileModifyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		api.HandleErrorWithHttpCode(ctx, http.StatusBadRequest, err, nil)
		return
	}

	userID := ctx.GetString("user_id")
	if userID == "" {
		api.HandleErrorWithHttpCode(ctx, http.StatusUnauthorized, common.ErrUnauthorized, nil)
		return
	}

	err := h.userService.ModifyUserProfile(ctx, userID, &req)
	if err != nil {
		log.Error(ctx, "failed to modify user profile: "+err.Error())
		api.HandleError(ctx, err, nil)
		return
	}

	api.HandleSuccess(ctx, nil)
}
