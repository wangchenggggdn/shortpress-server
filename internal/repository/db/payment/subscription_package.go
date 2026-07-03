package payment

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

// SubscriptionPackageRepository defines the repository interface for subscription packages
type SubscriptionPackageRepository interface {
	db.BaseOperation
	GetByPackageID(ctx context.Context, packageID string) (*model.SubscriptionPackage, error)
	GetByIOSProductID(ctx context.Context, iosProductID string) (*model.SubscriptionPackage, error)
	ListBySiteID(ctx context.Context, siteID string, status int) ([]*model.SubscriptionPackage, error)
	ListByStatus(ctx context.Context, status int) ([]*model.SubscriptionPackage, error)
}

type subscriptionPackageRepository struct {
	*db.Repository
}

// NewSubscriptionPackageRepository creates a new subscription package repository
func NewSubscriptionPackageRepository(r *db.Repository) SubscriptionPackageRepository {
	return &subscriptionPackageRepository{
		Repository: r,
	}
}

func (r *subscriptionPackageRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *subscriptionPackageRepository) Update(ctx context.Context, entity interface{}) error {
	recoder := entity.(*model.SubscriptionPackage)
	return r.DB(ctx).Model(&model.SubscriptionPackage{}).Where("package_id = ?", recoder.PackageID).
		Updates(map[string]interface{}{
			"name":           recoder.Name,
			"description":    recoder.Description,
			"coins":          recoder.Coins,
			"rights":         recoder.Rights,
			"status":         recoder.Status,
			"ios_product_id": recoder.IOSProductID,
		}).Error
}

func (r *subscriptionPackageRepository) GetByPackageID(ctx context.Context, packageID string) (*model.SubscriptionPackage, error) {
	var pkg model.SubscriptionPackage
	err := r.DB(ctx).Where("package_id = ?", packageID).First(&pkg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &pkg, nil
}

func (r *subscriptionPackageRepository) GetByIOSProductID(ctx context.Context, iosProductID string) (*model.SubscriptionPackage, error) {
	var pkg model.SubscriptionPackage
	err := r.DB(ctx).Where("ios_product_id = ?", iosProductID).First(&pkg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &pkg, nil
}

func (r *subscriptionPackageRepository) ListBySiteID(ctx context.Context, siteID string, status int) ([]*model.SubscriptionPackage, error) {
	var packages []*model.SubscriptionPackage
	query := r.DB(ctx).Where("site_id = ?", siteID)
	if status != -1 {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&packages).Error
	if err != nil {
		return nil, err
	}
	return packages, nil
}

func (r *subscriptionPackageRepository) ListByStatus(ctx context.Context, status int) ([]*model.SubscriptionPackage, error) {
	var packages []*model.SubscriptionPackage
	query := r.DB(ctx)
	if status != -1 {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&packages).Error
	if err != nil {
		return nil, err
	}
	return packages, nil
}
