package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"shortpress-server/internal/adapter/payment/stripe"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/service"
	"shortpress-server/internal/service/analytics"
	"shortpress-server/internal/service/payment/coins"
	payutil "shortpress-server/internal/service/payment"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/log"

	"github.com/spf13/viper"
	goStripe "github.com/stripe/stripe-go/v82"
	"go.uber.org/zap"

	"strconv"
	"strings"

	peymentRep "shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/user"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
// CALL_BACK_SERVICE = "http://localhost:8001/payment/callback/stripe"
// CALL_BACK_SERVICE = "http://localhost:8001/payment/callback/stripe?siteId="
)

type StripeService interface {
	GetAccountInfo(ctx context.Context, siteID string) (*api.PaymentAccountInfo, error)
	ConfTest(ctx context.Context, sk string) error
	SaveConfig(ctx *gin.Context, config api.PaymentProviderConfig) error
	CreateCoinPackage(ctx context.Context, req api.CoinPackageCreateRequest) (*api.CoinPackageCreateResponse, error)
	CreateOrder(ctx *gin.Context, userID string, req api.OrderCreateRequest) (*api.OrderCreateResponse, error)
	HandleWebhook(ctx *gin.Context) error
	GetConfigInfo(ctx *gin.Context, siteID string) (*api.PaymentConfigInfoResponse, error)
	CreateSubscriptionOrder(ctx *gin.Context, userID string, req api.OrderCreateRequest) (*api.OrderCreateResponse, error)
	// ConfirmSubscriptionPayment 支付成功页/Webhook 未到时，按订单号向 Stripe 核验并发放会员
	ConfirmSubscriptionPayment(ctx *gin.Context, userID, siteID, transactionID string) error
	// FulfillOrderByTransactionID 无需登录，供支付成功回跳或 Webhook 补偿（核验 Stripe 已付款）
	FulfillOrderByTransactionID(ctx *gin.Context, siteID, transactionID string) error
	CancelSubscription(ctx context.Context, userID string, subscriptionID string, cancelAtPeriodEnd bool) error
	GetUserPurchases(ctx context.Context, userID string, siteID string, page, pageSize int) ([]*api.PurchaseRecord, int64, error)
}

// NewStripeService creates a new Stripe payment service
func NewStripeService(
	service *service.Service,
	conf *viper.Viper,
	client stripe.StripeClient,
	paymentConfigRepo peymentRep.PaymentConfigRepository,
	coinPackageRepo peymentRep.CoinPackageRepository,
	paymentTransactionRepo peymentRep.PaymentTransactionRepository,
	userCoinsRepository peymentRep.UserCoinsRepository,
	webhookEventRepository peymentRep.WebhookEventRepository,
	subscriptionPackageRepo peymentRep.SubscriptionPackageRepository,
	userRepository user.UserRepository,
	userSubscriptionRepo peymentRep.UserSubscriptionRepository,
	coinsService coins.CoinsService,
	trackingService *analytics.TrackingService,
) StripeService {
	return &stripeService{
		service:                 service,
		client:                  client,
		paymentConfigRepo:       paymentConfigRepo,
		coinPackageRepo:         coinPackageRepo,
		paymentTransactionRepo:  paymentTransactionRepo,
		userCoinsRepository:     userCoinsRepository,
		webhookEventRepository:  webhookEventRepository,
		coinsService:            coinsService,
		subscriptionPackageRepo: subscriptionPackageRepo,
		userRepository:          userRepository,
		userSubscriptionRepo:    userSubscriptionRepo,
		conf:                    conf,
		trackingService:         trackingService,
	}
}

type stripeService struct {
	service                 *service.Service
	conf                    *viper.Viper
	client                  stripe.StripeClient
	paymentConfigRepo       peymentRep.PaymentConfigRepository
	coinPackageRepo         peymentRep.CoinPackageRepository
	paymentTransactionRepo  peymentRep.PaymentTransactionRepository
	userCoinsRepository     peymentRep.UserCoinsRepository
	webhookEventRepository  peymentRep.WebhookEventRepository
	subscriptionPackageRepo peymentRep.SubscriptionPackageRepository
	userRepository          user.UserRepository
	userSubscriptionRepo    peymentRep.UserSubscriptionRepository
	coinsService            coins.CoinsService
	trackingService         *analytics.TrackingService
}

func (s *stripeService) GetAccountInfo(ctx context.Context, siteID string) (*api.PaymentAccountInfo, error) {
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, siteID, "stripe")
	if err != nil || config == nil || !config.IsActive || config.StripeSecretKey == "" {
		return nil, common.ErrPaymentProviderNotConfigured
	}

	// Return account info from database AccountInfo field
	accountInfo := &api.PaymentAccountInfo{
		AccountID: config.StripeAccountID,
	}

	// Parse AccountInfo JSONMap if available
	if config.AccountInfo != nil {
		if accountID, ok := config.AccountInfo["id"].(string); ok {
			accountInfo.AccountID = accountID
		}
		if email, ok := config.AccountInfo["email"].(string); ok {
			accountInfo.Email = email
		}
		if country, ok := config.AccountInfo["country"].(string); ok {
			accountInfo.Country = country
		}
	}

	return accountInfo, nil
}

func (s *stripeService) ConfTest(ctx context.Context, sk string) error {
	s.client.SetAPIKey(sk)
	return s.client.TestConf(sk)
}

func (s *stripeService) SaveConfig(ctx *gin.Context, config api.PaymentProviderConfig) error {
	if config.StripeConf.PublicKey == "" || config.StripeConf.SecretKey == "" {
		return common.ErrPaymentProviderNotConfigured
	}
	var err error
	orikey := s.client.GetAPIKey()
	s.client.SetAPIKey(config.StripeConf.SecretKey)
	defer func() {
		if err != nil {
			log.Error(ctx, "SaveConfig error: "+err.Error())
			s.client.SetAPIKey(orikey)
		}
	}()

	accountInfo, err := s.client.GetAccountInfo()
	if err != nil {
		log.Error(ctx, "GetAccountInfo error: "+err.Error())
		return err
	}

	existingConfig, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, config.SiteID, config.Provider)
	if err != nil {
		log.Error(ctx, "GetBySiteIDAndProvider error: "+err.Error())
		return err
	}

	now := time.Now()

	accountInfoJSON := model.JSONMap{
		"id":      accountInfo.ID,
		"email":   accountInfo.Email,
		"country": accountInfo.Country,
	}

	webhookUrl := s.conf.GetString("webhook.stripe.url") + "?siteId=" + config.SiteID
	webhookID, secret, err := s.client.GetShotPressEndpoint(webhookUrl)
	if err != nil {
		log.Error(ctx, "get short press url failed")
		return err
	}

	// 决定是否复用: 只有当 Stripe 上存在的 ID 与数据库中保存的 ID 一致时，才复用（因为我们有它的 Secret）
	shouldReuse := false
	if webhookID != "" {
		if existingConfig != nil && existingConfig.ProviderWebhookID == webhookID && existingConfig.ProviderWebhookSK != "" {
			shouldReuse = true
		}
	}

	if shouldReuse {
		log.Warning(ctx, "Webhook exists and matches DB, reusing: "+webhookID)
		// 这种情况下 secret 为空，但我们不需要更新它，因为数据库里的是对的
	} else {
		// 如果 Stripe 上存在但我们无法复用（意味着我们没有它的 Secret），先删除它，防止重复
		if webhookID != "" {
			log.Warning(ctx, "Webhook exists but ID mismatch or Secret missing, deleting: "+webhookID)
			err = s.client.DeleteWebhookEndpoint(webhookID)
			if err != nil {
				log.Error(ctx, "Failed to delete old webhook: "+err.Error())
				// 继续尝试创建，虽然可能会失败
			}
		}

		// 创建新的 Webhook
		webhookID, secret, err = s.client.SetWebhookEndpoint(webhookUrl)
		if err != nil {
			return err
		}
	}

	if existingConfig != nil {

		existingConfig.StripePublicKey = config.StripeConf.PublicKey
		existingConfig.StripeSecretKey = config.StripeConf.SecretKey
		existingConfig.StripeAccountID = accountInfo.ID
		existingConfig.IsActive = true
		existingConfig.LastVerifiedAt = &now
		existingConfig.VerificationStatus = 1
		existingConfig.AccountInfo = accountInfoJSON
		existingConfig.ProviderWebhookID = webhookID
		if secret != "" {
			existingConfig.ProviderWebhookSK = secret
		}
		existingConfig.EndPointUrl = webhookUrl

		return s.paymentConfigRepo.Update(ctx, existingConfig)
	} else {

		newConfig := &model.PaymentConfig{
			ConfigID:           uuid.New().String(),
			SiteID:             config.SiteID,
			Provider:           config.Provider,
			IsActive:           true,
			StripePublicKey:    config.StripeConf.PublicKey,
			StripeSecretKey:    config.StripeConf.SecretKey,
			StripeAccountID:    accountInfo.ID,
			LastVerifiedAt:     &now,
			VerificationStatus: 1,
			AccountInfo:        accountInfoJSON,
			ProviderWebhookID:  webhookID,
			ProviderWebhookSK:  secret,
			EndPointUrl:        webhookUrl,
		}

		return s.paymentConfigRepo.Create(ctx, newConfig)
	}
}

