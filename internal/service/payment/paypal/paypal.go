package paypal

import (
	"context"
	"errors"
	"fmt"
	"time"

	paypalclient "shortpress-server/internal/adapter/payment/paypal"
	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/service"
	"shortpress-server/internal/service/analytics"
	"shortpress-server/internal/service/payment/coins"
	payutil "shortpress-server/internal/service/payment"
	"shortpress-server/internal/types"

	peymentRep "shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/user"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"

	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const (
	PayPalSandboxURL = "https://api-m.sandbox.paypal.com"
	PayPalLiveURL    = "https://api-m.paypal.com"
)

type payPalTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type payPalOrderRequest struct {
	Intent         string               `json:"intent"`
	PurchaseUnits  []payPalPurchaseUnit `json:"purchase_units"`
	ApplicationCtx payPalApplicationCtx `json:"application_context"`
}

type payPalPurchaseUnit struct {
	ReferenceID string       `json:"reference_id,omitempty"`
	CustomID    string       `json:"custom_id,omitempty"`
	Amount      payPalAmount `json:"amount"`
	Description string       `json:"description,omitempty"`
}

type payPalAmount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

type payPalApplicationCtx struct {
	ReturnURL string `json:"return_url"`
	CancelURL string `json:"cancel_url"`
}

type payPalOrderResponse struct {
	ID     string       `json:"id"`
	Status string       `json:"status"`
	Links  []payPalLink `json:"links"`
}

type payPalLink struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method"`
}

type payPalCaptureResponse struct {
	ID     string                 `json:"id"`
	Status string                 `json:"status"`
	Amount map[string]interface{} `json:"amount"`
}

type paypalService struct {
	service                 *service.Service
	conf                    *viper.Viper
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
	paypalClient            paypalclient.PayPalClient
}

// NewPaypalService creates a new PayPal payment service
func NewPaypalService(
	paymentConfigRepo peymentRep.PaymentConfigRepository,
	conf *viper.Viper,
) *paypalService {
	return &paypalService{
		paymentConfigRepo: paymentConfigRepo,
		conf:              conf,
	}
}

// NewPaypalServiceFull creates a new PayPal payment service with full dependencies
func NewPaypalServiceFull(
	service *service.Service,
	conf *viper.Viper,
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
) *paypalService {
	return &paypalService{
		service:                 service,
		conf:                    conf,
		paymentConfigRepo:       paymentConfigRepo,
		coinPackageRepo:         coinPackageRepo,
		paymentTransactionRepo:  paymentTransactionRepo,
		userCoinsRepository:     userCoinsRepository,
		webhookEventRepository:  webhookEventRepository,
		subscriptionPackageRepo: subscriptionPackageRepo,
		userRepository:          userRepository,
		userSubscriptionRepo:    userSubscriptionRepo,
		coinsService:            coinsService,
		trackingService:         trackingService,
		paypalClient:            paypalclient.NewClient(),
	}
}

// SaveConfig saves PayPal configuration
func (s *paypalService) SaveConfig(ctx *gin.Context, config api.PaymentProviderConfig) error {
	if config.PaypalConf.ClientID == "" || config.PaypalConf.ClientSecret == "" {
		return common.ErrPaymentProviderNotConfigured
	}

	// Get existing config
	existingConfig, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, config.SiteID, config.Provider)
	if err != nil {
		log.Error(ctx, "GetBySiteIDAndProvider error: "+err.Error())
		return err
	}

	now := time.Now()
	var savedConfig *model.PaymentConfig

	if existingConfig != nil {
		// Update existing config
		existingConfig.PaypalClientID = config.PaypalConf.ClientID
		existingConfig.PaypalClientSecret = config.PaypalConf.ClientSecret
		existingConfig.IsSandbox = config.PaypalConf.IsSandbox
		existingConfig.IsActive = true
		existingConfig.LastVerifiedAt = &now
		existingConfig.VerificationStatus = 1
		existingConfig.ErrorMessage = ""

		err = s.paymentConfigRepo.Update(ctx, existingConfig)
		if err != nil {
			log.Error(ctx, "Update payment config error: "+err.Error())
			return err
		}
		savedConfig = existingConfig
	} else {
		// Create new config (config_id must be unique; Stripe path sets UUID — PayPal was missing it and caused uk_config_id duplicate '')
		newConfig := &model.PaymentConfig{
			ConfigID:           uuid.New().String(),
			SiteID:             config.SiteID,
			Provider:           config.Provider,
			PaypalClientID:     config.PaypalConf.ClientID,
			PaypalClientSecret: config.PaypalConf.ClientSecret,
			IsSandbox:          config.PaypalConf.IsSandbox,
			IsActive:           true,
			LastVerifiedAt:     &now,
			VerificationStatus: 1,
			ErrorMessage:       "",
		}

		err = s.paymentConfigRepo.Create(ctx, newConfig)
		if err != nil {
			log.Error(ctx, "Create payment config error: "+err.Error())
			return err
		}
		savedConfig = newConfig
	}

	// Auto-configure webhook (like Stripe)
	err = s.autoConfigureWebhook(ctx, savedConfig)
	if err != nil {
		log.Warning(ctx, "Auto-configure webhook failed: "+err.Error())
		// Don't fail the config save if webhook setup fails
		// Users can manually configure webhook later
	} else {
		log.Info(ctx, "PayPal webhook auto-configured successfully")
	}

	log.Info(ctx, "PayPal configuration saved successfully for site: "+config.SiteID)
	return nil
}

