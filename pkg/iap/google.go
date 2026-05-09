package iap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/awa/go-iap/playstore"
	"github.com/spf13/cast"
	"google.golang.org/api/androidpublisher/v3"
)

const (
	OptionKeyServiceAccountJson = "service_account_json"
)

type GoogleService struct {
	client      *playstore.Client
}

func newGoogleService(opts map[string]string) (*GoogleService, error) {
	client, err := playstore.New([]byte(opts[OptionKeyServiceAccountJson]))
	if err != nil {
		return nil, err
	}
	return &GoogleService{
		client:      client,
	}, nil
}

func (g *GoogleService) VerifySubscription(ctx context.Context, args *IAPVerifyArgs) (*IAPVerifySubscriptionRes, error) {
	// https://developers.google.com/android-publisher/api-ref/rest/v3/purchases.subscriptionsv2
	sp, err := g.client.VerifySubscriptionV2(ctx, args.PackageName, args.PurchaseToken)
	if err != nil {
		return nil, err
	}
	if sp.SubscriptionState != "SUBSCRIPTION_STATE_ACTIVE" && sp.SubscriptionState != "SUBSCRIPTION_STATE_CANCELED" {
		//g.logger.Infof("google subscription purchases: %+v", sp)
		return nil, fmt.Errorf("invalid subscription state")
	}
	if len(sp.LineItems) <= 0 {
		//g.logger.Infof("google subscription purchases: %+v", sp)
		return nil, fmt.Errorf("invalid purchase state")
	}

	st, et := g.getSubTime(sp)
	res := &IAPVerifySubscriptionRes{
		IsActive:   et.After(time.Now()),
		OrderID:    sp.LatestOrderId,
		StartTime:  st,
		ExpiryTime: et,
		Sandbox:    sp.TestPurchase != nil,
		ProductID:  sp.LineItems[0].ProductId,
	}
	if strings.Contains(res.OrderID, "..") {
		res.OriginalOrderID = strings.Split(res.OrderID, "..")[0]
	} else {
		res.OriginalOrderID = res.OrderID
	}
	return res, nil
}

func (g *GoogleService) VerifyInAppPurchase(ctx context.Context, args *IAPVerifyArgs) (*IAPVerifyInAppPurchaseRes, error) {
	//TODO implement me
	return nil, fmt.Errorf("not implemented")
}

// GooglePub 谷歌开发者实时通知请求体
type GooglePub struct {
	Message struct {
		Attributes struct {
			Key string `json:"key"`
		} `json:"attributes"`
		Data      string `json:"data"`
		MessageId string `json:"messageId"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

type SubscriptionNotification struct {
	Version          string `json:"version"`
	NotificationType int    `json:"notificationType"`
	PurchaseToken    string `json:"purchaseToken"`
	SubscriptionId   string `json:"subscriptionId"`
}

type OneTimeProductNotification struct {
	Version          string `json:"version"`
	NotificationType int    `json:"notificationType"`
	PurchaseToken    string `json:"purchaseToken"`
	Sku              string `json:"sku"`
}

type TestNotification struct {
	Version interface{} `json:"version"`
}

type DeveloperNotification struct {
	Version                    string                      `json:"version"`
	PackageName                string                      `json:"packageName"`
	EventTimeMillis            string                      `json:"eventTimeMillis"`
	OneTimeProductNotification *OneTimeProductNotification `json:"oneTimeProductNotification"`
	SubscriptionNotification   *SubscriptionNotification   `json:"subscriptionNotification"`
	TestNotification           *TestNotification           `json:"testNotification"`
}

// Notify 处理用户后续订阅状态变更
func (g *GoogleService) Notify(ctx context.Context, args *IAPNotifyArgs) (*IAPNotifyRes, error) {
	res := &IAPNotifyRes{}
	var gp GooglePub
	err := json.Unmarshal(args.Body, &gp)
	if err != nil {
		//g.logger.Errorf("unmarshal google body failed: %s", err)
		return nil, err
	}

	//g.logger.Infof("receive google pub: %+v", gp)

	baseData := gp.Message.Data
	decoded, err := base64.StdEncoding.DecodeString(baseData)
	if err != nil {
		//g.logger.Errorf("decoding base64 str error: %s", err)
		return res, nil
	}

	// 将JSON字节解析为结构体
	developerNotification := DeveloperNotification{}
	err = json.Unmarshal(decoded, &developerNotification)
	if err != nil {
		//g.logger.Errorf("unmarshal developer notification error: %s", err)
		return res, nil
	}

	if developerNotification.SubscriptionNotification == nil {
		//g.logger.Infof("notification is not subscription")
		res.SubStatus = SubStatusTest
		return res, nil
	}

	packageName := developerNotification.PackageName
	subNotification := developerNotification.SubscriptionNotification
	// 订阅id
	subscriptionID := subNotification.SubscriptionId
	// 购买订阅时向用户设备提供的令牌
	purchaseToken := subNotification.PurchaseToken
	// 向google获取订单状态
	sp, err := g.client.VerifySubscriptionV2(ctx, packageName, purchaseToken)
	if err != nil {
		//g.logger.Error("failed to get subscription detail from google", err)
		return res, err
	}

	switch playstore.SubscriptionNotificationType(subNotification.NotificationType) {
	case playstore.SubscriptionNotificationTypeRenewed:
		// 续订
		if sp.AcknowledgementState != "ACKNOWLEDGEMENT_STATE_ACKNOWLEDGED" {
			return res, fmt.Errorf("invalid acknowledgement state")
		}
		if len(sp.LineItems) <= 0 {
			return res, fmt.Errorf("invalid purchase state")
		}
		res.SubStatus = SubStatusReNew
		//g.logger.Infof("google notify received, status: renew")
	case playstore.SubscriptionNotificationTypeExpired:
		// 订阅到期
		res.SubStatus = SubStatusExpired
		//g.logger.Infof("google notify received, status: expired")
	case playstore.SubscriptionNotificationTypeRevoked:
		// 用户退款
		res.SubStatus = SubStatusRefund
		//g.logger.Infof("google notify received, status: refund")
	default:
		res.SubStatus = SubStatusNone
	}

	st, et := g.getSubTime(sp)
	res.StartTime = st
	res.ExpiryTime = et
	res.EventType = cast.ToString(subNotification.NotificationType)
	res.MessageID = gp.Message.MessageId
	res.ProductID = subscriptionID
	res.OrderID = sp.LatestOrderId
	if strings.Contains(res.OrderID, "..") {
		res.OriginalOrderID = strings.Split(res.OrderID, "..")[0]
	} else {
		res.OriginalOrderID = res.OrderID
	}

	//res.Price=

	//g.logger.Infof("google IAPNotifyRes: %+v", res)
	return res, nil
}

func (g *GoogleService) getSubTime(sp *androidpublisher.SubscriptionPurchaseV2) (startTime, expiryTime time.Time) {
	var err error
	purchaseItem := sp.LineItems[0]
	// 此处将订阅的起始时间赋值为第一次付款时间
	startTime, err = time.Parse(time.RFC3339, sp.StartTime)
	if err != nil {
		//g.logger.Errorf("parse google's startTime error: %s", err)
	}
	expiryTime, err = time.Parse(time.RFC3339, purchaseItem.ExpiryTime)
	if err != nil {
		//g.logger.Errorf("parse google's expiryTime error: %s", err)
	}
	return
}