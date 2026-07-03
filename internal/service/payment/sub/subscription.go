package sub

import (
	"context"
	"errors"
	"shortpress-server/internal/api"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/service"
	"shortpress-server/internal/types"

	"github.com/google/uuid"
)

// Errors for subscription operations
var (
	ErrInvalidSubscriptionParams = errors.New("invalid subscription package parameters")
	ErrSubscriptionNotFound      = errors.New("subscription package not found")
)

// SubscriptionRequest represents the request to create or update a subscription package
// type SubscriptionRequest struct {
// 	SiteID 		 	   string  `json:"siteId"`
// 	PackageID          string  `json:"packageId"`
// 	Name               string  `json:"name"`
// 	Description        string  `json:"description"`
// 	Interval           string  `json:"interval"` //  day, week, month or year
// 	Price              float64 `json:"price"`
// 	OriginalPrice      float64 `json:"originalPrice"`
// 	DiscountPercentage int     `json:"discountPercentage"`
// 	Currency           string  `json:"currency"`
// 	Status             int     `json:"status"` // 1:enabled 2:disabled
// }

// SubscriptionService defines the interface for subscription operations
type SubscriptionService interface {
	// CreateSubscriptionPackage creates a new subscription package
	CreateSubscriptionPackage(ctx context.Context, siteID string, req *api.SubscriptionData) (*api.SubscriptionData, error)

	// ListBySiteID lists all subscription packages for a site
	ListBySiteID(ctx context.Context, siteID string, status int) ([]*api.SubscriptionData, error)

	// UpdateSubscriptionPackage updates a subscription package (only name, description and status can be updated)
	UpdateSubscriptionPackage(ctx context.Context, packageID string, req *api.SubscriptionData) error

	// GetByPackageID gets a specific subscription package by ID and siteID
	GetByPackageID(ctx context.Context, siteID, packageID string) (*api.SubscriptionData, error)
	// GetUserSubscriptions gets all subscriptions for a user
	GetUserSubscriptions(ctx context.Context, userID string) ([]*api.UserSubscriptionResponse, error)
	// GetUserSubscription gets a specific user subscription by ID
	GetUserSubscription(ctx context.Context, subscriptionID string) (*model.UserSubscription, error)
}

