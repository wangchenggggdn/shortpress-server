package payment

import (
	"context"
	"fmt"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
	"time"

	"gorm.io/gorm"
)

// PaymentTransactionRepository defines the repository interface for payment transactions
type PaymentTransactionRepository interface {
	db.BaseOperation
	GetByTransactionID(ctx context.Context, transactionID string) (*model.PaymentTransaction, error)
	GetByProviderPaymentID(ctx context.Context, provider, paymentID string) (*model.PaymentTransaction, error)
	MarkSuccessIfPending(ctx context.Context, transactionID, providerPaymentID, payerEmail string) (bool, error)
	ListBySiteIDAndTimeRange(ctx context.Context, siteID string, userID string, startTime, endTime time.Time, limit, offset int) ([]*model.PaymentTransactionView, error)
	CountBySiteIDAndTimeRange(ctx context.Context, siteID string, userID string, startTime, endTime time.Time) (int64, error)
	GetDailyIncomeStatistics(ctx context.Context, siteID string, startTime, endTime time.Time) ([]*model.DailyIncomeStatistics, error)
	GetUserTotalAmount(ctx context.Context, userID string, siteID string, startTime, endTime time.Time) (int64, error)
	ListUserPurchases(ctx context.Context, userID string, siteID string, page, pageSize int) ([]*model.PaymentTransaction, int64, error)
	HasUserPurchased(ctx context.Context, userID string, siteID string) (bool, error)
	GetUserIDByPayerEmail(ctx context.Context, siteID, payerEmail string) (string, error)
}

type paymentTransactionRepository struct {
	*db.Repository
}

// NewPaymentTransactionRepository creates a new payment transaction repository
func NewPaymentTransactionRepository(r *db.Repository) PaymentTransactionRepository {
	return &paymentTransactionRepository{
		Repository: r,
	}
}

