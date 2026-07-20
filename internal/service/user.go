package service

import (
	"context"
	"errors"
	"fmt"
	"shortpress-server/pkg/oauth2"
	"strings"
	"time"

	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/site"
	"shortpress-server/internal/repository/db/user"
	"shortpress-server/internal/service/analytics"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService defines the operations for user management
type UserService interface {
	RegisterUser(ctx *gin.Context, req *api.UserRegisterRequest, siteID string) (string, error)
	LoginUser(ctx *gin.Context, req *api.UserLoginRequest, siteID string) (string, string, error)
	LoginByOAuth2(ctx *gin.Context, req *api.UserLoginByAuthRequest, siteID string) (string, string, error)
	GetUserProfile(ctx *gin.Context, userID string) (*api.UserProfileData, error)
	ModifyUserProfile(ctx *gin.Context, userID string, req *api.UserProfileModifyRequest) error
	ChangePassword(ctx *gin.Context, userID string, newPassword string) error
	SyncMetaClick(ctx *gin.Context, userID string, req *api.MetaClickSyncRequest) error
	SyncPixel(ctx *gin.Context, userID string, req *api.PixelSyncRequest) (*api.PixelSyncResponseData, error)
}

type userService struct {
	*Service
	userRepository         user.UserRepository
	userProfileRepository  user.UserProfileRepository
	userAuthRepository     user.UserAuthRepository
	siteRepository         site.SiteRepository
	oauth2Client           oauth2.Client
	userCoinsRepository    payment.UserCoinsRepository
	paymentTransactionRepo payment.PaymentTransactionRepository
}

// NewUserService creates a new user service
func NewUserService(
	service *Service,
	userRepository user.UserRepository,
	userProfileRepository user.UserProfileRepository,
	userAuthRepository user.UserAuthRepository,
	siteRepository site.SiteRepository,
	userCoinsRepository payment.UserCoinsRepository,
	paymentTransactionRepo payment.PaymentTransactionRepository,
) UserService {
	return &userService{
		Service:                service,
		userRepository:         userRepository,
		userProfileRepository:  userProfileRepository,
		userAuthRepository:     userAuthRepository,
		siteRepository:         siteRepository,
		oauth2Client:           oauth2.New(oauth2.TypeGoogle, oauth2.TypeTikTok),
		userCoinsRepository:    userCoinsRepository,
		paymentTransactionRepo: paymentTransactionRepo,
	}
}

// RegisterUser registers a new user
func (s *userService) RegisterUser(ctx *gin.Context, req *api.UserRegisterRequest, siteID string) (string, error) {
	// Validate that the site exists
	site, err := s.siteRepository.GetBySiteID(ctx, siteID)
	if err != nil {
		return "", err
	}
	if site == nil {
		return "", common.ErrSiteNotFound
	}

	// Check if email already exists for this site
	existingUser, err := s.userRepository.GetByEmailAndSiteID(ctx, req.Email, siteID)
	if err != nil {
		return "", err
	}
	if existingUser != nil {
		return "", common.ErrEmailAlreadyRegistered
	}

	// Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	if req.Nickname == "" {
		req.Nickname = req.Email
	}
	templateID := ""
	if site.TemplateID != nil {
		templateID = *site.TemplateID
	}
	return s.register(ctx, &registerArgs{
		AuthType:   model.AuthTypeEmail,
		Email:      req.Email,
		SiteID:     siteID,
		TemplateID: templateID,
		Nickname:   req.Nickname,
		AvatarURL:  "",
		Password:   string(hash),
	})
}

// LoginUser authenticates a user and returns a JWT token and app version
func (s *userService) LoginUser(ctx *gin.Context, req *api.UserLoginRequest, siteID string) (string, string, error) {
	// Find user by email and site ID
	user, err := s.userRepository.GetByEmailAndSiteID(ctx, req.Email, siteID)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		log.Error(ctx, "user not found: "+req.Email+" for site: "+siteID)
		return "", "", common.ErrInvalidAccountOrPassword
	}
	token, err := s.login(ctx, &loginArgs{
		AuthType: model.AuthTypeEmail,
		Email:    req.Email,
		Password: req.Password,
		User:     user,
	})
	return token, user.Ver, err
}

