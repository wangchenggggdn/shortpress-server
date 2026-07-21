package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"shortpress-server/internal/api"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/user"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// AnalyticsService handles analytics operations
type AnalyticsService interface {
	// GetPaymentTransactions retrieves payment transactions for a site within a time range
	GetPaymentTransactions(ctx *gin.Context, siteID string, userID string, userEmail string, startTime, endTime time.Time, page, pageSize int) (*api.IncomeTransactionHistoryResponse, error)
	// GetIncomeStatistics retrieves daily income statistics for a site within a time range.
	// timezoneOffsetMinutes is minutes east of UTC used to bucket calendar days; nil keeps DATE(created_at).
	GetIncomeStatistics(ctx *gin.Context, siteID string, startTime, endTime time.Time, timezoneOffsetMinutes *int) (*api.IncomeStatisticsResponse, error)
	// GetTransactionByID retrieves a specific payment transaction by its ID
	GetTransactionByID(ctx *gin.Context, transactionID string) (*api.IncomeTransactionDetailResponse, error)
	// GetCreations lists recent user creation records for a site from generate Redis.
	GetCreations(ctx *gin.Context, siteID string, page, pageSize int) (*api.CreationsResponse, error)
}

type analyticsService struct {
	*Service
	paymentTransactionRepo payment.PaymentTransactionRepository
	userRepository         user.UserRepository
	userSubscriptionRepo   payment.UserSubscriptionRepository
	generateServiceURL     string
	httpClient             *http.Client
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(
	service *Service,
	paymentTransactionRepo payment.PaymentTransactionRepository,
	userRepository user.UserRepository,
	userSubscriptionRepo payment.UserSubscriptionRepository,
	conf *viper.Viper,
) AnalyticsService {
	return &analyticsService{
		Service:                service,
		paymentTransactionRepo: paymentTransactionRepo,
		userRepository:         userRepository,
		userSubscriptionRepo:   userSubscriptionRepo,
		generateServiceURL:     strings.TrimRight(conf.GetString("a2e.generate_service_url"), "/"),
		httpClient:             &http.Client{Timeout: 15 * time.Second},
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
			PixelID:       tx.PixelID,
			Platform:      tx.Platform,
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
	timezoneOffsetMinutes *int,
) (*api.IncomeStatisticsResponse, error) {
	// Add business context logging
	log.AddNotice(ctx, "target_site_id", siteID)
	log.AddNotice(ctx, "time_range", fmt.Sprintf("%s to %s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02")))
	if timezoneOffsetMinutes != nil {
		log.AddNotice(ctx, "timezone_offset_minutes", fmt.Sprintf("%d", *timezoneOffsetMinutes))
	}

	// Get daily statistics from repository
	stats, err := s.paymentTransactionRepo.GetDailyIncomeStatistics(
		ctx,
		siteID,
		startTime,
		endTime,
		timezoneOffsetMinutes,
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
	var platform string
	if transaction.UserID != "" {
		user, err := s.userRepository.GetByUserID(ctx, transaction.UserID)
		if err == nil && user != nil {
			email = user.Email
			platform = user.Platform
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
		Platform:      platform,
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

type generateCreationsUpstream struct {
	Code int `json:"code"`
	Data struct {
		Items    []map[string]any `json:"items"`
		Total    int64            `json:"total"`
		Page     int              `json:"page"`
		PageSize int              `json:"page_size"`
	} `json:"data"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// GetCreations lists recent user creation records for a site from generate Redis.
func (s *analyticsService) GetCreations(ctx *gin.Context, siteID string, page, pageSize int) (*api.CreationsResponse, error) {
	if s.generateServiceURL == "" {
		return nil, fmt.Errorf("generate service url is not configured")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	q := url.Values{}
	q.Set("site_id", siteID)
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))
	endpoint := s.generateServiceURL + "/creations?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx.Request.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if auth := ctx.GetHeader("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("X-Site-Id", siteID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call generate creations failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("generate creations status %d: %s", resp.StatusCode, string(body))
	}

	var upstream generateCreationsUpstream
	if err := json.Unmarshal(body, &upstream); err != nil {
		return nil, fmt.Errorf("decode generate creations failed: %w", err)
	}
	if upstream.Code != 0 && upstream.Code != 200 {
		msg := upstream.Message
		if msg == "" {
			msg = upstream.Error
		}
		if msg == "" {
			msg = "generate creations failed"
		}
		return nil, fmt.Errorf("%s", msg)
	}

	items := make([]*api.CreationRecordItem, 0, len(upstream.Data.Items))
	for _, raw := range upstream.Data.Items {
		items = append(items, mapCreationRecord(raw))
	}

	return &api.CreationsResponse{
		Items:    items,
		Total:    upstream.Data.Total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func mapCreationRecord(raw map[string]any) *api.CreationRecordItem {
	item := &api.CreationRecordItem{
		TaskID:          stringFromAny(raw["task_id"]),
		Status:          int32FromAny(raw["status"]),
		Model:           stringFromAny(raw["model"]),
		VideoID:         stringFromAny(raw["video_id"]),
		SiteID:          stringFromAny(raw["site_id"]),
		UserID:          stringFromAny(raw["user_id"]),
		Prompt:          stringFromAny(raw["prompt"]),
		ReferenceImages: stringSliceFromAny(raw["reference_images"]),
		Images:          stringSliceFromAny(raw["images"]),
		ErrorMsg:        stringFromAny(raw["error_msg"]),
		CreatedAt:       unixFromAny(raw["created_at"]),
		UpdatedAt:       unixFromAny(raw["updated_at"]),
	}
	if videos, ok := raw["videos"].([]any); ok {
		for _, v := range videos {
			switch typed := v.(type) {
			case map[string]any:
				item.Videos = append(item.Videos, api.CreationVideo{
					URL:      stringFromAny(typed["url"]),
					CoverURL: stringFromAny(typed["cover_url"]),
				})
			case string:
				item.Videos = append(item.Videos, api.CreationVideo{URL: typed})
			}
		}
	}
	return item
}

func stringFromAny(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	default:
		return ""
	}
}

func int32FromAny(v any) int32 {
	switch t := v.(type) {
	case float64:
		return int32(t)
	case int:
		return int32(t)
	case int32:
		return t
	case int64:
		return int32(t)
	case json.Number:
		i, _ := t.Int64()
		return int32(i)
	case string:
		i, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 32)
		return int32(i)
	default:
		return 0
	}
}

func stringSliceFromAny(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s := stringFromAny(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func unixFromAny(v any) int64 {
	switch t := v.(type) {
	case float64:
		// JSON numbers or unix seconds
		if t > 1e12 {
			return int64(t / 1000)
		}
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		if tm, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return tm.Unix()
		}
		if tm, err := time.Parse(time.RFC3339, s); err == nil {
			return tm.Unix()
		}
		return 0
	default:
		return 0
	}
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
