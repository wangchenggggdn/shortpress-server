package service

import (
	"context"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/ads"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

// AdsService defines the interface for advertisement related operations
type AdsService interface {
	ListAdUnits(ctx context.Context, siteID string) ([]*api.AdUnit, error)
	CreateAdUnit(ctx context.Context, req api.AdUnit) (string, error)
	ModifyAdUnit(ctx context.Context, req api.AdUnit) error
	GetAdUnitConf(ctx context.Context, siteID string, location string) (*api.AdUnitConf, error)
}

type adsService struct {
	*Service
	adRepository         ads.AdRepository
	adLocationRepository ads.AdLocationRepository
}

// NewAdsService creates a new instance of AdsService
func NewAdsService(
	service *Service,
	adRepository ads.AdRepository,
	adLocationRepository ads.AdLocationRepository,
) AdsService {
	return &adsService{
		Service:              service,
		adRepository:         adRepository,
		adLocationRepository: adLocationRepository,
	}
}

// ListAdUnits returns a list of ad units for a specific site
func (s *adsService) ListAdUnits(ctx context.Context, siteID string) ([]*api.AdUnit, error) {

	// Get all ads for the site
	ads, err := s.adLocationRepository.List(ctx, siteID)
	if err != nil {
		return nil, err
	}

	result := make([]*api.AdUnit, 0, len(ads))
	for _, ad := range ads {

		// A safer approach with type checking
		var frequency int
		switch v := ad.ShowStg["frequency"].(type) {
		case float64:
			frequency = int(v)
		case int:
			frequency = v
		}

		result = append(result, &api.AdUnit{
			SiteID:    ad.SiteID,
			AdID:      ad.AdID,
			Name:      ad.Name,
			AdNetwork: ad.NetWork,
			Page:      ad.Location,
			Format:    int(ad.Format),
			Status:    int(ad.Status),
			Frequency: frequency,
			ClientID:  ad.Conf["clientID"].(string),
			UnitID:    ad.Conf["unitID"].(string),
		})
	}
	return result, nil
}

// CreateAdUnit creates a new ad unit with location
func (s *adsService) CreateAdUnit(ctx context.Context, req api.AdUnit) (string, error) {
	// Generate a new UUID for the ad
	adID := uuid.NewString()

	// Create configuration map
	conf := model.AdConfig{
		"clientID": req.ClientID,
		"unitID":   req.UnitID,
	}

	// Create ad record
	ad := &model.Ad{
		AdID:      adID,
		Name:      adID,
		Format:    int8(req.Format),
		AdNetwork: req.AdNetwork,
		Conf:      conf,
		Status:    model.AdStatusEnabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create ad location record
	adLocation := &model.AdLocation{
		AdID:     adID,
		SiteID:   req.SiteID,
		Location: req.Page,
		Name:     req.Name,
		ShowStg: model.ShowStrategy{
			"frequency": req.Frequency,
		},
		Status:    model.AdStatusEnabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Use transaction to ensure both records are created or none
	err := s.tx.Transaction(ctx, func(ctx context.Context) error {
		// Create ad record
		if err := s.adRepository.Create(ctx, ad); err != nil {
			return err
		}

		// Create ad location record
		if err := s.adLocationRepository.Create(ctx, adLocation); err != nil {
			// Check if the error is due to a record already existing
			if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
				return common.ErrAdNameAlreadyExist
			}
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return adID, nil
}

// ModifyAdUnit modifies an existing ad unit
func (s *adsService) ModifyAdUnit(ctx context.Context, req api.AdUnit) error {
	if req.AdID == "" {
		return common.ErrBadRequest
	}

	// Check if ad exists
	ad, err := s.adRepository.GetByAdID(ctx, req.AdID)
	if err != nil {
		return err
	}
	if ad == nil {
		return common.ErrNotFound
	}

	// Update configuration map
	ad.Format = int8(req.Format)
	ad.AdNetwork = req.AdNetwork
	ad.Conf = model.AdConfig{
		"clientID": req.ClientID,
		"unitID":   req.UnitID,
	}

	// Get ad location
	adLocation, err := s.adLocationRepository.GetByAdIDAndLocation(ctx, req.AdID, req.Page)
	if err != nil {
		return err
	}
	if adLocation == nil {
		return common.ErrNotFound
	}

	// Update ad location
	adLocation.Name = req.Name
	adLocation.ShowStg = model.ShowStrategy{
		"frequency": req.Frequency,
	}
	adLocation.Status = int8(req.Status)
	adLocation.UpdatedAt = time.Now()

	// Use transaction to ensure both records are updated or none
	return s.tx.Transaction(ctx, func(ctx context.Context) error {
		// Update ad record
		if err := s.adRepository.Update(ctx, ad); err != nil {
			return err
		}

		// Update ad location record
		if err := s.adLocationRepository.Update(ctx, adLocation); err != nil {
			return err
		}

		return nil
	})
}

// GetAdUnitConf retrieves ad unit configuration for a specific site and location
func (s *adsService) GetAdUnitConf(ctx context.Context, siteID string, location string) (*api.AdUnitConf, error) {
	// Get ad locations for the specified site and location
	locations, err := s.adLocationRepository.GetByLocation(ctx, location, siteID, model.AdStatusEnabled)
	if err != nil {
		return nil, err
	}

	if len(locations) == 0 {

		return nil, common.ErrNotFound
	}

	// Use the first matching location
	adLocation := locations[0]

	// Get the ad details
	ad, err := s.adRepository.GetByAdID(ctx, adLocation.AdID)
	if err != nil {
		return nil, err
	}

	if ad == nil {
		return nil, common.ErrNotFound
	}

	// Construct the response
	adConf := &api.AdUnitConf{
		Page:      adLocation.Location,
		AdNetwork: ad.AdNetwork,
		Format:    int(ad.Format),
		Status:    int(ad.Status),
		Conf:      map[string]interface{}(ad.Conf),
		ShowStg:   map[string]interface{}(adLocation.ShowStg),
	}

	return adConf, nil
}