func (s *userService) LoginByOAuth2(ctx *gin.Context, req *api.UserLoginByAuthRequest, siteID string) (string, string, error) {
	// Validate that the site exists
	site, err := s.siteRepository.GetBySiteID(ctx, siteID)
	if err != nil {
		return "", "", err
	}
	if site == nil {
		return "", "", common.ErrSiteNotFound
	}
	user, err := s.oauth2Client.Authenticate(ctx, &oauth2.AuthArgs{
		Type:       oauth2.OauthType(req.SrcType),
		Token:      req.Token,
		Code:       req.Code,
		Credential: req.Credential,
	})
	if err != nil {
		return "", "", err
	}

	if user.ID == "" {
		log.Error(ctx, "user not found for site: "+siteID)
		return "", "", common.ErrInvalidAccountOrPassword
	}
	if user.Username == "" {
		user.Username = user.Email
		if user.Username == "" {
			user.Username = user.ID
		}
	}

	authType := s.getAuthType(oauth2.OauthType(req.SrcType))
	// 为三方id添加前缀，防止与其他认证方式的id冲突
	identifier := fmt.Sprintf("%d:%s", authType, user.ID)
	// 判断用户是否已注册
	existingUser, err := s.userRepository.GetByIdentifierAndSiteID(ctx, identifier, siteID)
	if err != nil {
		return "", "", err
	}

	// 旧版google登录使用邮箱作为唯一标识，此处需要兼容
	if existingUser == nil && authType == model.AuthTypeGoogle && user.Email != "" {
		existingUser, err = s.userRepository.GetByEmailAndSiteID(ctx, user.Email, siteID)
		if err != nil {
			return "", "", err
		}

		// 将旧版google用户的ID设置为唯一标识
		if existingUser != nil {
			existingUser.Identifier = identifier
			existingUser.Email = ""
			if err = s.userRepository.UpdateIdentifierAndEmail(ctx, existingUser); err != nil {
				// 失败不影响登录流程
				log.Error(ctx, "update user failed: "+err.Error())
			}
		}
	}

	// 如果用户已注册，直接登录
	if existingUser != nil {
		token, err := s.login(ctx, &loginArgs{
			AuthType:   authType,
			Identifier: identifier,
			Password:   "",
			User:       existingUser,
		})
		return token, existingUser.Ver, err
	} else {
		// 用户未注册，执行注册流程，增加默认赠送的金币
		templateID := ""
		if site.TemplateID != nil {
			templateID = *site.TemplateID
		}
		token, err := s.register(ctx, &registerArgs{
			AuthType:   authType,
			Identifier: identifier,
			SiteID:     siteID,
			TemplateID: templateID,
			Email:      user.Email,
			Nickname:   user.Username,
			AvatarURL:  user.Avatar,
			Password:   "",
		})
		// For new registration, return the version from header
		appVersion := ctx.GetHeader("x-app-version")
		if appVersion == "" {
			appVersion = ctx.GetHeader("X-App-Version")
		}
		return token, appVersion, err
	}

}

func (s *userService) getAuthType(srcType oauth2.OauthType) int8 {
	switch srcType {
	case oauth2.TypeGoogle:
		return model.AuthTypeGoogle
	case oauth2.TypeTikTok:
		return model.AuthTypeTikTok
	default:
		return model.AuthTypeEmail
	}
}

type registerArgs struct {
	AuthType   int8
	Email      string
	Identifier string
	SiteID     string
	TemplateID string
	Nickname   string
	AvatarURL  string
	Password   string
}