type subscriptionService struct {
	*service.Service
	subscriptionPackageRepo payment.SubscriptionPackageRepository
	userSubscriptionRepo    payment.UserSubscriptionRepository
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(
	svc *service.Service,
	subscriptionPackageRepo payment.SubscriptionPackageRepository,
	userSubscriptionRepo payment.UserSubscriptionRepository,
) SubscriptionService {
	return &subscriptionService{
		Service:                 svc,
		subscriptionPackageRepo: subscriptionPackageRepo,
		userSubscriptionRepo:    userSubscriptionRepo,
	}
}

// CreateSubscriptionPackage creates a new subscription package
func (s *subscriptionService) CreateSubscriptionPackage(ctx context.Context, siteID string, req *api.SubscriptionData) (*api.SubscriptionData, error) {
	// Validate request
	if req.Name == "" || req.Interval == "" || req.Price <= 0 {
		return nil, ErrInvalidSubscriptionParams
	}

	// Validate interval
	if req.Interval != "week" && req.Interval != "month" && req.Interval != "year" {
		return nil, ErrInvalidSubscriptionParams
	}

	// Set default values
	if req.Currency == "" {
		req.Currency = "USD"
	}

	if req.Status == 0 {
		req.Status = 1 // Default to enabled
	}

	// Create package
	pkg := &model.SubscriptionPackage{
		SiteID:             siteID,
		PackageID:          uuid.New().String(),
		Name:               req.Name,
		Description:        req.Description,
		Interval:           req.Interval,
		Price:              req.Price.Cents(),         // Store price in cents
		OriginalPrice:      req.OriginalPrice.Cents(), // Store original price in cents
		DiscountPercentage: req.DiscountPercentage,
		Currency:           req.Currency,
		Coins:              req.Coins,
		Rights:             req.Rights,
		Status:             req.Status,
		IOSProductID:       req.IOSProductID,
	}

	// Save to repository
	err := s.subscriptionPackageRepo.Create(ctx, pkg)
	if err != nil {
		return nil, err
	}

	// Convert to API response
	response := &api.SubscriptionData{
		PackageID:          pkg.PackageID,
		SiteID:             pkg.SiteID,
		Name:               pkg.Name,
		Description:        pkg.Description,
		Interval:           pkg.Interval,
		Price:              types.FromCents(pkg.Price),
		OriginalPrice:      types.FromCents(pkg.OriginalPrice),
		Currency:           pkg.Currency,
		DiscountPercentage: pkg.DiscountPercentage,
		Coins:              pkg.Coins,
		Rights:             pkg.Rights,
		Status:             pkg.Status,
		IOSProductID:       pkg.IOSProductID,
		CreatedAt:          pkg.CreatedAt.Unix(),
	}

	return response, nil
}

// ListBySiteID lists all subscription packages for a site
func (s *subscriptionService) ListBySiteID(ctx context.Context, siteID string, status int) ([]*api.SubscriptionData, error) {
	// In this implementation, we're using creatorID as siteID to match the interface
	// This assumes the Site ID matches the Creator ID in your system
	packages, err := s.subscriptionPackageRepo.ListBySiteID(ctx, siteID, status)
	if err != nil {
		return nil, err
	}

	// Convert model.SubscriptionPackage to api.SubscriptionPackageResponse
	var responses []*api.SubscriptionData
	for _, pkg := range packages {
		response := &api.SubscriptionData{
			PackageID:          pkg.PackageID,
			SiteID:             pkg.SiteID,
			Name:               pkg.Name,
			Description:        pkg.Description,
			Interval:           pkg.Interval,
			Price:              types.FromCents(pkg.Price),
			OriginalPrice:      types.FromCents(pkg.OriginalPrice),
			Currency:           pkg.Currency,
			DiscountPercentage: pkg.DiscountPercentage,
			Coins:              pkg.Coins,
			Rights:             pkg.Rights,
			Status:             pkg.Status,
			IOSProductID:       pkg.IOSProductID,
			CreatedAt:          pkg.CreatedAt.Unix(),
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// UpdateSubscriptionPackage updates a subscription package (only name, description, coins, rights and status can be updated)
func (s *subscriptionService) UpdateSubscriptionPackage(ctx context.Context, packageID string, req *api.SubscriptionData) error {
	// Find existing package
	pkg, err := s.subscriptionPackageRepo.GetByPackageID(ctx, packageID)
	if err != nil {
		return err
	}

	if pkg == nil {
		return ErrSubscriptionNotFound
	}

	// Update allowed fields only
	if req.Name != "" {
		pkg.Name = req.Name
	}

	if req.Description != "" {
		pkg.Description = req.Description
	}

	// Allow updating coins and rights
	pkg.Coins = req.Coins
	pkg.Rights = req.Rights

	if req.Status != 0 {
		pkg.Status = req.Status
	}

	pkg.IOSProductID = req.IOSProductID

	// Save changes
	err = s.subscriptionPackageRepo.Update(ctx, pkg)
	if err != nil {
		return err
	}

	return nil
}

// GetByPackageID gets a specific subscription package by ID and siteID
func (s *subscriptionService) GetByPackageID(ctx context.Context, siteID, packageID string) (*api.SubscriptionData, error) {
	// Find package by ID
	pkg, err := s.subscriptionPackageRepo.GetByPackageID(ctx, packageID)
	if err != nil {
		return nil, err
	}

	if pkg == nil {
		return nil, ErrSubscriptionNotFound
	}

	// 将模型对象转换为API响应对象
	response := &api.SubscriptionData{
		PackageID:          pkg.PackageID,
		SiteID:             pkg.SiteID,
		Name:               pkg.Name,
		Description:        pkg.Description,
		Interval:           pkg.Interval,
		Price:              types.FromCents(pkg.Price),
		OriginalPrice:      types.FromCents(pkg.OriginalPrice),
		Currency:           pkg.Currency,
		DiscountPercentage: pkg.DiscountPercentage,
		Coins:              pkg.Coins,
		Rights:             pkg.Rights,
		Status:             pkg.Status,
		IOSProductID:       pkg.IOSProductID,
		CreatedAt:          pkg.CreatedAt.Unix(),
	}

	return response, nil
}

// GetUserSubscriptions gets all subscriptions for a user
func (s *subscriptionService) GetUserSubscriptions(ctx context.Context, userID string) ([]*api.UserSubscriptionResponse, error) {
	// Get all subscriptions for the user
	subscriptions, err := s.userSubscriptionRepo.ListByUserID(ctx, userID, 1, 0)
	if err != nil {
		return nil, err
	}

	if len(subscriptions) == 0 {
		return []*api.UserSubscriptionResponse{}, nil
	}

	var responses []*api.UserSubscriptionResponse

	// For each subscription, create a response object
	//TODO 后续改为关联查询
	for _, sub := range subscriptions {
		// Get the associated subscription package
		pkg, err := s.subscriptionPackageRepo.GetByPackageID(ctx, sub.PackageID)
		if err != nil {
			return nil, err
		}

		response := &api.UserSubscriptionResponse{
			SubscriptionID:     sub.SubscriptionID,
			UserID:             sub.UserID,
			SiteID:             sub.SiteID,
			PackageID:          sub.PackageID,
			Provider:           sub.Provider,
			Status:             sub.Status,
			CurrentPeriodStart: sub.CurrentPeriodStart.Unix(),
			CurrentPeriodEnd:   sub.CurrentPeriodEnd.Unix(),
			CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
			CreatedAt:          sub.CreatedAt.Unix(),
		}

		// Add package details if available
		if pkg != nil {
			response.PackageName = pkg.Name
			response.PackageDescription = pkg.Description
			response.Interval = pkg.Interval
			response.Price = types.FromCents(pkg.Price)
			response.Currency = pkg.Currency
		}

		responses = append(responses, response)
	}

	return responses, nil
}

// GetUserSubscription gets a specific user subscription by ID
func (s *subscriptionService) GetUserSubscription(ctx context.Context, subscriptionID string) (*model.UserSubscription, error) {
	subscription, err := s.userSubscriptionRepo.GetBySubscriptionID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	if subscription == nil {
		return nil, ErrSubscriptionNotFound
	}

	return subscription, nil
}