func (s *stripeService) CreateCoinPackage(ctx context.Context, req api.CoinPackageCreateRequest) (*api.CoinPackageCreateResponse, error) {
	// No longer require Stripe key to create coin package
	// Creators can set up packages first, then configure Stripe later
	packageID := uuid.NewString()

	/*****************No longer set price and product, directly set the price when paying
	productMetadata := map[string]string{
		"package_id":   packageID,
		"site_id":      req.SiteID,
		"coin_amount":  strconv.Itoa(req.CoinAmount),
		"product_type": "coin",
	}

	productID, err := s.client.CreateProduct(req.Name, req.Description, productMetadata)
	if err != nil {
		return nil, common.ErrCreateStripeProduct
	}

	priceInCents := int64(req.Price * 100)

	priceMetadata := map[string]string{
		"package_id":  packageID,
		"coin_amount": strconv.Itoa(req.CoinAmount),
	}

	priceID, err := s.client.CreatePrice(priceInCents, "USD", productID, priceMetadata)
	if err != nil {
		delErr := s.client.DeleteProduct(productID)
		if delErr != nil {
			fmt.Println("Failed to delete Stripe product after price creation error: %v", delErr)
		}
		return nil, common.ErrCreateStripePrice
	}
	****************/

	coinPackage := &model.CoinPackage{
		PackageID:          packageID,
		SiteID:             req.SiteID,
		Name:               req.Name,
		Description:        req.Description,
		Features:           req.Features,
		CoinAmount:         req.CoinAmount,
		Price:              req.Price.Cents(),
		OriginalPrice:      req.OriginalPrice.Cents(),
		Currency:           "USD",
		DiscountPercentage: req.DiscountPercentage,
		// StripePriceID:      priceID,
		Status: 1,
	}

	err := s.coinPackageRepo.Create(ctx, coinPackage)
	if err != nil {
		return nil, err
	}

	return &api.CoinPackageCreateResponse{
		PackageID: packageID,
	}, nil
}

// CreateOrder creates a payment order for either coin packages or subscriptions
func (s *stripeService) CreateOrder(ctx *gin.Context, userID string, req api.OrderCreateRequest) (*api.OrderCreateResponse, error) {
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, req.SiteID, "stripe")
	if err != nil {
		return nil, err
	}
	if config == nil || !config.IsActive || config.StripeSecretKey == "" {
		return nil, common.ErrPaymentProviderNotConfigured
	}

	s.client.SetAPIKey(config.StripeSecretKey)

	transactionID := uuid.NewString()

	switch req.OrderType {
	case api.OrderTypeCoinPackage:
		return s.createCoinPackageOrder(ctx, req, userID, transactionID, config)
	case api.OrderTypeSubscription:
		return nil, errors.New("subscription orders not implemented yet")
	default:
		return nil, common.ErrBadRequest
	}
}

// createCoinPackageOrder handles the creation of coin package orders
func (s *stripeService) createCoinPackageOrder(
	ctx *gin.Context,
	req api.OrderCreateRequest,
	userID string,
	transactionID string,
	config *model.PaymentConfig,
) (*api.OrderCreateResponse, error) {

	coinPackage, err := s.coinPackageRepo.GetByPackageID(ctx, req.PackageID)
	if err != nil {
		return nil, err
	}

	if coinPackage == nil {
		return nil, common.ErrNotFound
	}

	if coinPackage.Status != 1 {
		return nil, common.ErrResourceNotActive
	}

	if coinPackage.SiteID != req.SiteID {
		return nil, common.ErrBadRequest
	}

	currency := coinPackage.Currency
	if req.Currency != "" {
		currency = req.Currency
	}
	metadata := map[string]string{
		"site_id":        req.SiteID,
		"user_id":        userID,
		"package_type":   "coin",
		"transaction_id": transactionID,
		"package_id":     coinPackage.PackageID,
		"coin_amount":    strconv.Itoa(coinPackage.CoinAmount),
	}

	var checkoutSessionID, checkoutURL, paymentIntentID string
	var createErr error

	if req.ReturnURL == "" {
		req.ReturnURL = s.paymentSuccessURL(s.conf.GetString("webhook.stripe.sucess_url"), req.SiteID, transactionID)

	} else {
		req.ReturnURL = s.paymentSuccessURL(req.ReturnURL, req.SiteID, transactionID)
	}
	if req.CancelURL == "" {
		// req.CancelURL = s.paymentCancelURL("https://shortpress.com/payment/cancel/", req.SiteID, transactionID)
		req.CancelURL = s.paymentCancelURL(s.conf.GetString("webhook.stripe.cancel_url"), req.SiteID, transactionID)
	} else {
		req.CancelURL = s.paymentCancelURL(req.CancelURL, req.SiteID, transactionID)
	}

	// checkoutSessionID, checkoutURL, createErr = s.client.CreateCheckoutSession(
	// 	coinPackage.StripePriceID,
	// 	req.ReturnURL,
	// 	req.CancelURL,
	// 	metadata,
	// )
	// if createErr != nil {
	// 	return nil, common.ErrCreateStripeCheckoutSession
	// }

	checkoutSessionID, checkoutURL, createErr = s.client.CreateCheckoutSessionWithPayment(coinPackage.Name, coinPackage.Description, coinPackage.Price, coinPackage.Currency,
		req.ReturnURL,
		req.CancelURL,
		metadata,
	)
	if createErr != nil {
		log.Error(ctx, " create checkout session error: "+createErr.Error())
		return nil, common.ErrCreateStripeCheckoutSession
	}

	paymentIntentID = checkoutSessionID

	snapshot := model.JSONMap{
		"package_id":          coinPackage.PackageID,
		"name":                coinPackage.Name,
		"description":         coinPackage.Description,
		"coin_amount":         coinPackage.CoinAmount,
		"price":               coinPackage.Price,
		"currency":            currency,
		"original_price":      coinPackage.OriginalPrice,
		"discount_percentage": coinPackage.DiscountPercentage,
	}
	if len(req.TrackingContext) > 0 {
		snapshot["tracking_context"] = req.TrackingContext
	}
	snapshot = analytics.MergeMetaIntoSnapshot(snapshot, req.Meta)
	tx := &model.PaymentTransaction{
		TransactionID:     transactionID,
		OrderID:           transactionID, // Normally requires an order table, currently the fund flow table temporarily replaces the Order table
		UserID:            userID,
		SiteID:            req.SiteID,
		Amount:            coinPackage.Price,
		Currency:          currency,
		Provider:          "stripe",
		ProviderPaymentID: paymentIntentID,
		PaymentType:       model.PaymentTypeCoinPackage,
		Status:            model.PaymentStatusPending,
		RelatedID:         coinPackage.PackageID,
		RelatedType:       model.RelatedTypeCoinPackage,
		Snapshot:          snapshot,
	}

	err = s.paymentTransactionRepo.Create(ctx, tx)
	if err != nil {
		return nil, err
	}

	response := &api.OrderCreateResponse{
		OrderID:       transactionID,
		SuccessURL:    req.ReturnURL,
		CancelURL:     req.CancelURL,
		PaymentStatus: "pending",
	}

	if checkoutURL != "" {
		response.CheckoutURL = checkoutURL
	}

	return response, nil
}

func (s *stripeService) paymentSuccessURL(baseUrl string, siteID string, orderID string) string {
	sep := "?"
	if strings.Contains(baseUrl, "?") {
		if strings.HasSuffix(baseUrl, "?") || strings.HasSuffix(baseUrl, "&") {
			sep = ""
		} else {
			sep = "&"
		}
	}
	return fmt.Sprintf("%s%ssiteId=%s&orderId=%s", baseUrl, sep, siteID, orderID)
}

func (s *stripeService) paymentCancelURL(baseUrl string, siteID string, orderID string) string {
	sep := "?"
	if strings.Contains(baseUrl, "?") {
		if strings.HasSuffix(baseUrl, "?") || strings.HasSuffix(baseUrl, "&") {
			sep = ""
		} else {
			sep = "&"
		}
	}
	return fmt.Sprintf("%s%ssiteId=%s&orderId=%s", baseUrl, sep, siteID, orderID)
}

