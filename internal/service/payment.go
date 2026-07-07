package service

import (
	"context"
	"errors"
	"fmt"
	"shortpress-server/internal/model"
	paymentRepo "shortpress-server/internal/repository/db/payment"
	"shortpress-server/internal/repository/db/site"
	"shortpress-server/internal/repository/db/user"
	"shortpress-server/internal/service/payment/coins"
	"shortpress-server/pkg/iap"
	"shortpress-server/pkg/oauth2"
)

type PaymentVerifyRequest struct {
	PackageName   string `json:"packageName"`
	PurchaseToken string `json:"purchaseToken"`
	TransactionID string `json:"transactionId"`
	Account       string `json:"account"` // 通过它获取到对应的验证key

	UserID    string
	SiteID    string
	PackageID string `json:"packageId"` // 购买的商品ID,是数据库中的packageID，不是iap的ProductID
}

type PaymentVerifyAndSyncSubStatusResponse struct {
	IsActive        bool `json:"isActive"`
	Sandbox         bool `json:"sandbox"`
	IsInFreeTrial   bool `json:"isInFreeTrial"`
	AutoRenewStatus bool `json:"autoRenewStatus"`
}

type PaymentVerifyInAppPurchaseResponse struct {
	ProductID string `json:"productId"`
}

type PaymentNotifyRequest struct {
	Account string // google/apple
	Body    []byte
}

type Empty struct{}

type PaymentBiz interface {
	// 验证订阅状态
	VerifyAndSyncSubStatus(ctx context.Context, req *PaymentVerifyRequest) (*PaymentVerifyAndSyncSubStatusResponse, error)
	// 验证应用内购买
	VerifyInAppPurchase(ctx context.Context, req *PaymentVerifyRequest) (*PaymentVerifyInAppPurchaseResponse, error)
	// 支付通知处理
	Notify(ctx context.Context, req *PaymentNotifyRequest) (*Empty, error)
}

type paymentBiz struct {
	*Service
	userRepository         user.UserRepository //
	userProfileRepository  user.UserProfileRepository
	userAuthRepository     user.UserAuthRepository
	siteRepository         site.SiteRepository
	oauth2Client           oauth2.Client
	userSubscriptionRepo   paymentRepo.UserSubscriptionRepository
	coinsService           coins.CoinsService
	userCoinsRepository    paymentRepo.UserCoinsRepository
	coinPackageRepo        paymentRepo.CoinPackageRepository
	subPackageRepo         paymentRepo.SubscriptionPackageRepository
	paymentTransactionRepo paymentRepo.PaymentTransactionRepository
}

func NewPaymentBiz(
	service *Service,
	userRepository user.UserRepository,
	userProfileRepository user.UserProfileRepository,
	userAuthRepository user.UserAuthRepository,
	siteRepository site.SiteRepository,
	oauth2Client oauth2.Client,
	userSubscriptionRepo paymentRepo.UserSubscriptionRepository,
	coinsService coins.CoinsService,
	userCoinsRepository paymentRepo.UserCoinsRepository,
	coinPackageRepo paymentRepo.CoinPackageRepository,
	subPackageRepo paymentRepo.SubscriptionPackageRepository,
	paymentTransactionRepo paymentRepo.PaymentTransactionRepository,
) PaymentBiz {
	return &paymentBiz{
		Service:                service,
		userRepository:         userRepository,
		userProfileRepository:  userProfileRepository,
		userAuthRepository:     userAuthRepository,
		siteRepository:         siteRepository,
		oauth2Client:           oauth2Client,
		userSubscriptionRepo:   userSubscriptionRepo,
		coinsService:           coinsService,
		userCoinsRepository:    userCoinsRepository,
		coinPackageRepo:        coinPackageRepo,
		paymentTransactionRepo: paymentTransactionRepo,
	}
}

