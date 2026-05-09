package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/creator"
	"shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/playlist"
	"shortpress-server/internal/repository/db/site"
	"shortpress-server/internal/repository/db/user"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/blacklist"
	"shortpress-server/pkg/log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type SiteService interface {
	GetSites(ctx *gin.Context, siteIds []string) ([]*model.Site, error)
	GetSiteAndSeo(ctx *gin.Context, siteId string) (*api.SiteInfo, error)
	GetSitesByCreator(ctx *gin.Context, creator string) ([]*model.Site, error)
	CreateSiteAndSeo(ctx *gin.Context, siteInfo *api.SiteInfo, creatorID string) (string, error)
	ModifySiteAndSeo(ctx *gin.Context, req *api.SiteInfo) error
	AddPlaylists(ctx *gin.Context, siteID string, playlistIDs []string) error
	RemovePlaylists(ctx *gin.Context, siteID string, playlistIDs []string) error
	GetSiteUsers(ctx *gin.Context, query *model.UserQuery, page, pageSize, sortType int) ([]*api.UserInfo, int64, error)
	ChangeUserStatus(ctx *gin.Context, siteID string, email string, status int8) error
	PathExists(ctx *gin.Context, path string) (bool, error)
}

func NewSiteService(
	service *Service,
	siteRepository site.SiteRepository,
	siteSeoRepository site.SiteSeoRepository,
	sitePlaylistRepository site.SitePlaylistsRepository,
	creatorSiteRepository creator.CreatorSiteRepository,
	playlistRepository playlist.PlaylistRepository,
	userRepository user.UserRepository,
	userCoinsRepository payment.UserCoinsRepository,
	siteBuilderDataRepository site.SiteBuilderDataRepository,
	sitePageTemplateRepository site.SitePageTemplateRepository,
	blacklistChecker *blacklist.Blacklist,

) SiteService {
	return &siteService{
		Service:                    service,
		siteRepository:             siteRepository,
		siteSeoRepository:          siteSeoRepository,
		sitePlaylistRepository:     sitePlaylistRepository,
		userRepository:             userRepository,
		creatorSiteRepository:      creatorSiteRepository,
		playlistRepository:         playlistRepository,
		userCoinsRepository:        userCoinsRepository,
		siteBuilderDataRepository:  siteBuilderDataRepository,
		sitePageTemplateRepository: sitePageTemplateRepository,
		blacklistChecker:           blacklistChecker,
	}
}

type siteService struct {
	*Service
	siteRepository             site.SiteRepository
	siteSeoRepository          site.SiteSeoRepository
	sitePlaylistRepository     site.SitePlaylistsRepository
	creatorSiteRepository      creator.CreatorSiteRepository
	playlistRepository         playlist.PlaylistRepository
	userRepository             user.UserRepository
	userCoinsRepository        payment.UserCoinsRepository
	siteBuilderDataRepository  site.SiteBuilderDataRepository
	sitePageTemplateRepository site.SitePageTemplateRepository
	blacklistChecker           *blacklist.Blacklist
}