// HandleWebhook processes Stripe webhook events
func (s *stripeService) HandleWebhook(ctx *gin.Context) error {
	// Read request body
	payload, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		s.service.Logger().Error("Failed to read request body", zap.Error(err))
		return err
	}
	// Get signature from header
	signature := ctx.GetHeader("Stripe-Signature")
	if signature == "" {
		return errors.New("stripe signature missing")
	}

	// siteId 优先来自 query；未带 query 时从 metadata 或 cs_ 对应本地订单反查
	siteID := ctx.Query("siteId")
	peekSiteID, peekSessionID := peekStripeWebhookPayload(payload)
	if siteID == "" {
		siteID = peekSiteID
	}
	if siteID == "" && peekSessionID != "" {
		if tx, _ := s.paymentTransactionRepo.GetByProviderPaymentID(ctx, "stripe", peekSessionID); tx != nil {
			siteID = tx.SiteID
		}
	}
	if siteID == "" {
		return errors.New("site ID missing: webhook URL 需带 ?siteId= 或在 Checkout metadata 中设置 site_id")
	}

	// Get payment config for the site
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, siteID, "stripe")
	if err != nil {
		log.Error(ctx, "Failed to get payment config: "+err.Error())
		return err
	}
	if config == nil {
		log.Error(ctx, "Payment provider config not found or not active: "+siteID)
		return common.ErrPaymentProviderNotConfigured
	}

	// Validate webhook signature //Test SK
	if s.conf.GetString("env") == "local" {
		s.service.Logger().Debug("local env, use test secret key")
		config.ProviderWebhookSK = s.conf.GetString("webhook.stripe.test_secret")
	}

	if config == nil || config.ProviderWebhookSK == "" {
		log.Error(ctx, "Payment provider config not found or not active: "+siteID)
		return common.ErrPaymentProviderNotConfigured
	}

	event, err := s.client.ValidateWebhookSignature(payload, signature, config.ProviderWebhookSK)
	if err != nil {
		log.Error(ctx, "Failed to validate webhook signature: "+err.Error())
		return err
	}

	s.service.Logger().Info("Stripe webhook event",
		zap.String("type", string(event.Type)),
		zap.String("siteId", siteID),
		zap.String("eventId", event.ID),
	)

	// Process the webhook based on event type

	var processErr error
	switch event.Type {
	case "checkout.session.completed":
		processErr = s.handleCheckoutSessionCompleted(ctx, siteID, &event)
	case "checkout.session.expired":
		processErr = s.handleCheckoutSessionExpired(ctx, &event)
	case "charge.refunded":
		processErr = s.handleChargeRefunded(ctx, &event)
	case "payment_intent.succeeded":
		processErr = s.handlePaymentIntentSucceeded(ctx, siteID, &event)
	case "checkout.session.async_payment_succeeded":
		processErr = s.handleCheckoutSessionCompleted(ctx, siteID, &event)
	case "invoice.paid", "invoice.payment_succeeded":
		// 与 Stripe 文档一致：在线扣款成功时 paid 与 payment_succeeded 可能都会投递，业务幂等靠 invoice.id
		processErr = s.handleInvoicePaid(ctx, siteID, &event)
		// case  "customer.subscription.created":
		// 	processErr = s.handleSubscriptionCreated(ctx, siteID, &event)
	case "customer.subscription.updated":
		processErr = s.handleSubscriptionUpdated(ctx, siteID, &event)
	case "customer.subscription.deleted":
		processErr = s.handleSubscriptionDeleted(ctx, siteID, &event)
	case "charge.succeeded":
		processErr = s.handleChargeSucceeded(ctx, siteID, &event)
	default:
		log.Error(ctx, "Unhandled event type: "+string(event.Type))
	}

	// Create webhook event record
	// result := 2
	if processErr != nil {
		log.Error(ctx, "Webhook event processing failed: "+processErr.Error())
	}
	// webhookEvent := &model.WebhookEvent{
	// 	EventID:    event.ID,
	// 	WebhookID:  config.ProviderWebhookID,
	// 	Provider:   "stripe",
	// 	EventType:  string(event.Type),
	// 	// Payload:    string(event.Data.Raw),//Currently save raw data directly, data is too large. If you need to query data, query by ID
	// 	Result:     result,
	// }

	// // Save webhook event to database TODO: If the database pressure is relatively high later, consider closing and reducing event saving
	// if err := s.webhookEventRepository.Create(ctx, webhookEvent); err != nil {
	// 	s.service.Logger().Error("Failed to save webhook event", zap.Error(err))
	// }

	return processErr
}

// handleCheckoutSessionCompleted processes checkout.session.completed events
func (s *stripeService) handleCheckoutSessionCompleted(ctx *gin.Context, siteID string, event *goStripe.Event) error {
	var session goStripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return err
	}
	return s.fulfillCheckoutSessionCompleted(ctx, siteID, &session)
}

// ConfirmSubscriptionPayment 客户端支付成功回跳后调用，补偿 Webhook 未送达或未处理的情况
func (s *stripeService) ConfirmSubscriptionPayment(ctx *gin.Context, userID, siteID, transactionID string) error {
	if transactionID == "" {
		return errors.New("orderId is required")
	}

	transaction, err := s.paymentTransactionRepo.GetByTransactionID(ctx, transactionID)
	if err != nil {
		return err
	}
	if transaction == nil {
		return errors.New("transaction not found")
	}
	if transaction.UserID != userID || transaction.SiteID != siteID {
		return common.ErrUnauthorized
	}
	if transaction.PaymentType != model.PaymentTypeSubscription {
		return errors.New("not a subscription transaction")
	}
	if transaction.Status == model.PaymentStatusSuccess && snapshotBool(transaction.Snapshot, "one_time_membership_granted") {
		return nil
	}

	return s.FulfillOrderByTransactionID(ctx, siteID, transactionID)
}

// FulfillOrderByTransactionID 根据本地订单号向 Stripe 核验 Checkout 是否已付款，并更新 status=2、发放会员
func (s *stripeService) FulfillOrderByTransactionID(ctx *gin.Context, siteID, transactionID string) error {
	if transactionID == "" || siteID == "" {
		return errors.New("siteId and orderId are required")
	}

	transaction, err := s.paymentTransactionRepo.GetByTransactionID(ctx, transactionID)
	if err != nil {
		return err
	}
	if transaction == nil {
		return errors.New("transaction not found")
	}
	if transaction.SiteID != siteID {
		return fmt.Errorf("transaction site mismatch")
	}

	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, siteID, "stripe")
	if err != nil {
		return err
	}
	if config == nil || !config.IsActive || config.StripeSecretKey == "" {
		return common.ErrPaymentProviderNotConfigured
	}
	s.client.SetAPIKey(config.StripeSecretKey)

	sessionID := transaction.ProviderPaymentID
	if sessionID == "" {
		return errors.New("checkout session not found for transaction")
	}
	session, err := s.client.RetrieveSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to retrieve checkout session: %w", err)
	}
	if string(session.PaymentStatus) != string(goStripe.CheckoutSessionPaymentStatusPaid) {
		return errors.New("payment not completed")
	}

	return s.fulfillCheckoutSessionCompleted(ctx, siteID, session)
}

func (s *stripeService) fulfillCheckoutSessionCompleted(ctx *gin.Context, siteID string, session *goStripe.CheckoutSession) error {
	if session == nil {
		return errors.New("checkout session is nil")
	}

	// Webhook 事件体里的 metadata 可能不完整，向 Stripe 拉取完整 Session
	if session.Metadata == nil || session.Metadata["transaction_id"] == "" {
		lookupSiteID := siteID
		if pending, _ := s.paymentTransactionRepo.GetByProviderPaymentID(ctx, "stripe", session.ID); pending != nil {
			lookupSiteID = pending.SiteID
		} else if session.Metadata != nil && session.Metadata["site_id"] != "" {
			lookupSiteID = session.Metadata["site_id"]
		}
		config, cfgErr := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, lookupSiteID, "stripe")
		if cfgErr == nil && config != nil && config.StripeSecretKey != "" {
			s.client.SetAPIKey(config.StripeSecretKey)
			full, retrieveErr := s.client.RetrieveSession(session.ID)
			if retrieveErr == nil && full != nil {
				session = full
			}
		}
	}

	transaction, err := s.resolveCheckoutTransaction(ctx, session)
	if err != nil {
		return err
	}
	if transaction == nil {
		return errors.New("transaction not found for checkout session")
	}

	// 业务处理一律以订单上的 site_id 为准，避免 Webhook URL 上 siteId 参数不准导致第二次支付被跳过
	if siteID != transaction.SiteID {
		log.Warning(ctx, fmt.Sprintf("Webhook siteId adjusted: callback=%s, transaction=%s, session=%s",
			siteID, transaction.SiteID, session.ID))
	}
	siteID = transaction.SiteID

	payerEmail := payutil.StripeCheckoutPayerEmail(session)

	if transaction.Status == model.PaymentStatusSuccess {
		if transaction.PaymentType == model.PaymentTypeSubscription &&
			isOneTimeSubscriptionCheckout(transaction, stripeSubscriptionIDFromSession(session)) &&
			!snapshotBool(transaction.Snapshot, "one_time_membership_granted") {
			// 已成功但未发放会员（例如上次 UpdateBalance 失败导致回滚不完整时的补偿，或历史脏数据）
			subid := stripeSubscriptionIDFromSession(session)
			return s.service.Tx().Transaction(ctx, func(ctx context.Context) error {
				return s.processSubscriptionPayment(ctx, subid, transaction)
			})
		}
		if payerEmail != "" && transaction.PayerEmail == "" {
			transaction.PayerEmail = payerEmail
			if err := s.paymentTransactionRepo.Update(ctx, transaction); err != nil {
				log.Warning(ctx, "failed to backfill payer email: "+err.Error())
			}
		}
		return nil
	}

	var claimed bool
	err = s.service.Tx().Transaction(ctx, func(ctx context.Context) error {
		// 原子幂等：只有第一个把 pending -> success 的请求，才继续发币/发会员。
		ok, err := s.paymentTransactionRepo.MarkSuccessIfPending(ctx, transaction.TransactionID, session.ID, payerEmail)
		if err != nil {
			return err
		}
		if !ok {
			// 并发 webhook 已被其他请求处理，直接跳过后续业务。
			return nil
		}
		claimed = true
		transaction.Status = model.PaymentStatusSuccess
		transaction.ProviderPaymentID = session.ID
		if payerEmail != "" {
			transaction.PayerEmail = payerEmail
		}

		switch transaction.PaymentType {
		case model.PaymentTypeCoinPackage:
			return s.processCoinPackagePayment(ctx, transaction)
		case model.PaymentTypeSubscription:
			subid := stripeSubscriptionIDFromSession(session)
			return s.processSubscriptionPayment(ctx, subid, transaction)
		default:
			return fmt.Errorf("unsupported payment type: %d", transaction.PaymentType)
		}
	})
	if err != nil {
		log.Error(ctx, "Transaction failed: "+err.Error())
		return err
	}
	if !claimed {
		return nil
	}

	if s.trackingService != nil {
		go func() {
			trackingCtx := context.Background()
			if trackErr := s.trackingService.TrackPurchase(trackingCtx, transaction); trackErr != nil {
				fmt.Printf("打点事件发送失败: %v\n", trackErr)
			}
		}()
	}

	return nil
}

func stripeSubscriptionIDFromSession(session *goStripe.CheckoutSession) string {
	if session.Subscription != nil {
		return session.Subscription.ID
	}
	return ""
}

