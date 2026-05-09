package iap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/appstore/api"
)

const (
	OptionKeyID      = "key_id"
	OptionKeyContent = "key_content"
	OptionBundleID   = "bundle_id"
	OptionIssuer     = "issuer"
)

type AppleService struct {
	apiClient        *api.StoreClient
	apiClientSandbox *api.StoreClient
	appstoreClient   *appstore.Client
}

type TransactionOtherFields struct {
	Price             int32  `json:"price"`
	Currency          string `json:"currency"`
	OfferType         int32  `json:"offerType"`
	OfferDiscountType string `json:"offerDiscountType"`
}

func newAppleService(opts map[string]string) *AppleService {
	cfg := &api.StoreConfig{
		KeyContent: []byte(opts[OptionKeyContent]),
		KeyID:      opts[OptionKeyID],
		BundleID:   opts[OptionBundleID],
		Issuer:     opts[OptionIssuer],
		Sandbox:    false,
	}
	apiClient := api.NewStoreClient(cfg)

	cfg.Sandbox = true
	apiClientSandbox := api.NewStoreClient(cfg)

	return &AppleService{
		apiClient:        apiClient,
		apiClientSandbox: apiClientSandbox,
		appstoreClient:   appstore.New(),
	}
}

func (a *AppleService) VerifySubscription(ctx context.Context, args *IAPVerifyArgs) (*IAPVerifySubscriptionRes, error) {
	var e *api.Error
	// GetALLSubscriptionStatuses 不管传订阅还是续订的订单ID都会返回对应的订阅续订列表的最新一个订阅的订单信息
	rsp, err := a.apiClient.GetALLSubscriptionStatuses(ctx, args.TransactionID, nil)

	if (err != nil && strings.Contains(err.Error(), "status code 401")) || (errors.As(err, &e) && e.ErrorCode() == 4040010) {
		rsp, err = a.apiClientSandbox.GetALLSubscriptionStatuses(ctx, args.TransactionID, nil)
	}

	if err != nil {
		return nil, err
	}

	var LastTransactionsItem api.LastTransactionsItem
	if len(rsp.Data) > 0 && len(rsp.Data[0].LastTransactions) > 0 {
		LastTransactionsItem = rsp.Data[0].LastTransactions[0]
	} else {
		//a.logger.Warnf("AppleService.Verify: No Last Transactions Item, transactionID: %s", args.TransactionID)
		return &IAPVerifySubscriptionRes{
			IsActive: false,
		}, nil
	}

	// 解析订单信息
	transaction, err := a.apiClient.ParseSignedTransaction(LastTransactionsItem.SignedTransactionInfo)
	if err != nil {
		return nil, err
	}
	if transaction == nil {
		return nil, errors.New("invalid transaction")
	}

	var price int32
	var currency string
	var isInFreeTrial bool
	isActive := LastTransactionsItem.Status == api.SubscriptionActive
	// 获取 TransactionInfo 的SDK未支持的其他字段
	parts := strings.Split(LastTransactionsItem.SignedTransactionInfo, ".")
	if len(parts) == 3 {
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, err
		}

		var data TransactionOtherFields
		err2 := json.Unmarshal(payload, &data)
		if err2 != nil {
			//a.logger.Infof("apple failed to unmarshal transaction other fields: %s", err2)
		} else {
			price = data.Price
			currency = data.Currency
			if isActive && data.OfferType == int32(appstore.IntroductoryOffer) && data.OfferDiscountType == "FREE_TRIAL" {
				isInFreeTrial = true
			}
		}

	}

	var autoRenewStatus bool
	// 解析 RenewalInfo 获取自动续订状态
	if len(strings.Split(LastTransactionsItem.SignedRenewalInfo, ".")) == 3 {
		renewalInfo, err2 := a.apiClient.ParseJWSEncodeString(LastTransactionsItem.SignedRenewalInfo)
		if err2 != nil {
			//a.logger.Infof("failed to parse jws renewal info: %s", err)
		} else {
			if info, ok := renewalInfo.(*api.JWSRenewalInfoDecodedPayload); ok {
				if info != nil && info.AutoRenewStatus == api.AutoRenewStatusOn {
					autoRenewStatus = true
				}
			}
		}
	}

	return &IAPVerifySubscriptionRes{
		IsActive:        isActive,
		OrderID:         transaction.TransactionID,
		OriginalOrderID: transaction.OriginalTransactionId,
		StartTime:       time.UnixMilli(transaction.PurchaseDate),
		ExpiryTime:      time.UnixMilli(transaction.ExpiresDate),
		Sandbox:         transaction.Environment == api.Sandbox,
		ProductID:       transaction.ProductID,
		IsInFreeTrial:   isInFreeTrial,
		AutoRenewStatus: autoRenewStatus,
		Price:           price,
		Currency:        currency,
	}, nil
}

