package paypal

import (
	"context"

	"shortpress-server/internal/api"

	"github.com/gin-gonic/gin"
)

// PaypalService defines the interface for PayPal payment service
type PaypalService interface {
	GetAccountInfo(ctx context.Context, siteID string) (*api.PaymentAccountInfo, error)
	ConfTest(ctx context.Context, clientSecret string) error
	SaveConfig(ctx *gin.Context, config api.PaymentProviderConfig) error
	CreateCoinPackage(ctx context.Context, req api.CoinPackageCreateRequest) (*api.CoinPackageCreateResponse, error)
	CreateOrder(ctx *gin.Context, userID string, req api.OrderCreateRequest) (*api.OrderCreateResponse, error)
	HandleWebhook(ctx *gin.Context) error
	GetConfigInfo(ctx *gin.Context, siteID string) (*api.PaymentConfigInfoResponse, error)
	CreateSubscriptionOrder(ctx *gin.Context, userID string, req api.OrderCreateRequest) (*api.OrderCreateResponse, error)
	CancelSubscription(ctx context.Context, userID string, subscriptionID string, cancelAtPeriodEnd bool) error
	GetUserPurchases(ctx context.Context, userID string, siteID string, page, pageSize int) ([]*api.PurchaseRecord, int64, error)
}