// resolveCheckoutTransaction 优先用 Checkout Session ID 定位当前这笔订单（避免 metadata 指向旧单）
func (s *stripeService) resolveCheckoutTransaction(ctx context.Context, session *goStripe.CheckoutSession) (*model.PaymentTransaction, error) {
	if session == nil {
		return nil, errors.New("checkout session is nil")
	}

	// 创建订单时 provider_payment_id = cs_xxx，与本次 Session 一一对应
	if session.ID != "" {
		tx, err := s.paymentTransactionRepo.GetByProviderPaymentID(ctx, "stripe", session.ID)
		if err != nil {
			return nil, err
		}
		if tx != nil {
			return tx, nil
		}
	}

	if session.Metadata != nil {
		if tid := session.Metadata["transaction_id"]; tid != "" {
			tx, err := s.paymentTransactionRepo.GetByTransactionID(ctx, tid)
			if err != nil {
				return nil, err
			}
			if tx != nil {
				return tx, nil
			}
		}
	}

	return nil, nil
}

// handlePaymentIntentSucceeded 一次性 Checkout（mode=payment）在部分场景下主要收到该事件而非 session.completed
func (s *stripeService) handlePaymentIntentSucceeded(ctx *gin.Context, siteID string, event *goStripe.Event) error {
	var pi goStripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		return err
	}

	transactionID := ""
	if pi.Metadata != nil {
		transactionID = pi.Metadata["transaction_id"]
	}

	var transaction *model.PaymentTransaction
	var err error
	if transactionID != "" {
		transaction, err = s.paymentTransactionRepo.GetByTransactionID(ctx, transactionID)
		if err != nil {
			return err
		}
	}
	if transaction == nil {
		return nil
	}

	if transaction.PaymentType != model.PaymentTypeSubscription && transaction.PaymentType != model.PaymentTypeCoinPackage {
		return nil
	}

	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, transaction.SiteID, "stripe")
	if err != nil {
		return err
	}
	if config == nil || !config.IsActive || config.StripeSecretKey == "" {
		return common.ErrPaymentProviderNotConfigured
	}
	s.client.SetAPIKey(config.StripeSecretKey)

	sessionID := transaction.ProviderPaymentID
	if sessionID == "" {
		return nil
	}
	session, err := s.client.RetrieveSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to retrieve checkout session for payment_intent: %w", err)
	}
	if string(session.PaymentStatus) != string(goStripe.CheckoutSessionPaymentStatusPaid) {
		return nil
	}

	return s.fulfillCheckoutSessionCompleted(ctx, transaction.SiteID, session)
}

// handleChargeSucceeded 兜底：部分环境只投递 charge.succeeded
func (s *stripeService) handleChargeSucceeded(ctx *gin.Context, siteID string, event *goStripe.Event) error {
	var charge goStripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
		return err
	}
	if charge.PaymentIntent == nil || charge.PaymentIntent.ID == "" {
		return nil
	}

	lookupSiteID := siteID
	if lookupSiteID == "" {
		lookupSiteID = peekSiteIDFromCharge(event)
	}
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, lookupSiteID, "stripe")
	if err != nil {
		return err
	}
	if config == nil || !config.IsActive || config.StripeSecretKey == "" {
		return common.ErrPaymentProviderNotConfigured
	}
	s.client.SetAPIKey(config.StripeSecretKey)

	pi, err := s.client.GetPaymentIntent(charge.PaymentIntent.ID)
	if err != nil {
		return fmt.Errorf("failed to get payment intent for charge: %w", err)
	}

	transactionID := ""
	if pi.Metadata != nil {
		transactionID = pi.Metadata["transaction_id"]
	}

	var transaction *model.PaymentTransaction
	if transactionID != "" {
		transaction, err = s.paymentTransactionRepo.GetByTransactionID(ctx, transactionID)
		if err != nil {
			return err
		}
	}
	if transaction == nil {
		return nil
	}

	if transaction.ProviderPaymentID == "" {
		return nil
	}
	session, err := s.client.RetrieveSession(transaction.ProviderPaymentID)
	if err != nil {
		return fmt.Errorf("failed to retrieve checkout session for charge: %w", err)
	}
	if string(session.PaymentStatus) != string(goStripe.CheckoutSessionPaymentStatusPaid) {
		return nil
	}

	return s.fulfillCheckoutSessionCompleted(ctx, transaction.SiteID, session)
}