// autoConfigureWebhook automatically configures PayPal webhook
func (s *paypalService) autoConfigureWebhook(ctx *gin.Context, config *model.PaymentConfig) error {
	// Get webhook URL from config
	webhookURL := s.conf.GetString("webhook.paypal.url")
	if webhookURL == "" {
		log.Info(ctx, "PayPal webhook URL not configured, skipping auto-configuration")
		return nil
	}

	// Add siteId parameter to webhook URL
	webhookURL = webhookURL + "?siteId=" + config.SiteID

	// Set PayPal credentials
	s.paypalClient.SetCredentials(config.PaypalClientID, config.PaypalClientSecret, config.IsSandbox)

	// Define events to listen for
	events := []string{
		"PAYMENT.CAPTURE.COMPLETED",
		"PAYMENT.CAPTURE.DENIED",
		"BILLING.SUBSCRIPTION.ACTIVATED",
		"BILLING.SUBSCRIPTION.CANCELLED",
		"BILLING.SUBSCRIPTION.PAYMENT.FAILED",
	}

	// Check if webhook already exists
	exists, err := s.paypalClient.VerifyWebhook(webhookURL)
	if err != nil {
		log.Warning(ctx, "Failed to verify existing webhook: "+err.Error())
		// Continue to create webhook
	}

	if exists {
		log.Info(ctx, "PayPal webhook already exists: "+webhookURL)
		// TODO: Update webhook events if needed
		return nil
	}

	// Create new webhook
	webhookID, err := s.paypalClient.CreateWebhook(webhookURL, events)
	if err != nil {
		log.Error(ctx, "Failed to create PayPal webhook: "+err.Error())
		return err
	}

	// Update config with webhook ID
	config.ProviderWebhookID = webhookID
	config.EndPointUrl = webhookURL

	err = s.paymentConfigRepo.Update(ctx, config)
	if err != nil {
		log.Error(ctx, "Failed to update config with webhook ID: "+err.Error())
		return err
	}

	log.Info(ctx, "PayPal webhook created successfully: "+webhookID)
	return nil
}

// GetAccountInfo retrieves PayPal account information
func (s *paypalService) GetAccountInfo(ctx context.Context, siteID string) (*api.PaymentAccountInfo, error) {
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, siteID, "paypal")
	if err != nil || config == nil || !config.IsActive || config.PaypalClientID == "" {
		return nil, common.ErrPaymentProviderNotConfigured
	}

	// Return account info from database AccountInfo field
	accountInfo := &api.PaymentAccountInfo{
		AccountID: config.PaypalClientID,
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

// ConfTest tests PayPal configuration by verifying credentials
func (s *paypalService) ConfTest(ctx context.Context, clientSecret string) error {
	// TODO: Implement actual PayPal API test
	// For now, just check if client secret is not empty
	if clientSecret == "" {
		return errors.New("PayPal client secret is empty")
	}
	return nil
}

// CreateCoinPackage creates a new coin package for PayPal payment
func (s *paypalService) CreateCoinPackage(ctx context.Context, req api.CoinPackageCreateRequest) (*api.CoinPackageCreateResponse, error) {
	// Generate package ID (same format as Stripe for consistency)
	packageID := uuid.NewString()

	coinPackage := &model.CoinPackage{
		PackageID:   packageID,
		SiteID:      req.SiteID,
		Name:        req.Name,
		Description: req.Description,
		Features:    req.Features,
		CoinAmount:  req.CoinAmount,
		Price:       int64(req.Price),
		Currency:    "USD",
		IOSProductID: req.IOSProductID,
		Status:      1,
	}

	err := s.coinPackageRepo.Create(ctx, coinPackage)
	if err != nil {
		return nil, err
	}

	return &api.CoinPackageCreateResponse{
		PackageID: packageID,
	}, nil
}

func (s *paypalService) getPayPalURL(ctx context.Context, siteID string) string {
	// Get PayPal configuration to check if sandbox mode
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, siteID, "paypal")
	if err != nil || config == nil {
		// If config not found, default to sandbox for safety
		return PayPalSandboxURL
	}

	// Use IsSandbox field from config: 1 = sandbox, 0 = live
	if config.IsSandbox {
		return PayPalSandboxURL
	}
	return PayPalLiveURL
}

func (s *paypalService) getPayPalAccessToken(ctx context.Context, siteID, clientID, clientSecret string) (string, error) {
	apiURL := s.getPayPalURL(ctx, siteID) + "/v1/oauth2/token"

	data := "grant_type=client_credentials"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data))
	if err != nil {
		return "", err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("paypal auth failed: %s", string(body))
	}

	var tokenResp payPalTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