// CreateSiteAndSeo calls the repository layer to create site record and SEO information
func (s *siteService) CreateSiteAndSeo(ctx *gin.Context, siteInfo *api.SiteInfo, creatorID string) (string, error) {

	count, err := s.creatorSiteRepository.Count(ctx, creatorID)
	if err != nil {
		return "", err
	}
	if count >= int64(1) {
		return "", common.ErrTooManySites
	}
	if siteInfo.TemplateID == nil {
		return "", common.ErrSiteTemplateRequired
	}

	// 检查站点名称是否包含敏感词
	log.AddNotice(ctx, "site_name", siteInfo.Name)
	log.AddNotice(ctx, "site_path", siteInfo.Path)
	if s.blacklistChecker != nil && s.blacklistChecker.IsEnabled() {
		if contains, word := s.blacklistChecker.ContainsSensitiveWord(siteInfo.Path); contains {
			log.AddNotice(ctx, "sensitive_word_detected", word)
			log.AddNotice(ctx, "original_text", siteInfo.Path)
			return "", common.ErrSensitiveWordDetected
		}
	}

	if siteInfo.Status == 0 {
		siteInfo.Status = int(model.SiteStatusPublished)
	}
	theme := 0
	if siteInfo.Theme != nil {
		theme = *siteInfo.Theme
	}
	site := &model.Site{
		SiteID:     uuid.NewString(),
		Domain:     siteInfo.Domain,
		Redirect:   siteInfo.Redirect,
		Path:       siteInfo.Path,
		Name:       siteInfo.Name,
		Logo:       siteInfo.Logo,
		Status:     siteInfo.Status, // Default status: published
		TemplateID: siteInfo.TemplateID,
		I18n:       siteInfo.SiteMultiLang,
		Theme:      theme,
	}

	// Create site and SEO records in a transaction
	err = s.tx.Transaction(ctx, func(ctx context.Context) error {

		if err = s.siteRepository.Create(ctx, site); err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				return common.ErrSiteAlreadyExist
			}
			return err
		}

		// Create site and creator association record
		if err = s.creatorSiteRepository.Create(ctx, &model.CreatorSites{
			CreatorID: creatorID,
			SiteID:    site.SiteID,
		}); err != nil {
			return err
		}
		// Create site SEO information
		seo := &model.SiteSeo{
			SiteID:      site.SiteID,
			Title:       siteInfo.Name,
			Description: siteInfo.Name,
			Keywords:    "",
		}
		if err = s.siteSeoRepository.Create(ctx, seo); err != nil {
			return err
		}

		// If templateId provided, initialize site_builder_data from template
		if siteInfo.TemplateID != nil && *siteInfo.TemplateID != "" {
			var tpl *model.SitePageTemplate
			tpl, err = s.sitePageTemplateRepository.GetByTemplateID(ctx, *siteInfo.TemplateID)
			if err != nil {
				return err
			}
			if tpl != nil {
				// Unmarshal and save via repository to set version 1
				var siteData interface{}
				if len(tpl.TemplateData) > 0 {
					if uerr := json.Unmarshal(tpl.TemplateData, &siteData); uerr != nil {
						return uerr
					}
				} else {
					siteData = map[string]interface{}{}
				}
				if _, err = s.siteBuilderDataRepository.UpdateBySiteID(ctx, site.SiteID, creatorID, siteData); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Error(ctx, "create site and seo failed, err: "+err.Error())
		return "", err
	}
	log.AddNotice(ctx, "site_name", site.Name)
	log.AddNotice(ctx, "site_path", site.Path)
	log.AddNotice(ctx, "domain", site.Domain)
	log.AddNotice(ctx, "site_id", site.SiteID)
	return site.SiteID, nil
}

// isValidDomain checks if a domain is valid by ensuring:
// 1. It's a legal host without path components
// 2. It doesn't contain http/https protocols
// 3. It's not an IP address
func (s *siteService) isValidDomain(domain string) error {
	if domain == "" {
		return nil
	}

	// Check if domain contains shortpress.com or http/https
	// if strings.Contains(domain, "shortpress.com") || strings.Contains(domain, "http") || strings.Contains(domain, "https") {
	// 	return common.ErrInvalidCustomDomain
	// }
	// TODO 待检查 是否和本地配置文件的domain 一致
	if strings.Contains(domain, "http") || strings.Contains(domain, "https") {
		return common.ErrInvalidCustomDomain
	}

	// Check if domain contains path separators
	if strings.Contains(domain, "/") || strings.Contains(domain, "\\") {
		return common.ErrInvalidCustomDomain
	}

	// Check if domain is an IP address
	if net.ParseIP(domain) != nil {
		return common.ErrInvalidCustomDomain
	}

	return nil
}

// ModifySiteAndSeo modifies site and SEO information
func (s *siteService) ModifySiteAndSeo(ctx *gin.Context, req *api.SiteInfo) error {
	if err := s.isValidDomain(req.Domain); err != nil {
		return err
	}

	// 1. Update site basic information
	site := &model.Site{
		SiteID:   req.SiteID,
		Domain:   req.Domain,
		Redirect: req.Redirect,
		Path:     req.Path,
		Name:     req.Name,
		Logo:     req.Logo,
		Status:   req.Status,
		I18n:     req.SiteMultiLang,
	}
	if req.GoogleAnalyticsID != nil {
		site.GoogleAnalyticsID = req.GoogleAnalyticsID
	} else {
		site.GoogleAnalyticsID = nil
	}
	if req.FacebookPixelID != nil {
		site.FacebookPixelID = req.FacebookPixelID
	} else {
		site.FacebookPixelID = nil
	}
	if req.ThinkingDataAppId != nil {
		site.ThinkingDataAppId = req.ThinkingDataAppId
	} else {
		site.ThinkingDataAppId = nil
	}
	if err := s.siteRepository.Update(ctx, site); err != nil {
		log.Error(ctx, "update site failed, err: "+err.Error())
		return err
	}
	if req.Theme != nil {
		if err := s.siteRepository.UpdateTheme(ctx, req.SiteID, *req.Theme); err != nil {
			log.Error(ctx, "update site theme failed, err: "+err.Error())
			return err
		}
	}

	// 2. Update SEO information
	if req.Seo != nil {
		seo := &model.SiteSeo{
			SiteID:      req.SiteID,
			Title:       req.Seo.Title,
			Description: req.Seo.Description,
			Keywords:    req.Seo.Keywords,
			I18n:        req.SeoMultiLang,
		}
		return s.siteSeoRepository.Save(ctx, seo)
	}
	return nil
}

// AddPlaylists adds playlists to a site
func (s *siteService) AddPlaylists(ctx *gin.Context, siteID string, playlistIDs []string) error {
	// check site exist
	site, err := s.siteRepository.GetBySiteID(ctx, siteID)
	if err != nil {
		return err
	}
	if site == nil {
		log.Warning(ctx, common.ErrSiteNotFound)
		return common.ErrSiteNotFound
	}

	//check playlist exist
	pCount, err := s.playlistRepository.CountByPlaylistIDs(ctx, playlistIDs)
	if err != nil {
		return err
	}
	if pCount != int64(len(playlistIDs)) {
		return common.ErrPlaylistNotFound
	}

	// Build batch creation data
	var sitePlaylists []*model.SitePlaylists
	for _, playlistID := range playlistIDs {
		sitePlaylists = append(sitePlaylists, &model.SitePlaylists{
			SiteID:     siteID,
			PlaylistID: playlistID,
			Status:     0,
		})
	}

	return s.sitePlaylistRepository.BatchCreate(ctx, sitePlaylists)
}

// DelPlaylists removes playlists from a site
func (s *siteService) RemovePlaylists(ctx *gin.Context, siteID string, playlistIDs []string) error {
	return s.sitePlaylistRepository.BatchDelete(ctx, siteID, playlistIDs)
}

func (s *siteService) GetSitesByCreator(ctx *gin.Context, creatorID string) ([]*model.Site, error) {
	v, err := s.creatorSiteRepository.GetByCreator(ctx, creatorID)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	var siteIDs []string
	for _, site := range v {
		siteIDs = append(siteIDs, site.SiteID)
	}
	sites, err := s.siteRepository.GetBySiteIDs(ctx, siteIDs)
	if err != nil {
		return nil, err
	}
	return sites, nil
}

func (s *siteService) GetSites(ctx *gin.Context, siteIds []string) ([]*model.Site, error) {
	sites, err := s.siteRepository.GetBySiteIDs(ctx, siteIds)
	if err != nil {
		return nil, err
	}

	// Populate template names for each site
	for _, site := range sites {
		if site.TemplateID != nil && *site.TemplateID != "" {
			template, err := s.sitePageTemplateRepository.GetByTemplateID(ctx, *site.TemplateID)
			if err == nil && template != nil {
				site.TemplateName = &template.Name
			}
		}
	}

	return sites, nil
}

func (s *siteService) GetSiteAndSeo(ctx *gin.Context, siteId string) (*api.SiteInfo, error) {
	site, err := s.siteRepository.GetBySiteID(ctx, siteId)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, common.ErrSiteNotFound
	}
	seo, err := s.siteSeoRepository.GetBySiteID(ctx, siteId)
	if err != nil {
		return nil, err
	}
	if seo == nil {
		return nil, fmt.Errorf("SEO information not found for site %s", siteId)
	}

	// Populate template name if template ID is present
	var templateName *string
	if site.TemplateID != nil && *site.TemplateID != "" {
		template, err := s.sitePageTemplateRepository.GetByTemplateID(ctx, *site.TemplateID)
		if err == nil && template != nil {
			templateName = &template.Name
		}
	}

	log.AddNotice(ctx, "site_name", site.Name)
	log.AddNotice(ctx, "site_path", site.Path)
	log.AddNotice(ctx, "domain", site.Domain)
	log.AddNotice(ctx, "site_id_think", site.ThinkingDataAppId)
	theme := site.Theme
	return &api.SiteInfo{
		SiteID: site.SiteID,
		OfficialDomain: &types.OfficialDomain{
			Subdomain: site.Path,
		},
		Domain:            site.Domain,
		Redirect:          site.Redirect,
		Logo:              site.Logo,
		Name:              site.Name,
		Path:              site.Path,
		TemplateID:        site.TemplateID,
		TemplateName:      templateName,
		Status:            site.Status,
		GoogleAnalyticsID: site.GoogleAnalyticsID,
		FacebookPixelID:   site.FacebookPixelID,
		ThinkingDataAppId: site.ThinkingDataAppId,
		Theme:             &theme,
		Seo: &api.SiteSeo{
			Title:       seo.Title,
			Description: seo.Description,
			Keywords:    seo.Keywords,
		},
		SiteMultiLang: site.I18n,
		SeoMultiLang:  seo.I18n,
	}, nil
}

