package payment

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

// CoinPackageRepository defines the repository interface for coin packages
type CoinPackageRepository interface {
	db.BaseOperation
	GetByPackageID(ctx context.Context, packageID string) (*model.CoinPackage, error)
	GetByIOSProductID(ctx context.Context, iosProductID string) (*model.CoinPackage, error) //ios_product_id
	ListBySiteID(ctx context.Context, siteID string, status int) ([]*model.CoinPackage, error)
}

type coinPackageRepository struct {
	*db.Repository
}

// NewCoinPackageRepository creates a new coin package repository
func NewCoinPackageRepository(r *db.Repository) CoinPackageRepository {
	return &coinPackageRepository{
		Repository: r,
	}
}

func (r *coinPackageRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *coinPackageRepository) Update(ctx context.Context, entity interface{}) error {
	recoder := entity.(*model.CoinPackage)
	return r.DB(ctx).Where("package_id = ?", recoder.PackageID).Updates(recoder).Error
}

func (r *coinPackageRepository) GetByPackageID(ctx context.Context, packageID string) (*model.CoinPackage, error) {
	var pkg model.CoinPackage
	err := r.DB(ctx).Where("package_id = ?", packageID).First(&pkg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &pkg, nil
}

func (r *coinPackageRepository) GetByIOSProductID(ctx context.Context, iosProductID string) (*model.CoinPackage, error) {
	var pkg model.CoinPackage
	err := r.DB(ctx).Where("ios_product_id = ?", iosProductID).First(&pkg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &pkg, nil
}

func (r *coinPackageRepository) ListBySiteID(ctx context.Context, siteID string, status int) ([]*model.CoinPackage, error) {
	var packages []*model.CoinPackage
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
