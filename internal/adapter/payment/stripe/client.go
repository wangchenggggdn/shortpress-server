package stripe

import (
	"fmt"
	"log"
	"strings"
	"sync"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/balance"
	"github.com/stripe/stripe-go/v82/charge"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"
	"github.com/stripe/stripe-go/v82/webhookendpoint"
)

// Client 定义Stripe客户端接口
type StripeClient interface {
	SetAPIKey(key string)
	GetAPIKey() string
	TestConf(key string) error
	GetCharges(limit int64) []*stripe.Charge
	GetAccountInfo() (*stripe.Account, error)
	CreateProduct(name string, description string, metadata map[string]string) (string, error)
	CreatePrice(amount int64, currency string, productID string, metadata map[string]string) (string, error)
	DeleteProduct(productID string) error
	CreateOneTimePrice(amount int64, currency string, productID string) (*stripe.Price, error)
	CreateRecurringPrice(amount int64, currency string, productID string, interval string) (string, error)
	CreatePaymentIntent(amount int64, currency, description string, metadata map[string]string) (string, string, error)
	CreateCheckoutSession(priceID, successURL, cancelURL string, metadata map[string]string) (string, string, error)
	CreateCheckoutSessionWithPayment(
		productName string,
		description string,
		amount int64,
		currency string,
		successURL string,
		cancelURL string,
		metadata map[string]string,
	) (string, string, error)
	CreateSubscriptionCheckoutSession(
		productName string,
		description string,
		amount int64,
		currency string,
		interval string,
		successURL string,
		cancelURL string,
		metadata map[string]string,
	) (string, string, error)
	CancelSubscription(subscriptionID string) error
	SetWebhookEndpoint(url string) (string, string, error)
	DeleteWebhookEndpoint(endpointID string) error
	ValidateWebhookSignature(payload []byte, signature string, secret string) (stripe.Event, error)
	RetrieveSession(sessionID string) (*stripe.CheckoutSession, error)
	GetPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error)
	GetShotPressEndpoint(fixWebHook string) (string, string, error)
	IsSubscriptionActive(subscriptionID string) (bool, error)
	GetSubscriptionStartEnd(subscriptionID string) (int64, int64, error)
}

// stripeClient 实现Stripe客户端接口
type stripeClient struct {
	apiKey string
	mutex  sync.Mutex // 添加互斥锁
}

// NewClient 创建新的Stripe客户端
func NewClient() StripeClient {
	return &stripeClient{}
}

// SetAPIKey 设置API密钥
func (c *stripeClient) SetAPIKey(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.apiKey = key
}

// GetAPIKey 获取API密钥
func (c *stripeClient) GetAPIKey() string {
	return stripe.Key
}

// GetBalance 获取当前账号余额信息
func (c *stripeClient) GetBalance() (*stripe.Balance, error) {
	return balance.Get(nil)
}

// GetCharges 获取最近的支付记录
func (c *stripeClient) GetCharges(limit int64) []*stripe.Charge {
	params := &stripe.ChargeListParams{}
	params.Limit = stripe.Int64(limit)

	var charges []*stripe.Charge
	i := charge.List(params)
	for i.Next() {
		charges = append(charges, i.Charge())
	}

	if err := i.Err(); err != nil {
		log.Printf("Error listing charges: %v", err)
	}

	return charges
}

// GetPaymentIntents 获取最近的支付意向
func (c *stripeClient) GetPaymentIntents(limit int64) []*stripe.PaymentIntent {
	params := &stripe.PaymentIntentListParams{}
	params.Limit = stripe.Int64(limit)

	var paymentIntents []*stripe.PaymentIntent
	i := paymentintent.List(params)
	for i.Next() {
		paymentIntents = append(paymentIntents, i.PaymentIntent())
	}

	if err := i.Err(); err != nil {
		log.Printf("Error listing payment intents: %v", err)
	}

	return paymentIntents
}

// GetAccountInfo 获取当前Stripe账号信息，包括邮箱、名称等
func (c *stripeClient) GetAccountInfo() (*stripe.Account, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	return account.Get()
}

func (c *stripeClient) TestConf(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	originalKey := stripe.Key
	stripe.Key = c.apiKey
	defer func() { stripe.Key = originalKey }()

	_, err := account.Get()
	return err
}