func (s *siteService) GetSiteUsers(ctx *gin.Context, query *model.UserQuery, page, pageSize, sortType int) ([]*api.UserInfo, int64, error) {
	// 1. Validate the site exists
	site, err := s.siteRepository.GetBySiteID(ctx, query.SiteID)
	if err != nil {
		log.Error(ctx, "get by siteid faild, err:"+err.Error())
		return nil, 0, err
	}
	if site == nil {
		return nil, 0, common.ErrSiteNotFound
	}

	// 2. Query users from the user repository
	users, total, err := s.userRepository.GetSiteUsers(ctx, query, page, pageSize, sortType)
	if err != nil {
		log.Error(ctx, "querying site users failed, err:"+err.Error())
		return nil, 0, err
	}

	// 3. Convert to API response format

	userInfos := make([]*api.UserInfo, 0, len(users))
	log.AddNotice(ctx, "users_count", len(users))
	for _, user := range users {
		var preAt int64
		if user.PremiumExpiresAt != nil {
			preAt = user.PremiumExpiresAt.Unix()
		} else {
			preAt = 0
		}
		if user.Nickname == "" {
			user.Nickname = user.Email
		}
		userInfos = append(userInfos, &api.UserInfo{
			UserID:           user.UserID,
			Email:            user.Email,
			Nickname:         user.Nickname,
			Status:           user.Status,
			PremiumType:      user.PremiumType,
			PremiumExpiresAt: preAt,
			LastLoginAt: func() int64 {
				if user.LastLoginAt == nil {
					return 0
				}
				return user.LastLoginAt.Unix()
			}(),
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		})
	}

	return userInfos, total, nil
}

// ChangeUserStatus changes a user's status or deletes them if status is UserStatusDeleted
func (s *siteService) ChangeUserStatus(ctx *gin.Context, siteID string, email string, status int8) error {
	// 1. Find the user by email and siteID
	user, err := s.userRepository.GetByEmailAndSiteID(ctx, email, siteID)
	if err != nil {
		return err
	}

	if user == nil {
		log.Warning(ctx, common.ErrUserNotFound)
		return common.ErrUserNotFound
	}

	// 3. Otherwise just update the user's status
	user.Status = status
	return s.userRepository.Update(ctx, user)
}

func (s *siteService) PathExists(ctx *gin.Context, path string) (bool, error) {
	if path == "" {
		return false, nil
	}

	// Check if the path is already in use by another site
	exists, err := s.siteRepository.GetByPath(ctx, path)
	if err != nil {
		log.Error(ctx, err)
		return false, err
	}
	if exists != nil {
		log.Warning(ctx, common.ErrSiteAlreadyExist)
		return true, nil
	}
	return false, nil
}