// CreateOrder creates a new PayPal payment order
func (s *paypalService) CreateOrder(ctx *gin.Context, userID string, req api.OrderCreateRequest) (*api.OrderCreateResponse, error) {
	// Get PayPal configuration
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, req.SiteID, "paypal")
	if err != nil || config == nil || !config.IsActive || config.PaypalClientID == "" {
		return nil, common.ErrPaymentProviderNotConfigured
	}

	// Generate internal order ID
	orderID := uuid.New().String()

	// Get package details based on order type
	var price int64
	var currency string
	var paymentType int
	var relatedType int
	var description string
	var coinAmount int = 0 // For coin packages

	if req.OrderType == api.OrderTypeCoinPackage {
		pkg, err := s.coinPackageRepo.GetByPackageID(ctx, req.PackageID)
		if err != nil {
			return nil, err
		}
		if pkg == nil {
			return nil, errors.New("coin package not found")
		}
		price = pkg.Price
		currency = pkg.Currency
		paymentType = model.PaymentTypeCoinPackage
		relatedType = model.RelatedTypeCoinPackage
		description = pkg.Name
		coinAmount = pkg.CoinAmount // Store coin amount for later use
	} else if req.OrderType == api.OrderTypeSubscription {
		pkg, err := s.subscriptionPackageRepo.GetByPackageID(ctx, req.PackageID)
		if err != nil {
			return nil, err
		}
		if pkg == nil {
			return nil, errors.New("subscription package not found")
		}
		price = pkg.Price
		currency = pkg.Currency
		paymentType = model.PaymentTypeSubscription
		relatedType = model.RelatedTypeSubscription
		description = pkg.Name
	}

	// Get Access Token
	accessToken, err := s.getPayPalAccessToken(ctx, req.SiteID, config.PaypalClientID, config.PaypalClientSecret)
	if err != nil {
		log.Error(ctx, "Failed to get PayPal access token: "+err.Error())
		return nil, err
	}

	// Build success and cancel URLs, always attaching siteId & orderId like coin payments
	var successURL, cancelURL string
	if req.ReturnURL == "" {
		// Use default success URL from config
		successURL = s.paymentSuccessURL(s.conf.GetString("webhook.paypal.success_url"), req.SiteID, orderID)
	} else {
		// Use custom frontend URL as base, but still append order info
		successURL = s.paymentSuccessURL(req.ReturnURL, req.SiteID, orderID)
	}

	if req.CancelURL == "" {
		// Use default cancel URL from config
		cancelURL = s.paymentCancelURL(s.conf.GetString("webhook.paypal.cancel_url"), req.SiteID, orderID)
	} else {
		// Use custom frontend URL as base, but still append order info
		cancelURL = s.paymentCancelURL(req.CancelURL, req.SiteID, orderID)
	}

	// Create PayPal Order via REST API
	paypalOrderReq := payPalOrderRequest{
		Intent: "CAPTURE",
		PurchaseUnits: []payPalPurchaseUnit{
			{
				ReferenceID: orderID,
				CustomID:    orderID, // Set custom_id to match our internal order ID for webhook
				Amount: payPalAmount{
					CurrencyCode: currency,
					Value:        fmt.Sprintf("%.2f", float64(price)/100.0), // Assuming price is in cents
				},
				Description: description,
			},
		},
		ApplicationCtx: payPalApplicationCtx{
			ReturnURL: successURL,
			CancelURL: cancelURL,
		},
	}

	body, _ := json.Marshal(paypalOrderReq)
	apiURL := s.getPayPalURL(ctx, req.SiteID) + "/v2/checkout/orders"
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(body))
	httpReq.Header.Add("Authorization", "Bearer "+accessToken)
	httpReq.Header.Add("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 10 * time.Second}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusCreated && httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("failed to create paypal order: %s", string(respBody))
	}

	var paypalResp payPalOrderResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&paypalResp); err != nil {
		return nil, err
	}

	// Find the approve link
	var checkoutURL string
	for _, link := range paypalResp.Links {
		if link.Rel == "approve" {
			checkoutURL = link.Href
			break
		}
	}

	if checkoutURL == "" {
		return nil, errors.New("failed to get paypal approve link")
	}

	// Create internal payment transaction record
	snapshot := model.JSONMap{
		"paypal_order_id": paypalResp.ID,
		"package_id":      req.PackageID,
		"order_type":      string(req.OrderType),
		"name":            description,
		"price":           price,
		"currency":        currency,
	}
	if len(req.TrackingContext) > 0 {
		snapshot["tracking_context"] = req.TrackingContext
	}

	// Add coin_amount to snapshot for coin packages
	if req.OrderType == api.OrderTypeCoinPackage && coinAmount > 0 {
		snapshot["coin_amount"] = float64(coinAmount)
	}
	snapshot = analytics.MergeMetaIntoSnapshot(snapshot, req.Meta)

	transaction := &model.PaymentTransaction{
		TransactionID:     orderID,
		OrderID:           orderID,
		UserID:            userID,
		SiteID:            req.SiteID,
		RelatedID:         req.PackageID,
		Amount:            price,
		Currency:          currency,
		Provider:          "paypal",
		ProviderPaymentID: paypalResp.ID, // Use PayPal's order ID
		PaymentType:       paymentType,
		RelatedType:       relatedType,
		Status:            model.PaymentStatusPending,
		Snapshot:          snapshot,
	}

	err = s.paymentTransactionRepo.Create(ctx, transaction)
	if err != nil {
		return nil, err
	}

	return &api.OrderCreateResponse{
		OrderID:       orderID,
		CheckoutURL:   checkoutURL,
		PaymentStatus: "pending",
		SuccessURL:    successURL,
		CancelURL:     cancelURL,
	}, nil
}