func (b *paymentBiz) VerifyAndSyncSubStatus(ctx context.Context, req *PaymentVerifyRequest) (*PaymentVerifyAndSyncSubStatusResponse, error) {
	service, err := b.getIapService(ctx, req.Account)
	if err != nil {
		return nil, err
	}

	res, err := service.VerifySubscription(ctx, &iap.IAPVerifyArgs{
		PackageName:   req.PackageName,
		PurchaseToken: req.PurchaseToken,
		TransactionID: req.TransactionID,
	})
	if err != nil {
		//b.logger.Errorf("sync subStatus Verify failed, req: %+v, err: %s", req.UserID, err)
		return nil, err
	}

	if res.IsActive {
		// 同步订阅状态到用户账户
		err = b.processSubscriptionPayment(ctx, req, res)
		if err != nil {
			//b.logger.Errorf("sync subStatus processSubscriptionPayment failed, req: %+v, err: %s", req.UserID, err)
			return nil, err
		}
	}

	return &PaymentVerifyAndSyncSubStatusResponse{
		IsActive:        res.IsActive,
		Sandbox:         res.Sandbox,
		IsInFreeTrial:   res.IsInFreeTrial,
		AutoRenewStatus: res.AutoRenewStatus,
	}, nil
}

func (b *paymentBiz) processSubscriptionPayment(ctx context.Context, req *PaymentVerifyRequest, res *iap.IAPVerifySubscriptionRes) error {

	sub, err := b.userSubscriptionRepo.GetBySubscriptionID(ctx, res.OriginalOrderID)
	if err != nil {
		return fmt.Errorf("failed to get user subscription: %w", err)
	}

	subPackage, err := b.subPackageRepo.GetByIOSProductID(ctx, res.ProductID)
	if err != nil {
		return fmt.Errorf("failed to get subscription package: %w", err)
	}

	userSubscription := &model.UserSubscription{
		SubscriptionID:         res.OriginalOrderID,
		UserID:                 req.UserID,
		SiteID:                 req.SiteID,           // siteID can be set if needed
		PackageID:              subPackage.PackageID, // The ID of the subscription package in your system
		Provider:               "ios",
		ProviderSubscriptionID: res.OrderID,    // IMPORTANT: This is currently the Stripe Checkout Session ID from the transaction.
		Status:                 1,              // Active
		CurrentPeriodStart:     res.StartTime,  // The start of the current billing cycle
		CurrentPeriodEnd:       res.ExpiryTime, // The end of the current billing cycle
	}

	if sub != nil {
		// Subscription already exists, no need to process again
		if err := b.userSubscriptionRepo.Update(ctx, userSubscription); err != nil {
			// This error will propagate up and cause the database transaction in handleCheckoutSessionCompleted to roll back.
			return fmt.Errorf("failed to save user subscription details: %w", err)
		}
	} else {
		// Create a record in the user_subscriptions table to track the subscription details.
		// This record is important for managing the subscription lifecycle,
		// including renewals, cancellations, and syncing with the payment provider.
		if err := b.userSubscriptionRepo.Create(ctx, userSubscription); err != nil {
			// This error will propagate up and cause the database transaction in handleCheckoutSessionCompleted to roll back.
			return fmt.Errorf("failed to save user subscription details: %w", err)
		}
	}

	// Get the user to check if they already have premium status
	user, err := b.userRepository.GetByUserID(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found: %s", req.UserID)
	}
	// Update the premium status
	user.PremiumType = 1 // Set to regular member
	user.PremiumExpiresAt = &res.ExpiryTime

	// Save the changes
	err = b.userRepository.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update user premium status: %w", err)
	}

	if subPackage.Coins > 0 {
		_, err = b.userCoinsRepository.UpdateBalance(ctx, req.UserID, req.SiteID, subPackage.Coins, 0)
		if err != nil {
			return fmt.Errorf("failed to update user balance: %w", err)
		}
	}

	return nil
}

