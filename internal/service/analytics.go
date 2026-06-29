package service

import (
	"fmt"
	"shortpress-server/internal/api"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/user"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AnalyticsService handles analytics operations
type AnalyticsService interface {
	// GetPaymentTransactions retrieves payment transactions for a site within a time range
	GetPaymentTransactions(ctx *gin.Context, siteID string, userID string, userEmail string, startTime, endTime time.Time, page, pageSize int) (*api.IncomeTransactionHistoryResponse, error)
	// GetIncomeStatistics retrieves daily income statistics for a site within a time range
	GetIncomeStatistics(ctx *gin.Context, siteID string, startTime, endTime time.Time) (*api.IncomeStatisticsResponse, error)
	// GetTransactionByID retrieves a specific payment transaction by its ID
	GetTransactionByID(ctx *gin.Context, transactionID string) (*api.IncomeTransactionDetailResponse, error)
}

type analyticsService struct {
	*Service
	paymentTransactionRepo payment.PaymentTransactionRepository
	userRepository         user.UserRepository
	userSubscriptionRepo   payment.UserSubscriptionRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(
	service *Service,
	paymentTransactionRepo payment.PaymentTransactionRepository,
	userRepository user.UserRepository,
	userSubscriptionRepo payment.UserSubscriptionRepository,
) AnalyticsService {
	return &analyticsService{
		Service:                service,
		paymentTransactionRepo: paymentTransactionRepo,
		userRepository:         userRepository,
		userSubscriptionRepo:   userSubscriptionRepo,
	}
}

// GetPaymentTransactions retrieves payment transactions for a site within a time range
func (s *analyticsService) GetPaymentTransactions(
	ctx *gin.Context,
	siteID string,
	userID string,
	userEmail string,
	startTime,
	endTime time.Time,
	page,
	pageSize int,
) (*api.IncomeTransactionHistoryResponse, error) {
	// Add business context logging
	log.AddNotice(ctx, "target_site_id", siteID)
	log.AddNotice(ctx, "filter_user_id", userID)
	log.AddNotice(ctx, "filter_user_email", userEmail)

	filterUserID := userID
	emailSearch := ""
	if filterUserID == "" && userEmail != "" {
		emailSearch = strings.TrimSpace(userEmail)
	}
	offset := (page - 1) * pageSize

	// Get transactions for the site within the time range
	transactions, err := s.paymentTransactionRepo.ListBySiteIDAndTimeRange(
		ctx,
		siteID,
		filterUserID,
		emailSearch,
		startTime,
		endTime,
		pageSize,
		offset,
	)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to query transactions for site %s: %v", siteID, err))
		return nil, err
	}

	// Get total count
	count, err := s.paymentTransactionRepo.CountBySiteIDAndTimeRange(
		ctx,
		siteID,
		filterUserID,
		emailSearch,
		startTime,
		endTime,
	)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to count transactions for site %s: %v", siteID, err))
		return nil, err
	}

	// Convert to response format
	responseTransactions := make([]*api.IncomeTransactionItem, 0, len(transactions))
	for _, tx := range transactions {
		name := ""
		if n, ok := tx.Snapshot["name"]; ok {
			name = n.(string)
		}
		responseTransactions = append(responseTransactions, &api.IncomeTransactionItem{
			TransactionID: tx.TransactionID,
			Name:          name,
			Email:         tx.Email,
			PayerEmail:    tx.PayerEmail,
			Amount:        types.FromCents(tx.Amount),
			Provider:      tx.Provider,
			Description:   tx.ErrorMessage,
			CreatedAt:     tx.CreatedAt.Unix(),
		})
	}

	// Add success logging
	log.AddNotice(ctx, "transactions_found", fmt.Sprintf("%d", len(responseTransactions)))
	log.AddNotice(ctx, "total_count", fmt.Sprintf("%d", count))

	// Return response
	return &api.IncomeTransactionHistoryResponse{
		Items:    responseTransactions,
		Total:    count,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetIncomeStatistics retrieves daily income statistics for a site within a time range
func (s *analyticsService) GetIncomeStatistics(
	ctx *gin.Context,
	siteID string,
	startTime,
	endTime time.Time,
) (*api.IncomeStatisticsResponse, error) {
	// Add business context logging
	log.AddNotice(ctx, "target_site_id", siteID)
	log.AddNotice(ctx, "time_range", fmt.Sprintf("%s to %s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02")))

	// Get daily statistics from repository
	stats, err := s.paymentTransactionRepo.GetDailyIncomeStatistics(
		ctx,
		siteID,
		startTime,
		endTime,
	)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to get daily income statistics for site %s: %v", siteID, err))
		return nil, err
	}

	// Convert to response format
	responseStats := make([]api.DailyIncomeStatistics, 0, len(stats))
	for _, stat := range stats {
		responseStats = append(responseStats, api.DailyIncomeStatistics{
			Date:               stat.Date,
			TotalAmount:        types.FromCents(stat.TotalAmount),
			TransactionCount:   stat.TransactionCount,
			IapAmount:          types.FromCents(stat.IapAmount),
			SubscriptionAmount: types.FromCents(stat.SubscriptionAmount),
			RenewalAmount:      types.FromCents(stat.RenewalAmount),
		})
	}

	// Add success logging
	log.AddNotice(ctx, "statistics_days", fmt.Sprintf("%d", len(responseStats)))

	// Return response
	return &api.IncomeStatisticsResponse{
		Items: responseStats,
	}, nil
}