// HandleWebhook handles PayPal webhook events
func (s *paypalService) HandleWebhook(ctx *gin.Context) error {
	// Read webhook body
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return err
	}

	// TODO: Verify PayPal webhook signature for security
	// For now, skip signature verification in testing mode

	var webhookEvent map[string]interface{}
	if err := json.Unmarshal(body, &webhookEvent); err != nil {
		return err
	}

	// Get event type
	eventType, _ := webhookEvent["event_type"].(string)
	log.Info(ctx, "PayPal webhook event: "+eventType)

	// Handle different event types
	switch eventType {
	case "CHECKOUT.ORDER.APPROVED":
		// Order approved by user, need to capture payment
		return s.handleOrderApproved(ctx, webhookEvent)
	case "PAYMENT.CAPTURE.COMPLETED":
		// Payment successfully completed
		return s.handlePaymentCompleted(ctx, webhookEvent)
	case "PAYMENT.CAPTURE.DENIED":
		// Payment denied
		return s.handlePaymentDenied(ctx, webhookEvent)
	case "BILLING.SUBSCRIPTION.CREATED":
		// Subscription created (user has approved but not yet activated)
		return s.handleSubscriptionCreated(ctx, webhookEvent)
	case "BILLING.SUBSCRIPTION.ACTIVATED":
		// Subscription activated
		return s.handleSubscriptionActivated(ctx, webhookEvent)
	case "BILLING.SUBSCRIPTION.CANCELLED":
		// Subscription cancelled
		return s.handleSubscriptionCancelled(ctx, webhookEvent)
	case "BILLING.SUBSCRIPTION.PAYMENT.FAILED":
		// Subscription payment failed
		return s.handleSubscriptionPaymentFailed(ctx, webhookEvent)
	default:
		log.Info(ctx, "Unhandled PayPal webhook event: "+eventType)
	}

	return nil
}

// handleOrderApproved handles CHECKOUT.ORDER.APPROVED event and captures payment
func (s *paypalService) handleOrderApproved(ctx *gin.Context, event map[string]interface{}) error {
	// Extract order information
	resource, ok := event["resource"].(map[string]interface{})
	if !ok {
		return errors.New("invalid webhook payload: missing resource")
	}

	// Get PayPal order ID
	paypalOrderID, ok := resource["id"].(string)
	if !ok || paypalOrderID == "" {
		return errors.New("missing paypal order id in webhook")
	}

	// Get custom_id from purchase_units
	purchaseUnits, ok := resource["purchase_units"].([]interface{})
	if !ok || len(purchaseUnits) == 0 {
		return errors.New("missing purchase_units in webhook")
	}

	firstUnit := purchaseUnits[0].(map[string]interface{})
	customID, ok := firstUnit["custom_id"].(string)
	if !ok || customID == "" {
		return errors.New("missing custom_id in purchase unit")
	}

	log.Info(ctx, fmt.Sprintf("Order %s approved, capturing payment for PayPal order %s", customID, paypalOrderID))

	// Find transaction first to get site_id
	transaction, err := s.paymentTransactionRepo.GetByTransactionID(ctx, customID)
	if err != nil {
		log.Error(ctx, "Failed to get transaction: "+err.Error())
		return err
	}
	if transaction == nil {
		return errors.New("transaction not found")
	}

	// Get PayPal configuration to capture payment
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, transaction.SiteID, "paypal")
	if err != nil {
		log.Error(ctx, "Failed to get PayPal config: "+err.Error())
		return err
	}
	if config == nil {
		return errors.New("paypal config not found")
	}

	// Capture payment using PayPal API
	captureResp, err := s.capturePayment(ctx, transaction.SiteID, config.PaypalClientID, config.PaypalClientSecret, paypalOrderID)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("Failed to capture payment for order %s: %v", customID, err))
		// Don't fail the webhook, payment might be captured later
		return nil
	}

	log.Info(ctx, fmt.Sprintf("Successfully captured payment for order %s, capture status: %s", customID, captureResp.Status))

	// Just update transaction with provider payment ID, don't process coins yet
	// The PAYMENT.CAPTURE.COMPLETED webhook will handle coin distribution
	if captureResp.Status == "COMPLETED" {
		transaction.ProviderPaymentID = captureResp.ID
		err = s.paymentTransactionRepo.Update(ctx, transaction)
		if err != nil {
			log.Error(ctx, "Failed to update transaction: "+err.Error())
			return err
		}
		log.Info(ctx, fmt.Sprintf("Payment captured for order %s, waiting for PAYMENT.CAPTURE.COMPLETED webhook", customID))
	}

	return nil
}

// capturePayment captures payment for a PayPal order
func (s *paypalService) capturePayment(ctx *gin.Context, siteID, clientID, clientSecret, orderID string) (*payPalCaptureResponse, error) {
	apiURL := s.getPayPalURL(ctx, siteID) + "/v2/checkout/orders/" + orderID + "/capture"

	// Get access token
	accessToken, err := s.getPayPalAccessToken(ctx, siteID, clientID, clientSecret)
	if err != nil {
		return nil, err
	}

	// Create capture request
	captureReq := map[string]interface{}{
		"payment_source": map[string]interface{}{
			"type": "paypal",
		},
	}

	body, _ := json.Marshal(captureReq)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(body))
	httpReq.Header.Add("Authorization", "Bearer "+accessToken)
	httpReq.Header.Add("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 10 * time.Second}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusCreated && httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("failed to capture payment: %s", string(respBody))
	}

	var captureResp payPalCaptureResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&captureResp); err != nil {
		return nil, err
	}

	return &captureResp, nil
}