func (b *paymentBiz) VerifyInAppPurchase(ctx context.Context, req *PaymentVerifyRequest) (*PaymentVerifyInAppPurchaseResponse, error) {
	fmt.Println("[VerifyInAppPurchase] called with req:", req)
	service, err := b.getIapService(ctx, req.Account)
	if err != nil {
		fmt.Println("[VerifyInAppPurchase] getIapService error:", err)
		return nil, err
	}

	fmt.Println("[VerifyInAppPurchase] calling service.VerifyInAppPurchase with args:", req.PackageName, req.PurchaseToken, req.TransactionID)
	res, err := service.VerifyInAppPurchase(ctx, &iap.IAPVerifyArgs{
		PackageName:   req.PackageName,
		PurchaseToken: req.PurchaseToken,
		TransactionID: req.TransactionID,
	})
	if err != nil {
		fmt.Println("[VerifyInAppPurchase] service.VerifyInAppPurchase error:", err)
		return nil, err
	}

	fmt.Println("[VerifyInAppPurchase] service.VerifyInAppPurchase result:", res)

	if res.ProductID == "" {
		fmt.Println("[VerifyInAppPurchase] invalid in-app purchase, ProductID empty")
		return nil, fmt.Errorf("invalid in-app purchase")
	} else {
		fmt.Println("[VerifyInAppPurchase] processCoinPackagePayment with ProductID:", res.ProductID)
		if err := b.processCoinPackagePayment(ctx, req, res); err != nil {
			fmt.Println("[VerifyInAppPurchase] processCoinPackagePayment error:", err)
			return nil, err
		}
	}

	fmt.Println("[VerifyInAppPurchase] returning ProductID:", res.ProductID)
	return &PaymentVerifyInAppPurchaseResponse{
		ProductID: res.ProductID,
	}, nil
}

func (b *paymentBiz) processCoinPackagePayment(ctx context.Context, req *PaymentVerifyRequest, res *iap.IAPVerifyInAppPurchaseRes) error {

	coinPackage, err := b.coinPackageRepo.GetByIOSProductID(ctx, res.ProductID)
	if err != nil {
		return fmt.Errorf("failed to get coin package: %w", err)
	}

	existing, err := b.paymentTransactionRepo.GetByTransactionID(ctx, req.TransactionID)
	if err != nil {
		return fmt.Errorf("failed to check payment transaction: %w", err)
	}
	if existing != nil {
		if existing.Status == model.PaymentStatusSuccess {
			return nil
		}
		existing.Status = model.PaymentStatusSuccess
		return b.addCoinsToUserAccount(ctx, existing)
	}

	snapshot := model.JSONMap{
		"package_id":          coinPackage.PackageID,
		"name":                coinPackage.Name,
		"description":         coinPackage.Description,
		"coin_amount":         coinPackage.CoinAmount,
		"price":               coinPackage.Price,
		"currency":            coinPackage.Currency,
		"original_price":      coinPackage.OriginalPrice,
		"discount_percentage": coinPackage.DiscountPercentage,
	}
	tx := &model.PaymentTransaction{
		TransactionID:     req.TransactionID,
		OrderID:           req.TransactionID,
		UserID:            req.UserID,
		SiteID:            req.SiteID,
		Amount:            coinPackage.Price,
		Currency:          coinPackage.Currency,
		Provider:          "ios",
		ProviderPaymentID: req.TransactionID,
		PaymentType:       model.PaymentTypeCoinPackage,
		Status:            model.PaymentStatusSuccess,
		RelatedID:         coinPackage.PackageID,
		RelatedType:       model.RelatedTypeCoinPackage,
		Snapshot:          snapshot,
	}

	err = b.paymentTransactionRepo.Create(ctx, tx)
	if err != nil {
		return err
	}

	err = b.addCoinsToUserAccount(ctx, tx)
	if err != nil {
		return err
	}
	return nil
}

// addCoinsToUserAccount adds coins to a user's account after a successful payment
func (b *paymentBiz) addCoinsToUserAccount(ctx context.Context, transaction *model.PaymentTransaction) error {
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
	_, err := b.coinsService.AddCoinsByPayment(ctx, coins, "ios", transaction)
	if err != nil {
		return err
	}
	return nil
}