// peekStripeWebhookPayload 从 Webhook 原始 JSON 预读 site_id 与 checkout session id（验签前用于定位站点配置）
func peekStripeWebhookPayload(payload []byte) (siteID, checkoutSessionID string) {
	var envelope struct {
		Type string `json:"type"`
		Data struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", ""
	}

	var obj struct {
		ID       string            `json:"id"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(envelope.Data.Object, &obj); err != nil {
		return "", ""
	}
	if obj.Metadata != nil {
		siteID = obj.Metadata["site_id"]
	}
	if strings.HasPrefix(string(envelope.Type), "checkout.session") {
		checkoutSessionID = obj.ID
	}
	return siteID, checkoutSessionID
}

func peekSiteIDFromCharge(event *goStripe.Event) string {
	var charge goStripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
		return ""
	}
	if charge.Metadata != nil && charge.Metadata["site_id"] != "" {
		return charge.Metadata["site_id"]
	}
	return ""
}

// processCoinPackagePayment handles successful coin package payments
func (s *stripeService) processCoinPackagePayment(ctx context.Context, transaction *model.PaymentTransaction) error {
	err := s.addCoinsToUserAccount(ctx, transaction)
	if err != nil {
		return err
	}
	return nil
}

// addCoinsToUserAccount adds coins to a user's account after a successful payment
func (s *stripeService) addCoinsToUserAccount(ctx context.Context, transaction *model.PaymentTransaction) error {
	if transaction.Status != model.PaymentStatusSuccess {
		return errors.New("cannot add coins for non-successful transaction")
	}

	coinAmount, ok := transaction.Snapshot["coin_amount"]
	if !ok {
		return errors.New("invalid coin amount in transaction snapshot")
	}

	// Convert to int for coin operations
	coins := int(coinAmount.(float64))
	if coins <= 0 {
		return errors.New("coin amount must be positive")
	}
	_, err := s.coinsService.AddCoinsByPayment(ctx, coins, "stripe", transaction)
	if err != nil {
		return err
	}
	return nil
}

// handleCheckoutSessionExpired processes checkout.session.expired events
func (s *stripeService) handleCheckoutSessionExpired(ctx *gin.Context, event *goStripe.Event) error {
	var session goStripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return err
	}

	transactionID := session.Metadata["transaction_id"]
	if transactionID == "" {
		return errors.New("missing transaction ID in metadata")
	}

	// Search and update transaction status to expired
	tx, err := s.paymentTransactionRepo.GetByTransactionID(ctx, transactionID)
	if err != nil {
		return err
	}

	if tx == nil || tx.Status != model.PaymentStatusPending {
		log.Error(ctx, "Transaction not found or already processed: "+transactionID)
		return nil
	}

	tx.Status = model.PaymentStatusExpired // Add this status constant
	return s.paymentTransactionRepo.Update(ctx, tx)
}

// handleChargeRefunded processes charge.refunded events
func (s *stripeService) handleChargeRefunded(ctx context.Context, event *goStripe.Event) error {
	var charge goStripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
		return err
	}

	// Find related transaction
	trans, err := s.paymentTransactionRepo.GetByProviderPaymentID(ctx, "stripe", charge.PaymentIntent.ID)
	if err != nil || trans == nil {
		return err
	}

	// Update refund amount
	trans.RefundAmount = &charge.AmountRefunded

	// Determine status based on whether refund amount equals total amount
	if charge.AmountRefunded == charge.Amount {
		trans.Status = model.PaymentStatusRefunded
	} else if charge.AmountRefunded > 0 {
		trans.Status = model.PaymentStatusPartialRefunded
	}

	// Update transaction record
	if err := s.paymentTransactionRepo.Update(ctx, trans); err != nil {
		return err
	}

	// If it's a coin package purchase, handle coin deduction
	if trans.PaymentType == model.PaymentTypeCoinPackage {
		return s.handleCoinPackageRefund(ctx, trans, charge.AmountRefunded)
	}

	return nil
}

// handleCoinPackageRefund processes coin package refunds
func (s *stripeService) handleCoinPackageRefund(ctx context.Context, trans *model.PaymentTransaction, refundAmount int64) error {
	// Get coin amount
	coinAmount, ok := trans.Snapshot["coin_amount"]
	if !ok {
		return errors.New("invalid coin amount in transaction snapshot")
	}

	coins := int(coinAmount.(float64))
	if coins <= 0 {
		log.Warning(ctx.(*gin.Context), fmt.Sprintf("Coin refund amount is 0: coins=%d", coins))
		return nil
	}

	// TODO Deduct coins

	return nil
}

// GetConfigInfo retrieves payment configuration information for a site
func (s *stripeService) GetConfigInfo(ctx *gin.Context, siteID string) (*api.PaymentConfigInfoResponse, error) {
	response := &api.PaymentConfigInfoResponse{}

	// Get Stripe configuration
	stripeConfig, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, siteID, "stripe")
	if err != nil {
		log.Warning(ctx, "Failed to get Stripe config: "+err.Error())
		// Continue instead of returning error to try to get PayPal config as well
	} else if stripeConfig != nil && stripeConfig.IsActive {
		response.Stripe = &api.StripeConfigInfo{
			PublicKey: stripeConfig.StripePublicKey,
			SecretKey: stripeConfig.StripeSecretKey,
			IsSandbox: stripeConfig.IsSandbox,
		}
	}

	return response, nil
}

// CreateSubscriptionOrder creates a new subscription order
func (s *stripeService) CreateSubscriptionOrder(ctx *gin.Context, userID string, req api.OrderCreateRequest) (*api.OrderCreateResponse, error) {
	// Validate request
	if req.PackageID == "" || req.SiteID == "" {
		return nil, errors.New("packageId and siteId are required")
	}

	// Get the subscription package
	pkg, err := s.subscriptionPackageRepo.GetByPackageID(ctx, req.PackageID)
	if err != nil {
		return nil, err
	}

	if pkg == nil {
		return nil, errors.New("subscription package not found")
	}

	if pkg.Status != 1 {
		return nil, errors.New("subscription package is not active")
	}

	if pkg.SiteID != req.SiteID {
		return nil, errors.New("subscription package does not belong to this site")
	}

	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, req.SiteID, "stripe")
	if err != nil {
		return nil, err
	}
	if config == nil || !config.IsActive || config.StripeSecretKey == "" {
		return nil, common.ErrPaymentProviderNotConfigured
	}
	s.client.SetAPIKey(config.StripeSecretKey)

	// 已恢复为 Stripe 订阅模式：默认创建 recurring checkout（sub_ 自动续订）。
	// 如后续需要重新启用“一次性购买订阅（one_time）”方案，可恢复下面被注释的代码块。
	return s.createLegacyRecurringSubscriptionOrder(ctx, userID, req, pkg)

	// // 一次性购买订阅（历史临时方案）
	// // 当前仍有有效的 Stripe 自动续订（sub_）则禁止重复下单；其余用户（含已过期老订阅）走一次性 Checkout
	// legacySub, err := s.userSubscriptionRepo.GetActiveLegacyStripeRecurringSubscription(ctx, userID, req.SiteID)
	// if err != nil {
	// 	return nil, err
	// }
	// if legacySub != nil {
	// 	stripeActive, stripeErr := s.client.IsSubscriptionActive(legacySub.ProviderSubscriptionID)
	// 	if stripeErr != nil {
	// 		s.service.Logger().Warn("Failed to verify legacy Stripe subscription status, blocking new order",
	// 			zap.Error(stripeErr),
	// 			zap.String("subscriptionID", legacySub.ProviderSubscriptionID),
	// 			zap.String("userID", userID),
	// 		)
	// 		return nil, errors.New("user already has an active subscription")
	// 	}
	// 	if stripeActive {
	// 		return nil, errors.New("user already has an active subscription")
	// 	}
	// }
	// return s.createOneTimeSubscriptionOrder(ctx, userID, req, pkg)
}

// createOneTimeSubscriptionOrder 与购金币相同的一次性 Checkout，每周期由用户主动购买
func (s *stripeService) createOneTimeSubscriptionOrder(ctx *gin.Context, userID string, req api.OrderCreateRequest, pkg *model.SubscriptionPackage) (*api.OrderCreateResponse, error) {
	transactionID := uuid.New().String()

	var successURL, cancelURL string
	if req.ReturnURL == "" {
		successURL = s.paymentSuccessURL(s.conf.GetString("webhook.stripe.sucess_url"), req.SiteID, transactionID)
	} else {
		successURL = s.paymentSuccessURL(req.ReturnURL, req.SiteID, transactionID)
	}
	if req.CancelURL == "" {
		cancelURL = s.paymentCancelURL(s.conf.GetString("webhook.stripe.cancel_url"), req.SiteID, transactionID)
	} else {
		cancelURL = s.paymentCancelURL(req.CancelURL, req.SiteID, transactionID)
	}

	metadata := map[string]string{
		"site_id":        req.SiteID,
		"user_id":        userID,
		"package_type":   "subscription",
		"transaction_id": transactionID,
		"package_id":     pkg.PackageID,
		"interval":       pkg.Interval,
		"billing_mode":   "one_time",
	}

	sessionID, checkoutURL, err := s.client.CreateCheckoutSessionWithPayment(
		pkg.Name,
		pkg.Description,
		pkg.Price,
		pkg.Currency,
		successURL,
		cancelURL,
		metadata,
	)
	if err != nil {
		return nil, err
	}

	snapshot := model.JSONMap{
		"package_id":                  pkg.PackageID,
		"name":                        pkg.Name,
		"description":                 pkg.Description,
		"interval":                    pkg.Interval,
		"price":                       pkg.Price,
		"currency":                    pkg.Currency,
		"original_price":              pkg.OriginalPrice,
		"discount_percentage":         pkg.DiscountPercentage,
		"coins":                       pkg.Coins,
		"coin_amount":                 float64(pkg.Coins),
		"rights":                      pkg.Rights,
		"stripe_subscription_billing": "one_time",
	}
	if len(req.TrackingContext) > 0 {
		snapshot["tracking_context"] = req.TrackingContext
	}
	snapshot = analytics.MergeMetaIntoSnapshot(snapshot, req.Meta)

	tx := &model.PaymentTransaction{
		TransactionID:     transactionID,
		OrderID:           transactionID,
		UserID:            userID,
		SiteID:            req.SiteID,
		Amount:            pkg.Price,
		Currency:          pkg.Currency,
		Provider:          "stripe",
		ProviderPaymentID: sessionID,
		PaymentType:       model.PaymentTypeSubscription,
		Status:            model.PaymentStatusPending,
		RelatedID:         pkg.PackageID,
		RelatedType:       model.RelatedTypeSubscription,
		Snapshot:          snapshot,
	}
	if err := s.paymentTransactionRepo.Create(ctx, tx); err != nil {
		return nil, err
	}

	return &api.OrderCreateResponse{
		OrderID:       transactionID,
		CheckoutURL:   checkoutURL,
		SuccessURL:    successURL,
		CancelURL:     cancelURL,
		PaymentStatus: "pending",
	}, nil
}

// createLegacyRecurringSubscriptionOrder 老用户 Stripe 自动续订 Checkout（保持原有逻辑）
func (s *stripeService) createLegacyRecurringSubscriptionOrder(ctx *gin.Context, userID string, req api.OrderCreateRequest, pkg *model.SubscriptionPackage) (*api.OrderCreateResponse, error) {
	transactionID := uuid.New().String()

	var successURL, cancelURL string
	if req.ReturnURL == "" {
		successURL = s.paymentSuccessURL(s.conf.GetString("webhook.stripe.sucess_url"), req.SiteID, transactionID)
	} else {
		successURL = s.paymentSuccessURL(req.ReturnURL, req.SiteID, transactionID)
	}
	if req.CancelURL == "" {
		cancelURL = s.paymentCancelURL(s.conf.GetString("webhook.stripe.cancel_url"), req.SiteID, transactionID)
	} else {
		cancelURL = s.paymentCancelURL(req.CancelURL, req.SiteID, transactionID)
	}

	metadata := map[string]string{
		"site_id":        req.SiteID,
		"user_id":        userID,
		"package_type":   "subscription",
		"transaction_id": transactionID,
		"package_id":     pkg.PackageID,
		"interval":       pkg.Interval,
		"billing_mode":   "recurring",
	}

	var stripeInterval string
	switch pkg.Interval {
	case "week":
		stripeInterval = "week"
	case "month":
		stripeInterval = "month"
	case "year":
		stripeInterval = "year"
	default:
		return nil, errors.New("invalid subscription interval")
	}

	sessionID, checkoutURL, err := s.client.CreateSubscriptionCheckoutSession(
		pkg.Name,
		pkg.Description,
		pkg.Price,
		pkg.Currency,
		stripeInterval,
		successURL,
		cancelURL,
		metadata,
	)
	if err != nil {
		return nil, err
	}

	snapshot := model.JSONMap{
		"package_id":                  pkg.PackageID,
		"name":                        pkg.Name,
		"description":                 pkg.Description,
		"interval":                    pkg.Interval,
		"price":                       pkg.Price,
		"currency":                    pkg.Currency,
		"original_price":              pkg.OriginalPrice,
		"discount_percentage":         pkg.DiscountPercentage,
		"coins":                       pkg.Coins,
		"coin_amount":                 float64(pkg.Coins),
		"rights":                      pkg.Rights,
		"stripe_subscription_billing": "initial",
	}
	snapshot = analytics.MergeMetaIntoSnapshot(snapshot, req.Meta)

	tx := &model.PaymentTransaction{
		TransactionID:     transactionID,
		OrderID:           transactionID,
		UserID:            userID,
		SiteID:            req.SiteID,
		Amount:            pkg.Price,
		Currency:          pkg.Currency,
		Provider:          "stripe",
		ProviderPaymentID: sessionID,
		PaymentType:       model.PaymentTypeSubscription,
		Status:            model.PaymentStatusPending,
		RelatedID:         pkg.PackageID,
		RelatedType:       model.RelatedTypeSubscription,
		Snapshot:          snapshot,
	}
	if err := s.paymentTransactionRepo.Create(ctx, tx); err != nil {
		return nil, err
	}

	return &api.OrderCreateResponse{
		OrderID:       transactionID,
		CheckoutURL:   checkoutURL,
		SuccessURL:    successURL,
		CancelURL:     cancelURL,
		PaymentStatus: "pending",
	}, nil
}

// processSubscriptionPayment handles successful subscription payments
func (s *stripeService) processSubscriptionPayment(ctx context.Context, subid string, transaction *model.PaymentTransaction) error {
	if isOneTimeSubscriptionCheckout(transaction, subid) {
		return s.processOneTimeSubscriptionPayment(ctx, transaction)
	}
	return s.processLegacyRecurringSubscriptionPayment(ctx, subid, transaction)
}

func isOneTimeSubscriptionCheckout(transaction *model.PaymentTransaction, subid string) bool {
	if transaction != nil && transaction.Snapshot != nil {
		if v, ok := transaction.Snapshot["stripe_subscription_billing"]; ok {
			if s, ok := v.(string); ok && s == "one_time" {
				return true
			}
		}
	}
	return subid == "" || !strings.HasPrefix(subid, "sub_")
}

func snapshotBool(snapshot model.JSONMap, key string) bool {
	if snapshot == nil {
		return false
	}
	v, ok := snapshot[key]
	if !ok {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		return x == "true" || x == "1"
	default:
		return false
	}
}

// processLegacyRecurringSubscriptionPayment 处理老用户 Stripe 自动续订订阅的首单
func (s *stripeService) processLegacyRecurringSubscriptionPayment(ctx context.Context, subid string, transaction *model.PaymentTransaction) error {
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, transaction.SiteID, "stripe")
	if err != nil {
		return err
	}
	if config == nil || !config.IsActive || config.StripeSecretKey == "" {
		return common.ErrPaymentProviderNotConfigured
	}
	s.client.SetAPIKey(config.StripeSecretKey)
	start, end, err := s.client.GetSubscriptionStartEnd(subid)
	if err != nil {
		// 使用全局 logger 记录错误，避免对 ctx 进行类型断言导致 panic
		s.service.Logger().Error("Failed to get subscription period", zap.Error(err))
		return fmt.Errorf("failed to get subscription period: %w", err)
	}

	// Convert the Unix timestamp to time.Time
	startTime := time.Unix(start, 0)
	endTime := time.Unix(end, 0)

	// Get the user to check if they already have premium status
	user, err := s.userRepository.GetByUserID(ctx, transaction.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found: %s", transaction.UserID)
	}
	// Update the premium status
	user.PremiumType = 1 // Set to regular member
	user.PremiumExpiresAt = &endTime

	// Save the changes
	err = s.userRepository.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update user premium status: %w", err)
	}

	// Create a record in the user_subscriptions table to track the subscription details.
	// This record is important for managing the subscription lifecycle,
	// including renewals, cancellations, and syncing with the payment provider.
	userSubscription := &model.UserSubscription{
		SubscriptionID:         uuid.New().String(),
		UserID:                 transaction.UserID,
		SiteID:                 transaction.SiteID,
		PackageID:              transaction.RelatedID, // The ID of the subscription package in your system
		Provider:               transaction.Provider,
		ProviderSubscriptionID: subid,     // IMPORTANT: This is currently the Stripe Checkout Session ID from the transaction.
		Status:                 1,         // Active
		CurrentPeriodStart:     startTime, // The start of the current billing cycle
		CurrentPeriodEnd:       endTime,   // The end of the current billing cycle
	}

	if err := s.userSubscriptionRepo.Create(ctx, userSubscription); err != nil {
		// This error will propagate up and cause the database transaction in handleCheckoutSessionCompleted to roll back.
		return fmt.Errorf("failed to save user subscription details: %w", err)
	}

	// userCoins, err := s.userCoinsRepo.UpdateBalance(ctx, params.UserID, params.SiteID, params.Amount, params.RealMoneySpent)
	// if err != nil {
	//     return nil, err
	// }
	// There's a coupling here - we need to record the subscription user's spending amount in the user's coin account
	_, err = s.userCoinsRepository.UpdateBalance(ctx, transaction.UserID, transaction.SiteID, 0, transaction.Amount)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	// Grant coins from subscription package if configured
	pkg, err := s.subscriptionPackageRepo.GetByPackageID(ctx, transaction.RelatedID)
	if err != nil {
		s.service.Logger().Error("Failed to get subscription package for coins grant",
			zap.Error(err),
			zap.String("userID", transaction.UserID),
			zap.String("siteID", transaction.SiteID),
			zap.String("packageID", transaction.RelatedID),
		)
		// Don't fail the transaction for this error, just log it
	} else if pkg != nil && pkg.Coins > 0 {
		_, err = s.userCoinsRepository.AddPresentCoins(ctx, transaction.UserID, transaction.SiteID, pkg.Coins)
		if err != nil {
			s.service.Logger().Error("Failed to grant present coins",
				zap.Error(err),
				zap.String("userID", transaction.UserID),
				zap.String("siteID", transaction.SiteID),
				zap.String("packageID", pkg.PackageID),
				zap.Int("coins", int(pkg.Coins)),
			)
			// Don't fail the transaction for this error, just log it
		} else {
			s.service.Logger().Info("Granted coins for subscription",
				zap.String("userID", transaction.UserID),
				zap.String("siteID", transaction.SiteID),
				zap.String("packageID", pkg.PackageID),
				zap.Int("coins", int(pkg.Coins)),
			)
		}
	}

	return nil
}

// processOneTimeSubscriptionPayment 处理按周期一次性购买的订阅（与购金币相同 Checkout，到期需再次购买）
func (s *stripeService) processOneTimeSubscriptionPayment(ctx context.Context, transaction *model.PaymentTransaction) error {
	if snapshotBool(transaction.Snapshot, "one_time_membership_granted") {
		return nil
	}

	pkg, err := s.subscriptionPackageRepo.GetByPackageID(ctx, transaction.RelatedID)
	if err != nil {
		return fmt.Errorf("failed to get subscription package: %w", err)
	}
	if pkg == nil {
		return errors.New("subscription package not found")
	}

	user, err := s.userRepository.GetByUserID(ctx, transaction.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", transaction.UserID)
	}

	now := time.Now()
	startTime := now
	// 若当前会员未过期，新购周期从到期日起算（叠加）
	if user.PremiumExpiresAt != nil && user.PremiumExpiresAt.After(now) {
		startTime = *user.PremiumExpiresAt
	}
	endTime := subscriptionPeriodEnd(startTime, pkg.Interval)

	onetimeSub := int8(1)
	if err := s.userRepository.UpdatePremiumType(ctx, transaction.UserID, 1, &endTime, &onetimeSub); err != nil {
		return fmt.Errorf("failed to update user premium status: %w", err)
	}

	// 查找该用户在本站的一次性 Stripe 订阅记录（非 sub_ 自动续订）
	subs, err := s.userSubscriptionRepo.GetByUserID(ctx, transaction.UserID, transaction.SiteID)
	if err != nil {
		return fmt.Errorf("failed to list user subscriptions: %w", err)
	}

	var existing *model.UserSubscription
	for _, sub := range subs {
		if sub.Provider == "stripe" && !strings.HasPrefix(sub.ProviderSubscriptionID, "sub_") {
			existing = sub
			break
		}
	}

	providerPaymentID := transaction.ProviderPaymentID
	if providerPaymentID == "" {
		providerPaymentID = transaction.TransactionID
	}

	if existing != nil {
		existing.PackageID = transaction.RelatedID
		existing.ProviderSubscriptionID = providerPaymentID
		existing.Status = model.SubscriptionStatusActive
		existing.CurrentPeriodStart = startTime
		existing.CurrentPeriodEnd = endTime
		existing.CancelAtPeriodEnd = false
		if err := s.userSubscriptionRepo.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update user subscription: %w", err)
		}
	} else {
		userSubscription := &model.UserSubscription{
			SubscriptionID:         uuid.New().String(),
			UserID:                 transaction.UserID,
			SiteID:                 transaction.SiteID,
			PackageID:              transaction.RelatedID,
			Provider:               transaction.Provider,
			ProviderSubscriptionID: providerPaymentID,
			Status:                 model.SubscriptionStatusActive,
			CurrentPeriodStart:     startTime,
			CurrentPeriodEnd:       endTime,
		}
		if err := s.userSubscriptionRepo.Create(ctx, userSubscription); err != nil {
			return fmt.Errorf("failed to save user subscription details: %w", err)
		}
	}

	if _, err = s.userCoinsRepository.UpdateBalance(ctx, transaction.UserID, transaction.SiteID, 0, transaction.Amount); err != nil {
		// 消费统计失败不应阻断会员发放
		s.service.Logger().Warn("Failed to update user coin spending stats for subscription",
			zap.Error(err),
			zap.String("userID", transaction.UserID),
			zap.String("transactionID", transaction.TransactionID),
		)
	}

	if pkg.Coins > 0 {
		if _, err = s.userCoinsRepository.AddPresentCoins(ctx, transaction.UserID, transaction.SiteID, pkg.Coins); err != nil {
			s.service.Logger().Error("Failed to grant present coins for one-time subscription",
				zap.Error(err),
				zap.String("userID", transaction.UserID),
				zap.String("packageID", pkg.PackageID),
			)
		}
	}

	if transaction.Snapshot == nil {
		transaction.Snapshot = model.JSONMap{}
	}
	transaction.Snapshot["one_time_membership_granted"] = true
	if err := s.paymentTransactionRepo.Update(ctx, transaction); err != nil {
		return fmt.Errorf("failed to mark one-time membership granted: %w", err)
	}

	return nil
}

// subscriptionPeriodEnd 根据套餐周期计算结束时间
func subscriptionPeriodEnd(start time.Time, interval string) time.Time {
	switch interval {
	case "week":
		return start.AddDate(0, 0, 7)
	case "month":
		return start.AddDate(0, 1, 0)
	case "year":
		return start.AddDate(1, 0, 0)
	default:
		return start.AddDate(0, 1, 0)
	}
}

// CancelSubscription cancels a subscription
func (s *stripeService) CancelSubscription(ctx context.Context, userID string, subscriptionID string, cancelAtPeriodEnd bool) error {
	// Validate request
	if subscriptionID == "" || userID == "" {
		return errors.New("subscriptionId and userId are required")
	}

	// Fetch the user subscription record
	userSubscription, err := s.userSubscriptionRepo.GetBySubscriptionID(ctx, subscriptionID)
	if err != nil {
		s.service.Logger().Error("Failed to get user subscription", zap.Error(err))
		return err
	}

	// Verify this subscription belongs to the requesting user
	if userSubscription == nil {
		return common.ErrNotFound
	}

	if userSubscription.UserID != userID {
		return common.ErrUnauthorized
	}

	// 一次性订阅无 Stripe 自动续订合约，仅更新本地状态
	if !strings.HasPrefix(userSubscription.ProviderSubscriptionID, "sub_") {
		userSubscription.CancelAtPeriodEnd = true
		if !cancelAtPeriodEnd {
			userSubscription.Status = model.SubscriptionStatusCancelled
		}
		return s.userSubscriptionRepo.Update(ctx, userSubscription)
	}

	// Get the payment config for the subscription's site
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, userSubscription.SiteID, userSubscription.Provider)
	if err != nil {
		return err
	}

	if config == nil || !config.IsActive || config.StripeSecretKey == "" {
		return common.ErrPaymentProviderNotConfigured
	}

	// Set the API key for the client
	s.client.SetAPIKey(config.StripeSecretKey)

	// Call Stripe API to cancel subscription
	err = s.client.CancelSubscription(userSubscription.ProviderSubscriptionID)
	if err != nil {
		return err
	}

	// 回调中 和 取消入口都处理
	userSubscription.CancelAtPeriodEnd = true
	err = s.userSubscriptionRepo.UpdatePeriodByProviderSubID(ctx, userSubscription.ProviderSubscriptionID, userSubscription)
	if err != nil {
		return err
	}
	// err = s.userRepository.UpdatePremiumType(ctx, userID, 0, nil )
	// if err != nil {
	// 	return err
	// }
	return nil

}

// extractInvoiceSubscriptionIDFromWebhook 从 invoice.paid 等事件的原始 JSON 解析订阅 ID。
// stripe-go v82 的 Invoice 已不含顶级 subscription 字段；旧版 Webhook 或部分 payload 仅含
// "subscription":"sub_xxx"，或 parent 未展开时，需从 Raw 回退解析，否则续订逻辑（含数数）整段被跳过。
func extractInvoiceSubscriptionIDFromWebhook(raw []byte) string {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return ""
	}
	trySubJSON := func(sub json.RawMessage) string {
		if len(sub) == 0 {
			return ""
		}
		var idStr string
		if json.Unmarshal(sub, &idStr) == nil && strings.HasPrefix(idStr, "sub_") {
			return idStr
		}
		var obj struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(sub, &obj) == nil && strings.HasPrefix(obj.ID, "sub_") {
			return obj.ID
		}
		return ""
	}
	if parent, ok := root["parent"]; ok {
		var p map[string]json.RawMessage
		if json.Unmarshal(parent, &p) == nil {
			if sd, ok := p["subscription_details"]; ok {
				var sdet map[string]json.RawMessage
				if json.Unmarshal(sd, &sdet) == nil {
					if sub, ok := sdet["subscription"]; ok {
						if id := trySubJSON(sub); id != "" {
							return id
						}
					}
				}
			}
		}
	}
	if sub, ok := root["subscription"]; ok {
		return trySubJSON(sub)
	}
	return ""
}

// handleInvoicePaid handles subscription renewal payments
func (s *stripeService) handleInvoicePaid(ctx *gin.Context, siteID string, event *goStripe.Event) error {
	var invoice goStripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return err
	}

	// 只处理订阅期间的数据。首次订阅创建不处理
	if invoice.BillingReason != goStripe.InvoiceBillingReasonSubscriptionCycle {
		log.Warning(ctx, fmt.Sprintf("Invoice billing reason is not subscription_cycle: %s, billingReason: %s", invoice.ID, invoice.BillingReason))
		return nil
	}
	providerSubscriptionID := ""
	if invoice.Parent != nil && invoice.Parent.SubscriptionDetails != nil && invoice.Parent.SubscriptionDetails.Subscription != nil {
		providerSubscriptionID = invoice.Parent.SubscriptionDetails.Subscription.ID
	}
	if providerSubscriptionID == "" {
		providerSubscriptionID = extractInvoiceSubscriptionIDFromWebhook(event.Data.Raw)
	}
	if providerSubscriptionID == "" {
		log.Warning(ctx, fmt.Sprintf("Invoice is not related to a subscription: %s", invoice.ID))
		return nil
	}

	// 从用户订阅记录中获取包 ID
	userSubscription, err := s.userSubscriptionRepo.GetByProviderSubscriptionID(ctx, "stripe", providerSubscriptionID)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to get user subscription for providerSubscriptionID: %s, error: %s", providerSubscriptionID, err.Error()))
		return err
	}
	if userSubscription == nil {
		log.Error(ctx, fmt.Sprintf("User subscription not found for providerSubscriptionID: %s", providerSubscriptionID))
		return errors.New("user subscription not found")
	}

	// 获取包信息
	pkg, err := s.subscriptionPackageRepo.GetByPackageID(ctx, userSubscription.PackageID)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to get subscription package for packageID: %s, error: %s", userSubscription.PackageID, err.Error()))
		return err
	}
	if pkg == nil {
		log.Error(ctx, fmt.Sprintf("Subscription package not found for packageID: %s", userSubscription.PackageID))
		return errors.New("subscription package not found")
	}

	if pkg.SiteID != siteID {
		log.Warning(ctx, fmt.Sprintf("Package not found for this site: packageID=%s, siteID=%s", pkg.PackageID, siteID))
		return nil
	}

	// 同一 invoice 可能收到 invoice.paid 与 invoice.payment_succeeded，避免重复记账/发币/打点
	existing, err := s.paymentTransactionRepo.GetByProviderPaymentID(ctx, "stripe", invoice.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		log.Info(ctx, fmt.Sprintf("Renewal invoice already processed, skip duplicate webhook: invoice=%s", invoice.ID))
		return nil
	}

	// 创建 snapshot 和交易记录
	snapshot := model.JSONMap{
		"package_id":                  pkg.PackageID,
		"name":                        pkg.Name,
		"description":                 pkg.Description,
		"interval":                    pkg.Interval,
		"price":                       pkg.Price,
		"currency":                    pkg.Currency,
		"original_price":              pkg.OriginalPrice,
		"discount_percentage":         pkg.DiscountPercentage,
		"coins":                       pkg.Coins,
		"coin_amount":                 float64(pkg.Coins),
		"rights":                      pkg.Rights,
		"stripe_subscription_billing": "renewal",
		"is_subscription_renewal":     true,
	}

	// Create transaction record
	transactionID := uuid.New().String()
	tx := &model.PaymentTransaction{
		TransactionID:     transactionID,
		OrderID:           transactionID,
		UserID:            userSubscription.UserID,
		SiteID:            userSubscription.SiteID,
		Amount:            pkg.Price,
		Currency:          pkg.Currency,
		Provider:          "stripe",
		ProviderPaymentID: invoice.ID,
		PaymentType:       model.PaymentTypeSubscription,
		Status:            model.PaymentStatusSuccess,
		RelatedID:         pkg.PackageID,
		RelatedType:       model.RelatedTypeSubscription,
		PayerEmail:        payutil.StripeInvoicePayerEmail(&invoice),
		Snapshot:          snapshot,
	}

	err = s.paymentTransactionRepo.Create(ctx, tx)
	if err != nil {
		// 并发：invoice.paid 与 invoice.payment_succeeded 同时到达，或 Stripe 重试；唯一键保证仅一条流水
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			log.Info(ctx, fmt.Sprintf("Renewal transaction duplicate (unique invoice id), skip: invoice=%s", invoice.ID))
			return nil
		}
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			log.Info(ctx, fmt.Sprintf("Renewal transaction duplicate key 1062, skip: invoice=%s", invoice.ID))
			return nil
		}
		return err
	}

	// 更新用户总消费
	_, err = s.userCoinsRepository.UpdateBalance(ctx, userSubscription.UserID, userSubscription.SiteID, 0, pkg.Price)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	// Grant coins from subscription package if configured for renewal
	if pkg.Coins > 0 {
		_, err = s.userCoinsRepository.AddPresentCoins(ctx, userSubscription.UserID, userSubscription.SiteID, pkg.Coins)
		if err != nil {
			log.Error(ctx, fmt.Sprintf("Failed to grant present coins for renewal: %s", err.Error()))
			// Don't fail the transaction for this error, just log it
		} else {
			log.Info(ctx, fmt.Sprintf("Granted %d coins to user %s for subscription renewal %s", pkg.Coins, userSubscription.UserID, pkg.PackageID))
		}
	}

	// 续订与首次订阅一致上报数数（同步 + 结构化日志，避免 goroutine 内失败不可见）
	if s.trackingService != nil {
		if trackErr := s.trackingService.TrackPurchase(ctx, tx); trackErr != nil {
			log.Error(ctx, fmt.Sprintf("续订数数 purchase 打点失败: invoice=%s txn=%s err=%v", invoice.ID, tx.TransactionID, trackErr))
		} else {
			log.Info(ctx, fmt.Sprintf("续订数数 purchase 打点成功: invoice=%s txn=%s", invoice.ID, tx.TransactionID))
		}
	}

	return nil
}

// handleSubscriptionUpdated handles subscription status changes
func (s *stripeService) handleSubscriptionUpdated(ctx *gin.Context, siteID string, event *goStripe.Event) error {
	var subscription goStripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return err
	}

	subscriptionID := subscription.ID

	// 更新订阅状态
	var newStatus int
	switch subscription.Status {
	case "active":
		newStatus = model.SubscriptionStatusActive
	case "past_due":
		newStatus = model.SubscriptionStatusActive // 可以考虑添加一个新状态来表示付款逾期
	case "canceled":
		newStatus = model.SubscriptionStatusCancelled
	case "unpaid":
		newStatus = model.SubscriptionStatusExpired
	default:
		newStatus = 0 // 保持不变
	}
	// 获取用户订阅记录
	userSubscription, err := s.userSubscriptionRepo.GetByProviderSubscriptionID(ctx, "stripe", subscriptionID)
	if err != nil {
		return err
	}
	if userSubscription == nil {
		return fmt.Errorf("subscription not found: %s", subscriptionID)
		// if newStatus != model.SubscriptionStatusActive {
		// 	s.service.Logger().Info("Subscription not found, but status is not active, ignoring", zap.String("subscriptionID", subscriptionID))
		// 	return nil
		// }
		//  s.service.Logger().Info("Subscription not found,create new", zap.String("subscriptionID", subscriptionID))
		//  userSubscription = &model.UserSubscription{
		// 	 ProviderSubscriptionID: subscriptionID,
		// 	 Provider:               "stripe",
		// 	 SiteID:                siteID,
		// 	 Status:                model.SubscriptionStatusActive,
		// 	 CurrentPeriodStart:    time.Unix(subscription.StartDate, 0),
		// 	 CurrentPeriodEnd:      time.Unix(subscription.EndedAt, 0),
		// 	 CancelAtPeriodEnd:     true,
		//  }
		//  return s.userSubscriptionRepo.Create(ctx, userSubscription)
	}

	// 确认该订阅是针对当前站点的
	if userSubscription.SiteID != siteID {
		log.Warning(ctx, fmt.Sprintf("Subscription not found for this site: subscriptionID=%s, siteID=%s", subscriptionID, siteID))
		return nil
	}

	// 更新账单周期和取消状态
	userSubscription.CurrentPeriodStart = time.Unix(subscription.Items.Data[0].CurrentPeriodStart, 0)
	userSubscription.CurrentPeriodEnd = time.Unix(subscription.Items.Data[0].CurrentPeriodEnd, 0)
	userSubscription.CancelAtPeriodEnd = subscription.CancelAtPeriodEnd
	userSubscription.Status = newStatus

	err = s.userSubscriptionRepo.UpdatePeriodByProviderSubID(ctx, subscriptionID, userSubscription)
	if err != nil {
		return err
	}

	// 如果订阅已取消或过期，更新用户状态
	user, err := s.userRepository.GetByUserID(ctx, userSubscription.UserID)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to get user for subscriptionID: %s, error: %s", subscriptionID, err.Error()))
		return err
	}
	if user == nil {
		log.Error(ctx, fmt.Sprintf("User not found for subscriptionID: %s", subscriptionID))
		return errors.New("user not found")
	}
	// 如果设置为"在周期结束时取消"，让会员资格保持到期末
	user.PremiumExpiresAt = &userSubscription.CurrentPeriodEnd
	if newStatus != model.SubscriptionStatusActive && !subscription.CancelAtPeriodEnd {
		user.PremiumType = 0
		// user.PremiumExpiresAt = time.Time{}
	}

	err = s.userRepository.Update(ctx, user)
	if err != nil {
		return err
	}

	log.AddNotice(ctx, "subscription_id", subscriptionID)
	log.AddNotice(ctx, "user_id", userSubscription.UserID)
	log.AddNotice(ctx, "new_status", string(subscription.Status))
	log.AddNotice(ctx, "new_status_code", fmt.Sprintf("%d", newStatus))
	log.AddNotice(ctx, "cancel_at_period_end", fmt.Sprintf("%t", subscription.CancelAtPeriodEnd))

	return nil
}

// handleSubscriptionDeleted handles subscription cancellation events
func (s *stripeService) handleSubscriptionDeleted(ctx *gin.Context, siteID string, event *goStripe.Event) error {
	var subscription goStripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return err
	}

	subscriptionID := subscription.ID

	// 获取用户订阅记录
	userSubscription, err := s.userSubscriptionRepo.GetByProviderSubscriptionID(ctx, "stripe", subscriptionID)
	if err != nil {
		return err
	}
	if userSubscription == nil {
		log.Error(ctx, fmt.Sprintf("Subscription not found for ID: %s", subscriptionID))
		return errors.New("subscription not found")
	}

	// 确认该订阅是针对当前站点的
	if userSubscription.SiteID != siteID {
		log.Warning(ctx, fmt.Sprintf("Subscription not found for this site: subscriptionID=%s, siteID=%s", subscriptionID, siteID))
		return nil
	}
	userSubscription.CancelAtPeriodEnd = subscription.CancelAtPeriodEnd

	err = s.userSubscriptionRepo.UpdateStatusByProviderSubID(ctx, subscriptionID, model.SubscriptionStatusCancelled)
	if err != nil {
		return err
	}

	// 取消会员的状态。TODO，事物
	err = s.userRepository.UpdatePremiumType(ctx, userSubscription.UserID, 0, nil, nil)
	if err != nil {
		log.Warning(ctx, fmt.Sprintf("Failed to update user premium status: %s", err.Error()))
	}

	log.AddNotice(ctx, "subscription_id", subscriptionID)
	log.AddNotice(ctx, "user_id", userSubscription.UserID)

	return nil
}

// CreateCoinPackageOrder creates a new coin package order
// handleSubscriptionCreated handles new subscription creation events
func (s *stripeService) handleSubscriptionCreated(ctx *gin.Context, siteID string, event *goStripe.Event) error {
	var subscription goStripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return err
	}

	subscriptionID := subscription.ID

	// Check if subscription record already exists
	userSubscription, err := s.userSubscriptionRepo.GetByProviderSubscriptionID(ctx, "stripe", subscriptionID)
	if err != nil {
		return err
	}

	// If subscription record already exists, just log and return
	if userSubscription != nil {
		log.Warning(ctx, fmt.Sprintf("Subscription record already exists: %s", subscriptionID))
		return nil
	}

	// Look for user_id and package_id in the metadata
	userID, hasUserID := subscription.Metadata["user_id"]
	packageID, hasPackageID := subscription.Metadata["package_id"]

	if !hasUserID || !hasPackageID {
		log.Warning(ctx, fmt.Sprintf("Subscription created without required metadata: %s", subscriptionID))
		return nil
	}

	// Get the subscription package
	pkg, err := s.subscriptionPackageRepo.GetByPackageID(ctx, packageID)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to get subscription package for packageID: %s, error: %s", packageID, err.Error()))
		return err
	}
	if pkg == nil {
		log.Error(ctx, fmt.Sprintf("Subscription package not found for packageID: %s", packageID))
		return errors.New("subscription package not found")
	}

	// Create a new user subscription record
	userSubscription = &model.UserSubscription{
		SubscriptionID:         uuid.New().String(),
		UserID:                 userID,
		SiteID:                 siteID,
		PackageID:              packageID,
		Provider:               "stripe",
		ProviderSubscriptionID: subscriptionID,
		Status:                 model.SubscriptionStatusActive,
		CurrentPeriodStart:     time.Unix(subscription.Items.Data[0].CurrentPeriodStart, 0),
		CurrentPeriodEnd:       time.Unix(subscription.Items.Data[0].CurrentPeriodEnd, 0),
		CancelAtPeriodEnd:      subscription.CancelAtPeriodEnd,
	}

	if err := s.userSubscriptionRepo.Create(ctx, userSubscription); err != nil {
		s.service.Logger().Error("Failed to create user subscription record",
			zap.Error(err),
			zap.String("userID", userID),
			zap.String("subscriptionID", subscriptionID))
		return err
	}

	// Update user premium status
	user, err := s.userRepository.GetByUserID(ctx, userID)
	if err != nil || user == nil {
		s.service.Logger().Error("Failed to get user", zap.Error(err))
		return err
	}

	// Set user as premium with expiration date
	user.PremiumType = 1
	user.PremiumExpiresAt = &userSubscription.CurrentPeriodEnd

	if err := s.userRepository.Update(ctx, user); err != nil {
		s.service.Logger().Error("Failed to update user premium status", zap.Error(err))
		return err
	}
	log.AddNotice(ctx, "subscription_id", subscriptionID)
	log.AddNotice(ctx, "user_id", userID)
	log.AddNotice(ctx, "expires_at", userSubscription.CurrentPeriodEnd)

	return nil
}

// GetUserPurchases retrieves user's purchase records (coin purchases and subscriptions)
func (s *stripeService) GetUserPurchases(ctx context.Context, userID string, siteID string, page, pageSize int) ([]*api.PurchaseRecord, int64, error) {
	transactions, total, err := s.paymentTransactionRepo.ListUserPurchases(ctx, userID, siteID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Convert to API response
	records := make([]*api.PurchaseRecord, 0, len(transactions))
	for _, tx := range transactions {
		record := &api.PurchaseRecord{
			TransactionID: tx.TransactionID,
			OrderID:       tx.OrderID,
			Amount:        types.Money(tx.Amount),
			Currency:      tx.Currency,
			Provider:      tx.Provider,
			PaymentType:   tx.PaymentType,
			Status:        tx.Status,
			RelatedID:     tx.RelatedID,
			RelatedType:   tx.RelatedType,
			CreatedAt:     tx.CreatedAt.Unix(),
		}

		// Set payment type name
		if tx.PaymentType == model.PaymentTypeSubscription {
			record.PaymentTypeName = "Subscription"
		} else if tx.PaymentType == model.PaymentTypeCoinPackage {
			record.PaymentTypeName = "Coin Package"
		}

		// Set status name
		switch tx.Status {
		case model.PaymentStatusPending:
			record.StatusName = "Pending"
		case model.PaymentStatusSuccess:
			record.StatusName = "Success"
		case model.PaymentStatusFailed:
			record.StatusName = "Failed"
		case model.PaymentStatusRefunded:
			record.StatusName = "Refunded"
		case model.PaymentStatusPartialRefunded:
			record.StatusName = "Partial Refunded"
		case model.PaymentStatusExpired:
			record.StatusName = "Expired"
		default:
			record.StatusName = "Unknown"
		}

		// Extract package details from snapshot
		if tx.Snapshot != nil {
			if name, ok := tx.Snapshot["name"].(string); ok {
				record.PackageName = name
			}
			if desc, ok := tx.Snapshot["description"].(string); ok {
				record.PackageDescription = desc
			}
			if coinAmount, ok := tx.Snapshot["coinAmount"].(float64); ok {
				record.CoinAmount = int(coinAmount)
			}
			if interval, ok := tx.Snapshot["interval"].(string); ok {
				record.Interval = interval
			}
		}

		records = append(records, record)
	}

	return records, total, nil
}