func (s *paypalService) handlePaymentCompleted(ctx *gin.Context, event map[string]interface{}) error {
	// Extract order information
	resource, ok := event["resource"].(map[string]interface{})
	if !ok {
		return errors.New("invalid webhook payload: missing resource")
	}

	customID, ok := resource["custom_id"].(string)
	if !ok || customID == "" {
		return errors.New("missing custom_id in webhook")
	}

	// Extract amount safely
	var totalAmount string
	if amount, ok := resource["amount"].(map[string]interface{}); ok {
		if value, ok := amount["value"].(string); ok {
			totalAmount = value
		}
	}

	log.Info(ctx, fmt.Sprintf("Processing PayPal payment for order %s, amount %s", customID, totalAmount))

	// Find transaction by order ID
	transaction, err := s.paymentTransactionRepo.GetByTransactionID(ctx, customID)
	if err != nil {
		return err
	}
	if transaction == nil {
		return errors.New("transaction not found")
	}

	// Check if payment is already processed to avoid duplicate coin distribution
	if transaction.Status == model.PaymentStatusSuccess {
		log.Info(ctx, fmt.Sprintf("Order %s already processed, skipping coin distribution", customID))
		return nil
	}

	// Update transaction status
	transaction.Status = model.PaymentStatusSuccess
	transaction.ProviderPaymentID = customID
	if payerEmail := payutil.PayPalPayerEmail(resource); payerEmail != "" {
		transaction.PayerEmail = payerEmail
	}

	err = s.paymentTransactionRepo.Update(ctx, transaction)
	if err != nil {
		return err
	}

	// Add coins to user account if this is a coin package purchase
	if transaction.PaymentType == model.PaymentTypeCoinPackage {
		// Get coin amount from snapshot
		coinAmount, ok := transaction.Snapshot["coin_amount"]
		if !ok {
			log.Error(ctx, "Missing coin_amount in transaction snapshot")
			return errors.New("missing coin_amount in transaction snapshot")
		}

		// Convert to int for coin operations
		coinAmountFloat, ok := coinAmount.(float64)
		if !ok {
			log.Error(ctx, "Invalid coin_amount type in transaction snapshot")
			return errors.New("invalid coin_amount type")
		}
		coinAmountInt := int(coinAmountFloat)

		addParams := &coins.CoinAdditionParams{
			UserID:      transaction.UserID,
			SiteID:      transaction.SiteID,
			Amount:      coinAmountInt,
			Source:      "coin_package",
			RelatedID:   transaction.TransactionID,
			Description: "Purchase coins package",
		}
		_, err = s.coinsService.AddCoins(ctx, addParams)
		if err != nil {
			log.Error(ctx, "Failed to add coins: "+err.Error())
			return err
		}
	}

	// Process subscription payment
	if transaction.PaymentType == model.PaymentTypeSubscription {
		err = s.processSubscriptionPayment(ctx, customID, transaction)
		if err != nil {
			log.Error(ctx, "Failed to process subscription payment: "+err.Error())
			return err
		}
	}

	log.Info(ctx, fmt.Sprintf("Successfully processed PayPal payment for order %s, amount %s", customID, totalAmount))
	s.emitTrackPurchase(transaction)
	return nil
}

func (s *paypalService) emitTrackPurchase(transaction *model.PaymentTransaction) {
	if s.trackingService == nil || transaction == nil {
		return
	}
	go func() {
		trackingCtx := context.Background()
		if trackErr := s.trackingService.TrackPurchase(trackingCtx, transaction); trackErr != nil {
			fmt.Printf("PayPal 打点事件发送失败: %v\n", trackErr)
		}
	}()
}

// processSubscriptionPayment handles successful subscription payments
func (s *paypalService) processSubscriptionPayment(ctx *gin.Context, paypalOrderID string, transaction *model.PaymentTransaction) error {
	// Calculate subscription period based on package
	pkg, err := s.subscriptionPackageRepo.GetByPackageID(ctx, transaction.RelatedID)
	if err != nil {
		log.Error(ctx, "Failed to get subscription package: "+err.Error())
		return err
	}

	if pkg == nil {
		return errors.New("subscription package not found")
	}

	// Calculate start and end time based on interval
	startTime := time.Now()
	var endTime time.Time

	switch pkg.Interval {
	case "week":
		endTime = startTime.AddDate(0, 0, 7) // 7 days
	case "month":
		endTime = startTime.AddDate(0, 1, 0) // 1 month
	case "year":
		endTime = startTime.AddDate(1, 0, 0) // 1 year
	default:
		endTime = startTime.AddDate(0, 1, 0) // Default to 1 month
	}

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

	// Create a record in the user_subscriptions table to track the subscription details
	userSubscription := &model.UserSubscription{
		SubscriptionID:         uuid.New().String(),
		UserID:                 transaction.UserID,
		SiteID:                 transaction.SiteID,
		PackageID:              transaction.RelatedID,
		Provider:               transaction.Provider,
		ProviderSubscriptionID: paypalOrderID,
		Status:                 1, // Active
		CurrentPeriodStart:     startTime,
		CurrentPeriodEnd:       endTime,
	}

	if err := s.userSubscriptionRepo.Create(ctx, userSubscription); err != nil {
		return fmt.Errorf("failed to save user subscription details: %w", err)
	}

	// Update user's total spending
	_, err = s.userCoinsRepository.UpdateBalance(ctx, transaction.UserID, transaction.SiteID, 0, transaction.Amount)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	// Grant coins from subscription package if configured
	if pkg.Coins > 0 {
		_, err = s.userCoinsRepository.AddPresentCoins(ctx, transaction.UserID, transaction.SiteID, pkg.Coins)
		if err != nil {
			log.Error(ctx, fmt.Sprintf("Failed to grant present coins: %s", err.Error()))
		} else {
			log.Info(ctx, fmt.Sprintf("Granted %d coins for subscription %s", pkg.Coins, pkg.PackageID))
		}
	}

	log.Info(ctx, fmt.Sprintf("Successfully processed subscription payment for user %s, package %s", transaction.UserID, pkg.Name))
	return nil
}