// GetTransactionByID retrieves a specific payment transaction by its ID
func (s *analyticsService) GetTransactionByID(ctx *gin.Context, transactionID string) (*api.IncomeTransactionDetailResponse, error) {

	// Get transaction from repository
	transaction, err := s.paymentTransactionRepo.GetByTransactionID(ctx, transactionID)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to get transaction %s: %v", transactionID, err))
		return nil, err
	}

	if transaction == nil {
		log.Warning(ctx, fmt.Sprintf("Transaction not found: %s", transactionID))
		return nil, nil // Not found, but not an error
	}

	// Get user account email
	var email string
	if transaction.UserID != "" {
		user, err := s.userRepository.GetByUserID(ctx, transaction.UserID)
		if err == nil && user != nil {
			email = user.Email
		} else if err != nil {
			log.Warning(ctx, fmt.Sprintf("Failed to get user info for transaction %s: %v", transactionID, err))

		}
	}
	name := ""
	if n, ok := transaction.Snapshot["name"]; ok {
		name = n.(string)
	}
	// Convert to response format
	response := &api.IncomeTransactionDetailResponse{
		TransactionID: transaction.TransactionID,
		Name:          name,
		UserID:        transaction.UserID,
		Email:         email,
		PayerEmail:    transaction.PayerEmail,
		Amount:        types.FromCents(transaction.Amount),
		Currency:      transaction.Currency,
		Provider:      transaction.Provider,
		PaymentType:   transaction.PaymentType,
		Status:        transaction.Status,
		RelatedID:     transaction.RelatedID,
		RelatedType:   transaction.RelatedType,
		CreatedAt:     transaction.CreatedAt.Unix(),
	}

	if isSubscriptionTransaction(transaction) {
		response.IsSubscriptionOrder = !isSubscriptionRenewalTransaction(transaction)
		if transaction.UserID != "" && transaction.SiteID != "" {
			var userSubscription *model.UserSubscription
			var subErr error
			if transaction.RelatedID != "" {
				userSubscription, subErr = s.userSubscriptionRepo.GetByUserSiteAndPackage(
					ctx,
					transaction.UserID,
					transaction.SiteID,
					transaction.RelatedID,
				)
			} else {
				userSubscription, subErr = s.userSubscriptionRepo.GetActiveByUserAndSite(
					ctx,
					transaction.UserID,
					transaction.SiteID,
				)
			}
			if subErr != nil {
				log.Warning(ctx, fmt.Sprintf("Failed to get user subscription for transaction %s: %v", transactionID, subErr))
			} else if userSubscription != nil &&
				userSubscription.Status == model.SubscriptionStatusActive &&
				!userSubscription.CancelAtPeriodEnd {
				response.SubscriptionID = userSubscription.SubscriptionID
			}
		}
	}

	return response, nil
}

func isSubscriptionTransaction(transaction *model.PaymentTransaction) bool {
	if transaction == nil {
		return false
	}
	return transaction.PaymentType == model.PaymentTypeSubscription ||
		transaction.RelatedType == model.RelatedTypeSubscription
}

func isSubscriptionRenewalTransaction(transaction *model.PaymentTransaction) bool {
	if transaction == nil {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(transaction.ProviderPaymentID), "in_")
}
