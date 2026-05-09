package payment

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
	"time"

	"gorm.io/gorm"
)

// CoinTransactionRepository defines the repository interface for coin transactions
type CoinTransactionRepository interface {
	db.BaseOperation
	GetByTransactionID(ctx context.Context, transactionID string) (*model.CoinTransaction, error)
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.CoinTransaction, error)
	ListAddCoionsByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.CoinTransaction, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	ListBySiteID(ctx context.Context, siteID string, limit, offset int) ([]*model.CoinTransaction, error)
	CountBySiteID(ctx context.Context, siteID string) (int64, error)
	ListBySiteIDAndTimeRange(ctx context.Context, siteID string, startTime, endTime time.Time, limit, offset int) ([]*model.CoinTransaction, error)
	CountBySiteIDAndTimeRange(ctx context.Context, siteID string, startTime, endTime time.Time) (int64, error)
}

type coinTransactionRepository struct {
	*db.Repository
}

// NewCoinTransactionRepository creates a new coin transaction repository
func NewCoinTransactionRepository(r *db.Repository) CoinTransactionRepository {
	return &coinTransactionRepository{
		Repository: r,
	}
}

func (r *coinTransactionRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *coinTransactionRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

func (r *coinTransactionRepository) GetByTransactionID(ctx context.Context, transactionID string) (*model.CoinTransaction, error) {
	var tx model.CoinTransaction
	err := r.DB(ctx).Where("transaction_id = ?", transactionID).First(&tx).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *coinTransactionRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.CoinTransaction, error) {
	var transactions []*model.CoinTransaction
	query := r.DB(ctx).Where("user_id = ? ", userID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&transactions).Error
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *coinTransactionRepository) ListAddCoionsByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.CoinTransaction, error) {
	var transactions []*model.CoinTransaction
	query := r.DB(ctx).Where("user_id = ? AND (source = ? OR source = ?)", userID, "purchase", "admin_adjustment").Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&transactions).Error
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *coinTransactionRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.DB(ctx).Model(&model.CoinTransaction{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *coinTransactionRepository) ListBySiteID(ctx context.Context, siteID string, limit, offset int) ([]*model.CoinTransaction, error) {
	var transactions []*model.CoinTransaction
	query := r.DB(ctx).Where("site_id = ?", siteID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&transactions).Error
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *coinTransactionRepository) CountBySiteID(ctx context.Context, siteID string) (int64, error) {
	var count int64
	err := r.DB(ctx).Model(&model.CoinTransaction{}).Where("site_id = ?", siteID).Count(&count).Error
	return count, err
}

func (r *coinTransactionRepository) ListBySiteIDAndTimeRange(ctx context.Context, siteID string, startTime, endTime time.Time, limit, offset int) ([]*model.CoinTransaction, error) {
	var transactions []*model.CoinTransaction
	query := r.DB(ctx).Where("site_id = ?", siteID).Order("created_at DESC")

	// Add time range conditions
	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&transactions).Error
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *coinTransactionRepository) CountBySiteIDAndTimeRange(ctx context.Context, siteID string, startTime, endTime time.Time) (int64, error) {
	var count int64
	query := r.DB(ctx).Model(&model.CoinTransaction{}).Where("site_id = ?", siteID)

	// Add time range conditions
	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	err := query.Count(&count).Error
	return count, err
}