func (a *AppleService) VerifyInAppPurchase(ctx context.Context, args *IAPVerifyArgs) (*IAPVerifyInAppPurchaseRes, error) {

	rsp, err := a.apiClient.GetTransactionInfo(ctx, args.TransactionID)

	var e *api.Error
	if (err != nil && strings.Contains(err.Error(), "status code 401")) || (errors.As(err, &e) && e.ErrorCode() == 4040010) {
		rsp, err = a.apiClientSandbox.GetTransactionInfo(ctx, args.TransactionID)
	}

	if err != nil {
		return nil, err
	}

	// 解析订单信息
	transaction, err := a.apiClient.ParseSignedTransaction(rsp.SignedTransactionInfo)
	if err != nil {
		return nil, err
	}
	if transaction == nil {
		return nil, errors.New("invalid transaction")
	}

	return &IAPVerifyInAppPurchaseRes{
		ProductID: transaction.ProductID,
	}, nil
}

func (a *AppleService) Notify(_ context.Context, args *IAPNotifyArgs) (*IAPNotifyRes, error) {
	var body appstore.SubscriptionNotificationV2SignedPayload
	if err := json.Unmarshal(args.Body, &body); err != nil {
		return nil, err
	}

	np := appstore.SubscriptionNotificationV2DecodedPayload{}
	if err := a.appstoreClient.ParseNotificationV2WithClaim(body.SignedPayload, &np); err != nil {
		return nil, err
	}
	tp := appstore.JWSTransactionDecodedPayload{}
	if err := a.appstoreClient.ParseNotificationV2WithClaim(string(np.Data.SignedTransactionInfo), &tp); err != nil {
		return nil, err
	}

	var subStatus int32
	var autoRenew bool
	switch np.NotificationType {
	case appstore.NotificationTypeV2DidRenew:
		subStatus = SubStatusReNew // 成功续订
	case appstore.NotificationTypeV2DidFailToRenew, appstore.NotificationTypeV2Expired:
		subStatus = SubStatusExpired // 续订失败 订阅到期
	case appstore.NotificationTypeV2Refund:
		subStatus = SubStatusRefund // 用户退款
	case appstore.NotificationTypeV2DidChangeRenewalStatus:
		subStatus = SubStatusChangeRenewalStatus // 修改自动续订状态
		if np.Subtype == appstore.SubTypeV2AutoRenewEnabled {
			autoRenew = true
		}
	default:
		subStatus = SubStatusNone // 不处理
	}

	var price int32
	var currency string
	if subStatus == SubStatusReNew {
		// 获取 TransactionInfo 的SDK未支持的其他字段
		parts := strings.Split(string(np.Data.SignedTransactionInfo), ".")
		if len(parts) == 3 {
			payload, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err != nil {
				return nil, err
			}

			var data TransactionOtherFields
			err2 := json.Unmarshal(payload, &data)
			if err2 != nil {
				//a.logger.Infof("apple notify failed to unmarshal transaction other fields: %s", err2)
			} else {
				price = data.Price
				currency = data.Currency
			}
		}
	}

	//a.logger.Infof("apple notify type: %s,res:  %+v", np.NotificationType, tp)

	return &IAPNotifyRes{
		SubStatus:       subStatus,
		EventType:       string(np.NotificationType),
		MessageID:       np.NotificationUUID,
		OrderID:         tp.TransactionId,
		OriginalOrderID: tp.OriginalTransactionId,
		ProductID:       tp.ProductId,
		StartTime:       time.UnixMilli(tp.PurchaseDate),
		ExpiryTime:      time.UnixMilli(tp.ExpiresDate),
		Sandbox:         tp.Environment == appstore.Sandbox,
		Price:           price,
		Currency:        currency,
		AutoRenew:       autoRenew,
	}, nil

}