func (b *paymentBiz) Notify(ctx context.Context, req *PaymentNotifyRequest) (*Empty, error) {
	service, err := b.getIapService(ctx, req.Account)
	if err != nil {
		return nil, err
	}

	res, err := service.Notify(ctx, &iap.IAPNotifyArgs{
		Body: req.Body,
	})
	if err != nil {
		//b.logger.Errorf("payment notify iap failed: %s", err)
		return nil, err
	}

	// // 如果是重复通知，直接返回
	// if b.isRepeatedMessage(ctx, res.MessageID) {
	// 	return nil, nil
	// }

	sub, err := b.userSubscriptionRepo.GetBySubscriptionID(ctx, res.OriginalOrderID)
	if err != nil && sub == nil {
		return nil, fmt.Errorf("failed to get user subscription: %w", err)
	}

	// Get the user to check if they already have premium status
	user, err := b.userRepository.GetByUserID(ctx, sub.UserID)
	if err != nil {
		return &Empty{}, nil
	}

	userSubscription := &model.UserSubscription{
		SubscriptionID:         res.OriginalOrderID,
		UserID:                 sub.UserID,
		SiteID:                 sub.SiteID,    // siteID can be set if needed
		PackageID:              sub.PackageID, // The ID of the subscription package in your system
		Provider:               sub.Provider,
		ProviderSubscriptionID: res.OrderID,                  // IMPORTANT: This is currently the Stripe Checkout Session ID from the transaction.
		Status:                 model.SubscriptionStatusNone, // None
		CurrentPeriodStart:     res.StartTime,                // The start of the current billing cycle
		CurrentPeriodEnd:       res.ExpiryTime,               // The end of the current billing cycle
	}

	switch res.SubStatus {
	case iap.SubStatusReNew:
		// 续订成功, 修改订阅状态
		userSubscription.Status = model.SubscriptionStatusActive
	case iap.SubStatusRefund:
		// 退款，取消订阅权益
		userSubscription.Status = model.SubscriptionStatusCancelled
	case iap.SubStatusExpired:
		// 订阅失效
		userSubscription.Status = model.SubscriptionStatusExpired
	case iap.SubStatusTest:
		// 非正式订阅消息，直接返回
		return &Empty{}, nil
	case iap.SubStatusChangeRenewalStatus:
		// 修改自动续订状态
	default:
		// 其他事件记录原始通知类型
		return &Empty{}, err
	}

	if userSubscription.Status != model.SubscriptionStatusNone {
		// Subscription already exists, no need to process again
		if err := b.userSubscriptionRepo.Update(ctx, userSubscription); err != nil {
			// This error will propagate up and cause the database transaction in handleCheckoutSessionCompleted to roll back.
			return nil, fmt.Errorf("failed to save user subscription details: %w", err)
		}

		// Update the premium status
		user.PremiumType = 1 // Set to regular member
		user.PremiumExpiresAt = &res.ExpiryTime

		// Save the changes
		err = b.userRepository.Update(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("failed to update user premium status: %w", err)
		}
	}

	return &Empty{}, nil
}

// 根据accname获取平台服务验证
func (b *paymentBiz) getIapService(ctx context.Context, account string) (svc iap.Service, err error) {

	// 通过account查询对应的配置信息
	option := map[string]string{
		"key_id": "M7AP3ZSCFL",
		"key_content": `-----BEGIN PRIVATE KEY-----
MIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQgyWLJ9cqM041Opvso
UWYm7uyzgxUXR66adWnHGKuuI12gCgYIKoZIzj0DAQehRANCAASfWPOIImjOV9O1
HEfXM0sN6Vm6Jx9jS3V1lU8GKvmMdYJZIleWGnzoFAHf9a1dDQEGYeOWsoiP1S+2
MbZ13+R6
-----END PRIVATE KEY-----`,
		"bundle_id": "super.smart.photo.cleaner.pro",
		"issuer":    "4beb7e54-4675-4c65-89ec-081940277d7e",
	}
	svc, err = iap.GetService("apple", option)
	if err != nil {
		return nil, err
	}

	return svc, err
}