func (r *paymentTransactionRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *paymentTransactionRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

func (r *paymentTransactionRepository) GetByTransactionID(ctx context.Context, transactionID string) (*model.PaymentTransaction, error) {
	var tx model.PaymentTransaction
	err := r.DB(ctx).Where("transaction_id = ?", transactionID).First(&tx).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *paymentTransactionRepository) GetByProviderPaymentID(ctx context.Context, provider, paymentID string) (*model.PaymentTransaction, error) {
	var tx model.PaymentTransaction
	err := r.DB(ctx).Where("provider = ? AND provider_payment_id = ?", provider, paymentID).First(&tx).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

// MarkSuccessIfPending atomically marks a pending transaction as success.
// Returns true when current request successfully claims processing ownership.
func (r *paymentTransactionRepository) MarkSuccessIfPending(ctx context.Context, transactionID, providerPaymentID, payerEmail string) (bool, error) {
	updates := map[string]interface{}{
		"status":              model.PaymentStatusSuccess,
		"provider_payment_id": providerPaymentID,
	}
	if payerEmail != "" {
		updates["payer_email"] = payerEmail
	}
	result := r.DB(ctx).Model(&model.PaymentTransaction{}).
		Where("transaction_id = ? AND status = ?", transactionID, model.PaymentStatusPending).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// ListBySiteIDAndTimeRange retrieves payment transactions for a site within a time range
func (r *paymentTransactionRepository) ListBySiteIDAndTimeRange(ctx context.Context, siteID string, userID string, startTime, endTime time.Time, limit, offset int) ([]*model.PaymentTransactionView, error) {
	var transactions []*model.PaymentTransactionView

	query := r.DB(ctx).Table("payment_transactions").Select(
		"payment_transactions.*, users.email AS email",
	).
		Joins("LEFT JOIN users ON payment_transactions.user_id = users.user_id").
		Where("payment_transactions.site_id = ?", siteID)

	if userID != "" {
		query = query.Where("payment_transactions.user_id = ?", userID)
	}

	// Add time range conditions
	if !startTime.IsZero() {
		query = query.Where("payment_transactions.created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("payment_transactions.created_at <= ?", endTime)
	}

	query = query.Where("payment_transactions.status = ?", model.PaymentStatusSuccess)

	//created_at desc
	query = query.Order("payment_transactions.created_at DESC")

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

// CountBySiteIDAndTimeRange counts payment transactions for a site within a time range
func (r *paymentTransactionRepository) CountBySiteIDAndTimeRange(ctx context.Context, siteID string, userID string, startTime, endTime time.Time) (int64, error) {
	var count int64
	query := r.DB(ctx).Model(&model.PaymentTransaction{}).Where("site_id = ?", siteID)

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	// Add time range conditions
	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	query = query.Where("status = ?", model.PaymentStatusSuccess)

	err := query.Count(&count).Error
	return count, err
}

// GetDailyIncomeStatistics retrieves daily income statistics for a site within a time range
func (r *paymentTransactionRepository) GetDailyIncomeStatistics(ctx context.Context, siteID string, startTime, endTime time.Time) ([]*model.DailyIncomeStatistics, error) {
	var statistics []*model.DailyIncomeStatistics

	// Renewal: Stripe invoice id on provider_payment_id (must start with "in_").
	// Use REGEXP '^in_' (not LIKE 'in_%') — SQL LIKE treats "_" as a single-char wildcard.
	isRenewal := `TRIM(COALESCE(provider_payment_id, '')) REGEXP '^in_'`
	notRenewal := fmt.Sprintf(`NOT (%s)`, isRenewal)
	isSubscriptionOrder := fmt.Sprintf(`(
		related_type = %d OR payment_type = %d
		OR COALESCE(JSON_UNQUOTE(JSON_EXTRACT(snapshot, '$.interval')), '') IN ('week', 'month', 'year')
		OR COALESCE(JSON_UNQUOTE(JSON_EXTRACT(snapshot, '$.stripe_subscription_billing')), '') IN ('initial', 'one_time', 'recurring')
	)`, model.RelatedTypeSubscription, model.PaymentTypeSubscription)
	isNewSubscription := fmt.Sprintf(`(%s AND %s)`, notRenewal, isSubscriptionOrder)
	isIAP := fmt.Sprintf(`(%s AND NOT (%s) AND (related_type = %d OR payment_type = %d))`,
		notRenewal, isSubscriptionOrder, model.RelatedTypeCoinPackage, model.PaymentTypeCoinPackage)

	query := r.DB(ctx).Model(&model.PaymentTransaction{}).
		Select(fmt.Sprintf(`DATE(created_at) as date,
			SUM(amount) as total_amount,
			COUNT(*) as transaction_count,
			SUM(CASE WHEN %s THEN amount ELSE 0 END) as renewal_amount,
			SUM(CASE WHEN %s THEN amount ELSE 0 END) as subscription_amount,
			SUM(CASE WHEN %s THEN amount ELSE 0 END) as iap_amount`,
			isRenewal,
			isNewSubscription,
			isIAP)).
		Where("site_id = ? AND status = ?", siteID, model.PaymentStatusSuccess)

	// Add time range conditions
	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	err := query.Group("DATE(created_at)").
		Order("date DESC").
		Find(&statistics).Error

	if err != nil {
		return nil, err
	}

	return statistics, nil
}

// GetUserTotalAmount retrieves the total amount spent by a user on a site within a time range
func (r *paymentTransactionRepository) GetUserTotalAmount(ctx context.Context, userID string, siteID string, startTime, endTime time.Time) (int64, error) {
	var totalAmount int64

	query := r.DB(ctx).Model(&model.PaymentTransaction{}).
		// Select("SUM(amount)").
		Select("COALESCE(SUM(amount), 0)").
		Where("user_id = ? AND site_id = ? AND status = ?", userID, siteID, model.PaymentStatusSuccess)

	// Add time range conditions
	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	err := query.Scan(&totalAmount).Error
	if err != nil {
		return 0, err
	}

	return totalAmount, nil
}

// ListUserPurchases retrieves user's purchase records (coin purchases and subscriptions)
func (r *paymentTransactionRepository) ListUserPurchases(ctx context.Context, userID string, siteID string, page, pageSize int) ([]*model.PaymentTransaction, int64, error) {
	var transactions []*model.PaymentTransaction
	var total int64

	// Build query - only get successful purchases
	query := r.DB(ctx).Model(&model.PaymentTransaction{}).
		Where("user_id = ? AND site_id = ?", userID, siteID).
		Where("payment_type IN ?", []int{model.PaymentTypeCoinPackage, model.PaymentTypeSubscription}).
		Where("status = ?", model.PaymentStatusSuccess)

	// Count total
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Fetch records with pagination
	err = query.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&transactions).Error
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

// HasUserPurchased checks if a user has made any successful purchases
func (r *paymentTransactionRepository) HasUserPurchased(ctx context.Context, userID string, siteID string) (bool, error) {
	var count int64
	err := r.DB(ctx).Model(&model.PaymentTransaction{}).
		Where("user_id = ? AND site_id = ?", userID, siteID).
		Where("payment_type IN ?", []int{model.PaymentTypeCoinPackage, model.PaymentTypeSubscription}).
		Where("status = ?", model.PaymentStatusSuccess).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserIDByPayerEmail finds the most recent user who paid with the given checkout email.
func (r *paymentTransactionRepository) GetUserIDByPayerEmail(ctx context.Context, siteID, payerEmail string) (string, error) {
	if payerEmail == "" {
		return "", nil
	}
	var userID string
	err := r.DB(ctx).Model(&model.PaymentTransaction{}).
		Select("user_id").
		Where("site_id = ? AND payer_email = ? AND status = ?", siteID, payerEmail, model.PaymentStatusSuccess).
		Order("created_at DESC").
		Limit(1).
		Scan(&userID).Error
	if err != nil {
		return "", err
	}
	return userID, nil
}
