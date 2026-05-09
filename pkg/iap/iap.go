package iap

import (
	"context"
	"fmt"
	"time"
)

const (
	ChannelApple  = "apple"
	ChannelGoogle = "google"
)

type IAPVerifyArgs struct {
	PackageName   string
	PurchaseToken string
	TransactionID string
}

type IAPVerifySubscriptionRes struct {
	IsActive        bool
	OrderID         string
	OriginalOrderID string
	StartTime       time.Time
	ExpiryTime      time.Time
	Sandbox         bool
	ProductID       string
	IsInFreeTrial   bool
	AutoRenewStatus bool
	Price           int32
	Currency        string
}

type IAPVerifyInAppPurchaseRes struct {
	ProductID string
}

type IAPNotifyArgs struct {
	Body []byte
}

type IAPNotifyRes struct {
	MessageID       string    // 消息ID
	SubStatus       int32     // 0-续订 1-取消
	OrderID         string    // 订单ID
	OriginalOrderID string    // 原始订单ID
	StartTime       time.Time // 订阅周期起始时间
	ExpiryTime      time.Time // 订阅周期结束时间
	ProductID       string    // 订阅套餐
	EventType       string    // 事件类型
	Sandbox         bool      // 是否为沙盒环境
	Price           int32     // 金额
	Currency        string    // 货币
	AutoRenew       bool      // 是否自动续订(仅收到 SubStatusChangeRenewalStatus 时有效)
}

const (
	SubStatusNone                int32 = 0 // 不处理
	SubStatusReNew               int32 = 1 // 续订
	SubStatusExpired             int32 = 2 // 订阅到期未续订(续订失败)
	SubStatusRefund              int32 = 3 // 退款（取消当前订阅权益）
	SubStatusTest                int32 = 4 // 测试
	SubStatusChangeRenewalStatus int32 = 5 // 修改自动续订状态
)

type Service interface {
	VerifyInAppPurchase(ctx context.Context, args *IAPVerifyArgs) (*IAPVerifyInAppPurchaseRes, error)
	VerifySubscription(ctx context.Context, args *IAPVerifyArgs) (*IAPVerifySubscriptionRes, error)
	Notify(ctx context.Context, args *IAPNotifyArgs) (*IAPNotifyRes, error)
}

type InfoGetter func(accName string) (channel string, opts map[string]string, err error)

func GetService(channel string, opts map[string]string) (svc Service, err error) {
	switch channel {
	case ChannelApple:
		svc = newAppleService(opts)
	case ChannelGoogle:
		svc, err = newGoogleService(opts)
	default:
		return nil, fmt.Errorf("unknown pay channel: %s", channel)
	}
	if err != nil {
		return nil, err
	}
	return svc, err
}