func (s *paypalService) handlePaymentDenied(ctx *gin.Context, event map[string]interface{}) error {
	resource, _ := event["resource"].(map[string]interface{})
	customID, _ := resource["custom_id"].(string)

	if customID == "" {
		return errors.New("missing custom_id in webhook")
	}

	// Find and update transaction
	transaction, err := s.paymentTransactionRepo.GetByTransactionID(ctx, customID)
	if err != nil {
		return err
	}
	if transaction == nil {
		return errors.New("transaction not found")
	}

	transaction.Status = model.PaymentStatusFailed
	err = s.paymentTransactionRepo.Update(ctx, transaction)
	if err != nil {
		return err
	}

	log.Info(ctx, fmt.Sprintf("PayPal payment denied for order %s", customID))
	return nil
}

// CreateSubscriptionOrder creates a new PayPal subscription order
func (s *paypalService) CreateSubscriptionOrder(ctx *gin.Context, userID string, req api.OrderCreateRequest) (*api.OrderCreateResponse, error) {
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

	// Get PayPal configuration
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, req.SiteID, "paypal")
	if err != nil {
		return nil, err
	}

	if config == nil || !config.IsActive || config.PaypalClientID == "" {
		return nil, common.ErrPaymentProviderNotConfigured
	}

	// Set PayPal credentials
	s.paypalClient.SetCredentials(config.PaypalClientID, config.PaypalClientSecret, config.IsSandbox)

	// Check if the user already has an active subscription
	sub, err := s.userSubscriptionRepo.GetActiveByUserAndSite(ctx, userID, req.SiteID)
	if err != nil {
		return nil, err
	}

	if sub != nil {
		// Check if subscription is still active with PayPal
		subDetails, err := s.paypalClient.GetSubscriptionDetails(sub.ProviderSubscriptionID)
		if err == nil && subDetails != nil {
			if status, ok := subDetails["status"].(string); ok && status == "ACTIVE" {
				log.Error(ctx, "User already has an active subscription: "+userID)
				return nil, errors.New("user already has an active subscription")
			}
		}
	}

	// Generate transaction ID
	transactionID := uuid.New().String()

	// Build success and cancel URLs, always attaching siteId & orderId
	var successURL, cancelURL string
	if req.ReturnURL == "" {
		// Use default success URL from config
		successURL = s.paymentSuccessURL(s.conf.GetString("webhook.paypal.success_url"), req.SiteID, transactionID)
	} else {
		// Use custom frontend URL as base, but still append order info
		successURL = s.paymentSuccessURL(req.ReturnURL, req.SiteID, transactionID)
	}

	if req.CancelURL == "" {
		// Use default cancel URL from config
		cancelURL = s.paymentCancelURL(s.conf.GetString("webhook.paypal.cancel_url"), req.SiteID, transactionID)
	} else {
		// Use custom frontend URL as base, but still append order info
		cancelURL = s.paymentCancelURL(req.CancelURL, req.SiteID, transactionID)
	}

	// Set up metadata
	metadata := map[string]string{
		"site_id":        req.SiteID,
		"user_id":        userID,
		"package_type":   "subscription",
		"transaction_id": transactionID,
		"package_id":     pkg.PackageID,
		"interval":       pkg.Interval,
	}

	// Create PayPal product first
	productID, err := s.paypalClient.CreateProduct(
		pkg.Name,
		pkg.Description,
	)
	if err != nil {
		log.Error(ctx, "Failed to create PayPal product: "+err.Error())
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	log.Info(ctx, "Created PayPal product: "+productID)

	// Create PayPal subscription plan
	planID, err := s.paypalClient.CreateSubscriptionPlan(
		productID,
		pkg.Name,
		pkg.Description,
		pkg.Price,
		pkg.Currency,
		pkg.Interval,
	)
	if err != nil {
		log.Error(ctx, "Failed to create PayPal subscription plan: "+err.Error())
		return nil, fmt.Errorf("failed to create subscription plan: %w", err)
	}

	log.Info(ctx, "Created PayPal subscription plan: "+planID)

	// Create PayPal subscription
	subscriptionID, approvalURL, err := s.paypalClient.CreateSubscription(
		planID,
		successURL,
		cancelURL,
		metadata,
	)
	if err != nil {
		log.Error(ctx, "Failed to create PayPal subscription: "+err.Error())
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	log.Info(ctx, "Created PayPal subscription: "+subscriptionID)

	// Create snapshot of package at time of purchase
	snapshot := model.JSONMap{
		"package_id":          pkg.PackageID,
		"name":                pkg.Name,
		"description":         pkg.Description,
		"interval":            pkg.Interval,
		"price":               pkg.Price,
		"currency":            pkg.Currency,
		"original_price":      pkg.OriginalPrice,
		"discount_percentage": pkg.DiscountPercentage,
		"coins":               pkg.Coins,
		"rights":              pkg.Rights,
		"paypal_plan_id":      planID,
	}
	if len(req.TrackingContext) > 0 {
		snapshot["tracking_context"] = req.TrackingContext
	}
	snapshot = analytics.MergeMetaIntoSnapshot(snapshot, req.Meta)

	// Create transaction record
	tx := &model.PaymentTransaction{
		TransactionID:     transactionID,
		OrderID:           transactionID,
		UserID:            userID,
		SiteID:            req.SiteID,
		Amount:            pkg.Price,
		Currency:          pkg.Currency,
		Provider:          "paypal",
		ProviderPaymentID: subscriptionID,
		PaymentType:       model.PaymentTypeSubscription,
		Status:            model.PaymentStatusPending,
		RelatedID:         pkg.PackageID,
		RelatedType:       model.RelatedTypeSubscription,
		Snapshot:          snapshot,
	}

	err = s.paymentTransactionRepo.Create(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction record: %w", err)
	}

	return &api.OrderCreateResponse{
		OrderID:       transactionID,
		CheckoutURL:   approvalURL,
		PaymentStatus: "pending",
		SuccessURL:    successURL,
		CancelURL:     cancelURL,
	}, nil
}

// CancelSubscription cancels a PayPal subscription
func (s *paypalService) CancelSubscription(ctx context.Context, userID string, subscriptionID string, cancelAtPeriodEnd bool) error {
	// Get user subscription
	subscription, err := s.userSubscriptionRepo.GetBySubscriptionID(ctx, subscriptionID)
	if err != nil {
		return err
	}
	if subscription == nil {
		return errors.New("subscription not found")
	}

	// Check if subscription belongs to user
	if subscription.UserID != userID {
		return errors.New("subscription does not belong to user")
	}

	// Get PayPal configuration
	config, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, subscription.SiteID, "paypal")
	if err != nil {
		return err
	}

	if config == nil || !config.IsActive || config.PaypalClientID == "" {
		return common.ErrPaymentProviderNotConfigured
	}

	// Set PayPal credentials
	s.paypalClient.SetCredentials(config.PaypalClientID, config.PaypalClientSecret, config.IsSandbox)

	// Cancel subscription with PayPal API
	if !cancelAtPeriodEnd {
		// Cancel immediately
		err = s.paypalClient.CancelSubscription(subscription.ProviderSubscriptionID)
		if err != nil {
			return fmt.Errorf("failed to cancel subscription with PayPal: %w", err)
		}

		// Update subscription status in database
		subscription.Status = 3 // Cancelled
	} else {
		// Set cancel_at_period_end flag
		subscription.CancelAtPeriodEnd = true
	}

	err = s.userSubscriptionRepo.Update(ctx, subscription)
	if err != nil {
		return errors.New("failed to update subscription: " + err.Error())
	}

	return nil
}