// CreateProduct creates a new product in Stripe with metadata
func (c *stripeClient) CreateProduct(name string, description string, metadata map[string]string) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	params := &stripe.ProductParams{
		Name: stripe.String(name),
	}

	if description != "" {
		params.Description = stripe.String(description)
	}

	if metadata != nil {
		params.Metadata = metadata
	}

	p, err := product.New(params)
	if err != nil {
		return "", err
	}

	return p.ID, nil
}

// CreatePrice creates a new price in Stripe with metadata
func (c *stripeClient) CreatePrice(amount int64, currency string, productID string, metadata map[string]string) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	params := &stripe.PriceParams{
		Currency:   stripe.String(currency),
		Product:    stripe.String(productID),
		UnitAmount: stripe.Int64(amount),
	}

	if metadata != nil {
		params.Metadata = metadata
	}

	p, err := price.New(params)
	if err != nil {
		return "", err
	}

	return p.ID, nil
}

// DeleteProduct deletes a product in Stripe
func (c *stripeClient) DeleteProduct(productID string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	_, err := product.Del(productID, nil)
	return err
}

// CreateOneTimePrice creates a new price in Stripe
func (c *stripeClient) CreateOneTimePrice(amount int64, currency string, productID string) (*stripe.Price, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	params := &stripe.PriceParams{
		Currency:   stripe.String(currency),
		Product:    stripe.String(productID),
		UnitAmount: stripe.Int64(amount),
	}
	// One-time purchase doesn't need the recurring parameter
	return price.New(params)
}

// CreateRecurringPrice creates a new recurring price in Stripe with specified billing interval
func (c *stripeClient) CreateRecurringPrice(amount int64, currency string, productID string, interval string) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	params := &stripe.PriceParams{
		Currency:   stripe.String(currency),
		Product:    stripe.String(productID),
		UnitAmount: stripe.Int64(amount),
		Recurring: &stripe.PriceRecurringParams{
			Interval: stripe.String(interval),
		},
	}
	price, err := price.New(params)
	if err != nil {
		return "", err
	}
	return price.ID, nil
}

// CreatePaymentIntent creates a payment intent for direct payment processing
func (c *stripeClient) CreatePaymentIntent(amount int64, currency, description string, metadata map[string]string) (string, string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
	}

	if description != "" {
		params.Description = stripe.String(description)
	}

	if metadata != nil {
		params.Metadata = metadata
	}

	// Set payment method types to card only for now
	params.PaymentMethodTypes = []*string{stripe.String("card")}

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", "", err
	}

	return pi.ID, pi.ClientSecret, nil
}

// CreateCheckoutSession creates a checkout session for hosted payment pages
func (c *stripeClient) CreateCheckoutSession(priceID, successURL, cancelURL string, metadata map[string]string) (string, string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
	}

	//TESTDATA
	if metadata != nil {
		params.Metadata = metadata
	}

	s, err := session.New(params)
	if err != nil {
		return "", "", err
	}
	return s.ID, s.URL, nil
}

// SetWebhookEndpoint creates a webhook endpoint for Stripe events
func (c *stripeClient) SetWebhookEndpoint(url string) (string, string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	params := &stripe.WebhookEndpointParams{
		URL: stripe.String(url),
	}
	// Define the payment-related events we want to listen for
	events := []string{
		"payment_intent.succeeded",
		"payment_intent.payment_failed",
		"checkout.session.completed",
		"checkout.session.expired",
		"charge.succeeded",
		"charge.failed",
		"charge.refunded",
		"invoice.payment_succeeded",
		"invoice.payment_failed",
		"invoice.paid",
		"customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted",
	}

	for _, event := range events {
		params.EnabledEvents = append(params.EnabledEvents, stripe.String(event))
	}

	// Create the webhook endpoint
	endpoint, err := webhookendpoint.New(params)
	if err != nil {
		return "", "", err
	}

	return endpoint.ID, endpoint.Secret, nil
}

// DeleteWebhookEndpoint deletes a webhook endpoint
func (c *stripeClient) DeleteWebhookEndpoint(endpointID string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	_, err := webhookendpoint.Del(endpointID, nil)
	return err
}

// ValidateWebhookSignature validates the webhook signature
func (c *stripeClient) ValidateWebhookSignature(payload []byte, signature string, secret string) (stripe.Event, error) {

	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	//TODO 忽略了版本好的差异,
	return webhook.ConstructEventWithOptions(
		payload,
		signature,
		secret,
		webhook.ConstructEventOptions{
			// Set the Stripe API version to the one you are using
			IgnoreAPIVersionMismatch: true,
		},
	)
}