func (s *userService) register(ctx *gin.Context, args *registerArgs) (string, error) {
	userID := uuid.NewString()
	log.AddNotice(ctx, "new_user_id", userID)

	// Get app version from header
	appVersion := ctx.GetHeader("x-app-version")
	if appVersion == "" {
		appVersion = ctx.GetHeader("X-App-Version")
	}
	log.AddNotice(ctx, "app_version", appVersion)

	// 从 referer 中提取 UTM 参数
	//utmSource, utmCampaign := extractUTMParams(ctx)

	// 根据 UTM 参数判断 referer 值
	balance := 0
	referer := ctx.Request.Header.Get("Utm-source0")
	page := ctx.Request.Header.Get("Page0")

	if args.TemplateID != "" && strings.Contains(args.TemplateID, "sora") {

		if referer != "" {
			if strings.Contains(referer, "m1") {
				referer = "m1"
				pageReward := map[string]int{
					"textToVideobasic":    30,
					"textToVideopro":      45,
					"imageToVideobasic":   30,
					"imageToVideoquality": 40,
					"imageToVideopremium": 50,
					"imageToVideopro":     60,
					"imageToVideoelite":   90,
					"imageToVideoultra":   120,
					"textToImagepro":      15,
					"textToImagepremium":  10,
					"textToImagebasic":    5,
					"textToImagequality":  8,
					"template":            50,
					"create":              10,
				}
				if reward, ok := pageReward[page]; ok {
					balance = reward
				} else {
					balance = 10
				}
			} else if strings.Contains(referer, "f1") {
				referer = "f1"
				balance = 0
			} else if strings.Contains(referer, "a1") {
				referer = "a1"
				balance = 0
			} else if strings.Contains(referer, "v1") {
				referer = "v1"
				balance = 0
			}
		}
	}
	// 针对短剧站点 (template_id 包含 short 或 drama) 设置默认赠送 200 点
	if args.TemplateID != "" && (strings.Contains(args.TemplateID, "short") || strings.Contains(args.TemplateID, "drama")) {
		balance = 0
	}

	// organic 或其它情况保持空字符串

	log.AddNotice(ctx, "balance", balance)
	log.AddNotice(ctx, "referer", referer)
	log.AddNotice(ctx, "page", page)

	pixelID, err := s.resolveRequestPixelID(ctx, args.SiteID)
	if err != nil {
		return "", err
	}
	log.AddNotice(ctx, "pixel_id", pixelID)

	platform := strings.ToLower(strings.TrimSpace(ctx.GetHeader("X-Client-Type")))
	log.AddNotice(ctx, "platform", platform)

	// Create user model
	now := time.Now()
	user := &model.User{
		UserID:      userID,
		Email:       args.Email,
		Identifier:  args.Identifier,
		SiteID:      args.SiteID,
		Status:      model.UserStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Referer:     referer,
		PixelID:     pixelID,
		Platform:    platform,
		LastLoginAt: &now,
		Ver:         appVersion,
	}

	// Create user profile
	unLockStatus := true
	profile := &model.UserProfile{
		UserID:     userID,
		Nickname:   args.Nickname,
		AvatarURL:  args.AvatarURL,
		AutoUnlock: &unLockStatus,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Create user authentication
	identifier := args.Identifier
	if args.AuthType != model.AuthTypeEmail {
		identifier = args.Email
	}
	userAuth := &model.UserAuth{
		UserID:       userID,
		Type:         args.AuthType,
		Identifier:   identifier,
		PasswordHash: args.Password,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Use transaction to ensure data consistency
	err = s.tx.Transaction(ctx, func(ctx context.Context) error {
		if err := s.userRepository.Create(ctx, user); err != nil {
			return err
		}

		if err := s.userProfileRepository.Create(ctx, profile); err != nil {
			return err
		}

		if err := s.userAuthRepository.Create(ctx, userAuth); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	// 注册成功，给新用户增加金币
	err = s.userCoinsRepository.Update(ctx, &model.UserCoins{
		UserID:      userID,
		SiteID:      args.SiteID,
		Balance:     balance, // 默认赠送35金币
		TotalEarned: balance,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		log.Error(ctx, "First registration comes with a bonus of gold coins:"+err.Error())
	}

	token, err := s.jwt.GenClientUserToken(userID, time.Now().Add(time.Hour*24*30)) //gust token 30天有效期
	if err != nil {
		log.Error(ctx, "failed to generate JWT token:"+err.Error())
		return "", err
	}
	return token, nil
}

type loginArgs struct {
	AuthType   int8
	Email      string
	Identifier string
	Password   string
	*model.User
}

func (s *userService) login(ctx *gin.Context, args *loginArgs) (string, error) {
	// Get user auth
	userAuth, err := s.userAuthRepository.GetByUserIDAndType(ctx, args.UserID, args.AuthType)
	if err != nil {
		return "", err
	}
	if userAuth == nil {
		return "", common.ErrInvalidAccountOrPassword
	}

	// if user.Status  == model.UserStatusInactive {
	//  return "", common.ErrUserNotActive
	// } else if user.Status == model.UserStatusDisabled ||  user.Status == model.UserStatusDeleted {
	//  return "", common.ErrUserBanned
	// }

	switch args.Status {
	case model.UserStatusInactive:
		return "", common.ErrUserNotActive
	case model.UserStatusDisabled, model.UserStatusDeleted: // Combine cases
		return "", common.ErrUserBanned
	}

	// Verify password (skip for OAuth2 login)
	if args.AuthType == model.AuthTypeEmail {
		if err := bcrypt.CompareHashAndPassword([]byte(userAuth.PasswordHash), []byte(args.Password)); err != nil {
			return "", common.ErrInvalidAccountOrPassword
		}
	}

	// Update last login time
	if err := s.userRepository.UpdateLoginTime(ctx, args.UserID); err != nil {
		log.Warning(ctx, "failed to update last login time: "+err.Error()) // Log the error but continue
		// Continue even if this fails
	}

	// Generate JWT token (7 days validity)
	log.AddNotice(ctx, "user_login", args.UserID)
	log.AddNotice(ctx, "user_email", args.Email)
	log.AddNotice(ctx, "user_identifier", args.Identifier)

	token, err := s.jwt.GenClientUserToken(args.UserID, time.Now().Add(time.Hour*24*7))
	if err != nil {
		return "", err
	}

	return token, nil
}

// GetUserProfile returns the profile information for a user
func (s *userService) GetUserProfile(ctx *gin.Context, userID string) (*api.UserProfileData, error) {
	user, err := s.userRepository.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, common.ErrUserNotFound
	}

	profile, err := s.userProfileRepository.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, common.ErrUserProfileNotFound
	}

	userAuths, err := s.userAuthRepository.GetAllByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(userAuths) == 0 {
		return nil, common.ErrUserAuthNotFound
	}

	// Check if user has made any purchases
	hasPurchased, err := s.paymentTransactionRepo.HasUserPurchased(ctx, userID, user.SiteID)
	if err != nil {
		log.Warning(ctx, "failed to check user purchase status: "+err.Error())
		hasPurchased = false
	}

	var expAt int64
	if user.PremiumExpiresAt != nil {
		expAt = user.PremiumExpiresAt.Unix()
	}
	log.AddNotice(ctx, "user_email", user.Email)
	return &api.UserProfileData{
		UserID:           user.UserID,
		Email:            user.Email,
		Referer:          user.Referer,
		Nickname:         profile.Nickname,
		AvatarURL:        profile.AvatarURL,
		PremiumType:      user.PremiumType,
		OnetimeSub:       user.OnetimeSub,
		AutoUnlock:       *profile.AutoUnlock,
		PremiumExpiresAt: expAt,
		Status:           user.Status,
		LoginType:        userAuths[0].Type,
		HasPurchased:     hasPurchased,
		PixelID:          user.PixelID,
		Ver:              user.Ver,
	}, nil
}

func (s *userService) resolveRequestPixelID(ctx *gin.Context, siteID string) (string, error) {
	pixelID := strings.TrimSpace(ctx.GetHeader("First-Pixel-Id"))
	if pixelID == "" {
		pixelID = strings.TrimSpace(ctx.GetHeader("Pixel-Id"))
	}
	if pixelID != "" {
		return pixelID, nil
	}
	return s.defaultPixelIDForSite(ctx, siteID)
}

func (s *userService) defaultPixelIDForSite(ctx context.Context, siteID string) (string, error) {
	siteRow, err := s.siteRepository.GetBySiteID(ctx, siteID)
	if err != nil {
		return "", err
	}
	if siteRow == nil {
		return "", common.ErrSiteNotFound
	}
	if siteRow.FacebookPixelID == nil {
		return "", nil
	}
	return strings.TrimSpace(*siteRow.FacebookPixelID), nil
}

// ChangePassword updates the password for an email-authenticated user.
func (s *userService) ChangePassword(ctx *gin.Context, userID string, newPassword string) error {
	user, err := s.userRepository.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return common.ErrUserNotFound
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	userAuth, err := s.userAuthRepository.GetByUserIDAndType(ctx, userID, model.AuthTypeEmail)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if userAuth != nil {
		return s.userAuthRepository.UpdatePassword(ctx, userID, string(hash))
	}

	identifier := user.Email
	if identifier == "" {
		return common.ErrUserAuthNotFound
	}

	userAuth = &model.UserAuth{
		UserID:       userID,
		Type:         model.AuthTypeEmail,
		Identifier:   identifier,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return s.userAuthRepository.Create(ctx, userAuth)
}

// ModifyUserProfile modifies the profile information for a user
func (s *userService) ModifyUserProfile(ctx *gin.Context, userID string, req *api.UserProfileModifyRequest) error {

	// Find user by ID
	user, err := s.userRepository.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return common.ErrUserNotFound
	}

	// Find user profile by user ID
	profile := &model.UserProfile{
		UserID:     userID,
		AutoUnlock: req.AutoUnlock,
		UpdatedAt:  time.Now(),
	}
	if req.Nickname != "" {
		profile.Nickname = req.Nickname
	}

	// Use transaction to ensure data consistency
	err = s.tx.Transaction(ctx, func(ctx context.Context) error {
		if err := s.userProfileRepository.Update(ctx, profile); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// SyncMetaClick persists Meta fbc/fbp/fbclid on the user row for delayed conversions.
func (s *userService) SyncMetaClick(ctx *gin.Context, userID string, req *api.MetaClickSyncRequest) error {
	if req == nil {
		return nil
	}
	payload := &api.MetaClickPayload{
		Fbc:            req.Fbc,
		Fbp:            req.Fbp,
		Fbclid:         req.Fbclid,
		EventSourceURL: req.EventSourceURL,
	}
	return analytics.PersistUserMetaClick(ctx, s.userRepository, userID, payload)
}

// SyncPixel persists the effective Facebook Pixel ID on the user row.
func (s *userService) SyncPixel(ctx *gin.Context, userID string, req *api.PixelSyncRequest) (*api.PixelSyncResponseData, error) {
	user, err := s.userRepository.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, common.ErrUserNotFound
	}

	existing := strings.TrimSpace(user.PixelID)
	if existing != "" {
		return &api.PixelSyncResponseData{PixelID: existing, Source: "database"}, nil
	}

	pixelID := ""
	if req != nil {
		pixelID = strings.TrimSpace(req.PixelID)
	}
	if pixelID == "" {
		pixelID, err = s.resolveRequestPixelID(ctx, user.SiteID)
		if err != nil {
			return nil, err
		}
	}
	if pixelID == "" {
		return &api.PixelSyncResponseData{}, nil
	}

	if err := s.userRepository.UpdatePixelID(ctx, userID, pixelID); err != nil {
		return nil, err
	}
	return &api.PixelSyncResponseData{PixelID: pixelID, Source: "client_written"}, nil
}
