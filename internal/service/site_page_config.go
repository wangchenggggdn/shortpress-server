package service

import (
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/site"

	"github.com/gin-gonic/gin"
)

type SitePageConfigService interface {
	Create(ctx *gin.Context, siteID string, req *api.SitePageConfigCreateRequest) (uint, error)
	Update(ctx *gin.Context, siteID string, req *api.SitePageConfigUpdateRequest) (uint, error)
	GetBySiteIDAndType(ctx *gin.Context, siteID string, pageType string) (*model.SitePageConfig, error)
	ListBySiteID(ctx *gin.Context, siteID string) ([]*model.SitePageConfig, int64, error)
}

type sitePageConfigService struct {
	*Service
	sitePageConfigRepository site.SitePageConfigRepository
}

func NewSitePageConfigService(
	service *Service,
	sitePageConfigRepository site.SitePageConfigRepository,
) SitePageConfigService {
	return &sitePageConfigService{
		Service:                  service,
		sitePageConfigRepository: sitePageConfigRepository,
	}
}

func (s *sitePageConfigService) Create(ctx *gin.Context, siteID string, req *api.SitePageConfigCreateRequest) (uint, error) {
	// Check if config with same site_id and type already exists
	existing, err := s.sitePageConfigRepository.GetBySiteIDAndType(ctx, siteID, req.Type)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		// Update existing config instead of creating new one
		existing.Config = req.Config
		if err := s.sitePageConfigRepository.Update(ctx, existing); err != nil {
			return 0, err
		}
		return existing.ID, nil
	}

	// Create new config
	config := &model.SitePageConfig{
		SiteID: siteID,
		Type:   req.Type,
		Config: req.Config,
	}

	if err := s.sitePageConfigRepository.Create(ctx, config); err != nil {
		return 0, err
	}

	return config.ID, nil
}

func (s *sitePageConfigService) Update(ctx *gin.Context, siteID string, req *api.SitePageConfigUpdateRequest) (uint, error) {
	// Get existing config by site_id and type
	config, err := s.sitePageConfigRepository.GetBySiteIDAndType(ctx, siteID, req.Type)
	if err != nil {
		return 0, err
	}
	if config == nil {
		return 0, common.ErrSitePageConfigNotFound
	}

	// Update config data
	config.Config = req.Config

	if err := s.sitePageConfigRepository.Update(ctx, config); err != nil {
		return 0, err
	}

	return config.ID, nil
}

func (s *sitePageConfigService) GetBySiteIDAndType(ctx *gin.Context, siteID string, pageType string) (*model.SitePageConfig, error) {
	return s.sitePageConfigRepository.GetBySiteIDAndType(ctx, siteID, pageType)
}

func (s *sitePageConfigService) ListBySiteID(ctx *gin.Context, siteID string) ([]*model.SitePageConfig, int64, error) {
	configs, err := s.sitePageConfigRepository.ListBySiteID(ctx, siteID)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(configs))
	return configs, total, nil
}