// RetrieveSession retrieves a checkout session
func (c *stripeClient) RetrieveSession(sessionID string) (*stripe.CheckoutSession, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	return session.Get(sessionID, nil)
}

// GetPaymentIntent retrieves a payment intent by ID
func (c *stripeClient) GetPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	return paymentintent.Get(paymentIntentID, nil)
}

// CreateCheckoutSessionWithPrice creates a checkout session with a price
func (c *stripeClient) CreateCheckoutSessionWithPayment(
	productName string,
	description string,
	amount int64,
	currency string,
	successURL string,
	cancelURL string,
	metadata map[string]string,
) (string, string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	sessionParams := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(currency),
					UnitAmount: stripe.Int64(amount),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(productName),
						Description: stripe.String(productName + " " + description),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
	}

	// Session 与 PaymentIntent 都写入 metadata，便于 checkout.session.completed / payment_intent.succeeded 定位订单
	if metadata != nil {
		sessionParams.Metadata = metadata
		sessionParams.PaymentIntentData = &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: metadata,
		}
	}

	s, err := session.New(sessionParams)
	if err != nil {
		return "", "", err
	}
	if s.Subscription != nil {
		log.Printf("******* Subscription ID: %s", s.Subscription.ID)
	}
	return s.ID, s.URL, nil
}

// GetWebhookEndpoints retrieves all webhook endpoints configured in the Stripe account
func (c *stripeClient) GetShotPressEndpoint(fixWebHook string) (string, string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	params := &stripe.WebhookEndpointListParams{}
	var endpoints []*stripe.WebhookEndpoint

	i := webhookendpoint.List(params)
	for i.Next() {
		endpoints = append(endpoints, i.WebhookEndpoint())
	}

	if err := i.Err(); err != nil {
		return "", "", err
	}
	for _, endpoint := range endpoints {
		if endpoint.URL == fixWebHook {
			return endpoint.ID, endpoint.Secret, nil
		}
	}
	return "", "", nil
}

func (c *stripeClient) CreateSubscriptionCheckoutSession(
	productName string,
	description string,
	amount int64,
	currency string,
	interval string,
	successURL string,
	cancelURL string,
	metadata map[string]string,
) (string, string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	sessionParams := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(currency),
					UnitAmount: stripe.Int64(amount),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(productName),
						Description: stripe.String(productName + " " + description),
					},
					Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
						Interval: stripe.String(interval),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
	}

	// 保持元数据一致
	if metadata != nil {
		sessionParams.Metadata = metadata
	}

	s, err := session.New(sessionParams)
	if err != nil {
		return "", "", err
	}

	return s.ID, s.URL, nil
}

func (c *stripeClient) CancelSubscription(subscriptionID string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	// 	CancelAtPeriodEnd: stripe.Bool(cancelAtPeriodEnd),

	// 在周期结束后取消
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	_, err := subscription.Update(subscriptionID, params)

	if err != nil {
		// Check if the error is a Stripe error
		if stripeErr, ok := err.(*stripe.Error); ok {

			if stripeErr.Code == stripe.ErrorCodeResourceMissing {
				return nil // Treat as success (idempotent)
			}

			if stripeErr.Type == stripe.ErrorTypeInvalidRequest {
				// Note: Using strings.ToLower and strings.Contains requires importing the "strings" package.
				msgLower := strings.ToLower(stripeErr.Msg)
				if strings.Contains(msgLower, "already canceled") ||
					strings.Contains(msgLower, "has been canceled") ||
					strings.Contains(msgLower, "cannot cancel a canceled subscription") {
					return nil // Already canceled, so treat as success
				}
			}
		}

		return err
	}
	return nil // Successfully canceled or was already in a non-active/canceled state
}

// IsSubscriptionActive checks if a subscription is currently active.
func (c *stripeClient) IsSubscriptionActive(subscriptionID string) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey

	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return false, err
	}

	return sub.Status == stripe.SubscriptionStatusActive, nil
}

func (c *stripeClient) GetSubscriptionStartEnd(subscriptionID string) (int64, int64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	stripe.Key = c.apiKey
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return 0, 0, err
	}
	if len(sub.Items.Data) == 0 {
		return 0, 0, fmt.Errorf("subscription %s has no items", subscriptionID)
	}
	return sub.Items.Data[0].CurrentPeriodStart, sub.Items.Data[0].CurrentPeriodEnd, nil
}