// GetUserPurchases retrieves user's PayPal purchase history
func (s *paypalService) GetUserPurchases(ctx context.Context, userID string, siteID string, page, pageSize int) ([]*api.PurchaseRecord, int64, error) {
	// Get PayPal transactions for user using ListUserPurchases
	transactions, total, err := s.paymentTransactionRepo.ListUserPurchases(ctx, userID, siteID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Filter for PayPal transactions only and convert to API response
	records := make([]*api.PurchaseRecord, 0, len(transactions))
	for _, txn := range transactions {
		// Only include PayPal transactions
		if txn.Provider != "paypal" {
			continue
		}

		record := &api.PurchaseRecord{
			TransactionID: txn.TransactionID,
			OrderID:       txn.OrderID,
			Amount:        types.Money(txn.Amount),
			Currency:      txn.Currency,
			Provider:      txn.Provider,
			PaymentType:   txn.PaymentType,
			Status:        txn.Status,
			RelatedID:     txn.RelatedID,
			RelatedType:   txn.RelatedType,
			CreatedAt:     txn.CreatedAt.Unix(),
		}

		// Set payment type name
		if txn.PaymentType == model.PaymentTypeSubscription {
			record.PaymentTypeName = "Subscription"
		} else if txn.PaymentType == model.PaymentTypeCoinPackage {
			record.PaymentTypeName = "Coin Package"
		} else {
			record.PaymentTypeName = "Other"
		}

		// Set status name
		switch txn.Status {
		case model.PaymentStatusPending:
			record.StatusName = "Pending"
		case model.PaymentStatusSuccess:
			record.StatusName = "Completed"
		case model.PaymentStatusFailed:
			record.StatusName = "Failed"
		case model.PaymentStatusRefunded:
			record.StatusName = "Refunded"
		default:
			record.StatusName = "Unknown"
		}

		// Set package details from snapshot if available
		if txn.Snapshot != nil {
			if name, ok := txn.Snapshot["name"].(string); ok {
				record.PackageName = name
			}
			if desc, ok := txn.Snapshot["description"].(string); ok {
				record.PackageDescription = desc
			}
		}

		records = append(records, record)
	}

	return records, total, nil
}

// GetConfigInfo retrieves PayPal configuration information for a site
func (s *paypalService) GetConfigInfo(ctx *gin.Context, siteID string) (*api.PaymentConfigInfoResponse, error) {
	response := &api.PaymentConfigInfoResponse{}

	// Get PayPal configuration
	paypalConfig, err := s.paymentConfigRepo.GetBySiteIDAndProvider(ctx, siteID, "paypal")
	if err != nil {
		log.Warning(ctx, "Failed to get PayPal config: "+err.Error())
		return response, nil // Return empty response instead of error
	}

	if paypalConfig != nil && paypalConfig.IsActive {
		response.PayPal = &api.PayPalConfigInfo{
			ClientID:     paypalConfig.PaypalClientID,
			ClientSecret: paypalConfig.PaypalClientSecret,
			IsSandbox:    paypalConfig.IsSandbox,
		}
	}

	return response, nil
}

func (s *paypalService) paymentSuccessURL(baseUrl string, siteID string, orderID string) string {
	if baseUrl == "" {
		return ""
	}
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

func (s *paypalService) paymentCancelURL(baseUrl string, siteID string, orderID string) string {
	if baseUrl == "" {
		return ""
	}
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

// handleSubscriptionActivated handles BILLING.SUBSCRIPTION.ACTIVATED event
func (s *paypalService) handleSubscriptionActivated(ctx *gin.Context, event map[string]interface{}) error {
	resource, ok := event["resource"].(map[string]interface{})
	if !ok {
		return errors.New("invalid webhook payload: missing resource")
	}

	subscriptionID, ok := resource["id"].(string)
	if !ok || subscriptionID == "" {
		return errors.New("missing subscription id in webhook")
	}

	customID, _ := resource["custom_id"].(string)
	log.Info(ctx, fmt.Sprintf("PayPal subscription activated: %s, custom_id: %s", subscriptionID, customID))

	// Find transaction by custom_id if available
	if customID != "" {
		transaction, err := s.paymentTransactionRepo.GetByTransactionID(ctx, customID)
		if err != nil {
			log.Error(ctx, "Failed to get transaction: "+err.Error())
			return err
		}

		if transaction != nil && transaction.Status == model.PaymentStatusPending {
			// Update transaction status
			transaction.Status = model.PaymentStatusSuccess
			transaction.ProviderPaymentID = subscriptionID
			if payerEmail := payutil.PayPalPayerEmail(resource); payerEmail != "" {
				transaction.PayerEmail = payerEmail
			}

			err = s.paymentTransactionRepo.Update(ctx, transaction)
			if err != nil {
				log.Error(ctx, "Failed to update transaction: "+err.Error())
				return err
			}

			// Process subscription payment
			err = s.processSubscriptionPayment(ctx, subscriptionID, transaction)
			if err != nil {
				log.Error(ctx, "Failed to process subscription payment: "+err.Error())
				return err
			}
			s.emitTrackPurchase(transaction)
		}
	}

	return nil
}

// handleSubscriptionCreated handles BILLING.SUBSCRIPTION.CREATED event
func (s *paypalService) handleSubscriptionCreated(ctx *gin.Context, event map[string]interface{}) error {
	resource, ok := event["resource"].(map[string]interface{})
	if !ok {
		return errors.New("invalid webhook payload: missing resource")
	}

	subscriptionID, ok := resource["id"].(string)
	if !ok || subscriptionID == "" {
		return errors.New("missing subscription id in webhook")
	}

	customID, _ := resource["custom_id"].(string)
	status, _ := resource["status"].(string)

	log.Info(ctx, fmt.Sprintf("PayPal subscription created: %s, custom_id: %s, status: %s", subscriptionID, customID, status))

	// Check if subscription is already active (some cases it's created in ACTIVE status)
	if status == "ACTIVE" && customID != "" {
		log.Info(ctx, "Subscription is already active, processing payment immediately")

		transaction, err := s.paymentTransactionRepo.GetByTransactionID(ctx, customID)
		if err != nil {
			log.Error(ctx, "Failed to get transaction: "+err.Error())
			return err
		}

		if transaction != nil && transaction.Status == model.PaymentStatusPending {
			// Update transaction status
			transaction.Status = model.PaymentStatusSuccess
			transaction.ProviderPaymentID = subscriptionID
			if payerEmail := payutil.PayPalPayerEmail(resource); payerEmail != "" {
				transaction.PayerEmail = payerEmail
			}

			err = s.paymentTransactionRepo.Update(ctx, transaction)
			if err != nil {
				log.Error(ctx, "Failed to update transaction: "+err.Error())
				return err
			}

			// Process subscription payment
			err = s.processSubscriptionPayment(ctx, subscriptionID, transaction)
			if err != nil {
				log.Error(ctx, "Failed to process subscription payment: "+err.Error())
				return err
			}
			s.emitTrackPurchase(transaction)
		}
	}

	return nil
}

// handleSubscriptionCancelled handles BILLING.SUBSCRIPTION.CANCELLED event
func (s *paypalService) handleSubscriptionCancelled(ctx *gin.Context, event map[string]interface{}) error {
	resource, ok := event["resource"].(map[string]interface{})
	if !ok {
		return errors.New("invalid webhook payload: missing resource")
	}

	subscriptionID, ok := resource["id"].(string)
	if !ok || subscriptionID == "" {
		return errors.New("missing subscription id in webhook")
	}

	log.Info(ctx, fmt.Sprintf("PayPal subscription cancelled: %s", subscriptionID))

	// Find and update user subscription
	subscription, err := s.userSubscriptionRepo.GetByProviderSubscriptionID(ctx, "paypal", subscriptionID)
	if err != nil {
		log.Error(ctx, "Failed to get user subscription: "+err.Error())
		return err
	}

	if subscription != nil {
		subscription.Status = 3 // Cancelled
		err = s.userSubscriptionRepo.Update(ctx, subscription)
		if err != nil {
			log.Error(ctx, "Failed to update subscription: "+err.Error())
			return err
		}
	}

	return nil
}

// handleSubscriptionPaymentFailed handles BILLING.SUBSCRIPTION.PAYMENT.FAILED event
func (s *paypalService) handleSubscriptionPaymentFailed(ctx *gin.Context, event map[string]interface{}) error {
	resource, ok := event["resource"].(map[string]interface{})
	if !ok {
		return errors.New("invalid webhook payload: missing resource")
	}

	subscriptionID, ok := resource["id"].(string)
	if !ok || subscriptionID == "" {
		return errors.New("missing subscription id in webhook")
	}

	log.Info(ctx, fmt.Sprintf("PayPal subscription payment failed: %s", subscriptionID))

	// Find and update user subscription
	subscription, err := s.userSubscriptionRepo.GetByProviderSubscriptionID(ctx, "paypal", subscriptionID)
	if err != nil {
		log.Error(ctx, "Failed to get user subscription: "+err.Error())
		return err
	}

	if subscription != nil {
		// Optionally suspend the subscription or notify the user
		subscription.Status = 2 // Suspended
		err = s.userSubscriptionRepo.Update(ctx, subscription)
		if err != nil {
			log.Error(ctx, "Failed to update subscription: "+err.Error())
			return err
		}
	}

	return nil
}
