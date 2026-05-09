package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"shortpress-server/internal/model"
	"time"

	"net/http"

	"github.com/spf13/viper"
)

// FacebookClient Facebook Conversions API 客户端
type FacebookClient struct {
	accessToken string
	datasetID   string
	apiVersion  string
	client      *http.Client
}

// ShenshuClient 数数（ThinkingData）客户端
type ShenshuClient struct {
	appID  string
	url    string
	client *http.Client
}

// TrackingService 统一的打点服务
type TrackingService struct {
	facebookClient *FacebookClient
	shenshuClient  *ShenshuClient
}

// NewTrackingService 创建打点服务
// 从 viper 配置中读取打点配置
func NewTrackingService(conf *viper.Viper) *TrackingService {
	// 从配置中读取 Facebook 和数数的配置
	fbDatasetID := conf.GetString("analytics.facebook.datasetId")
	fbAccessToken := conf.GetString("analytics.facebook.accessToken")
	fbApiVersion := conf.GetString("analytics.facebook.apiVersion")
	shenshuAppID := conf.GetString("analytics.shenshu.appId")

	// 如果配置为空，使用默认值
	if fbDatasetID == "" {
		fbDatasetID = "905540855186929"
	}
	if fbAccessToken == "" {
		fbAccessToken = "EAAH8migLdZAABQ6nm1dY43i0TNj2GQR3igsMU8F45MjWqrtXqwfa0VLYoSDYAMZAnpkBrvZA46D62UyiEgZAQYpzj4Q4XGs8pYdfqVwsUzIWEWSkK44lp8smxgvhOhBSwNkYzOvz9cpmTagokkHzNPwgdrIh0TayJ4ZC0qdiltdUT0k0y8ntZAnqgeNhX9vkdrhwZDZD"
	}
	if fbApiVersion == "" {
		fbApiVersion = "v18.0"
	}
	if shenshuAppID == "" {
		shenshuAppID = "f1b5c41b749442e889414f227eb17a75"
	}

	var fbClient *FacebookClient
	if fbAccessToken != "" && fbDatasetID != "" {
		fbClient = &FacebookClient{
			accessToken: fbAccessToken,
			datasetID:   fbDatasetID,
			apiVersion:  fbApiVersion,
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	}

	var ssClient *ShenshuClient
	if shenshuAppID != "" {
		ssClient = &ShenshuClient{
			appID: shenshuAppID,
			url:   "https://ta-receiver.mojoly.net/sync_json",
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	}

	return &TrackingService{
		facebookClient: fbClient,
		shenshuClient:  ssClient,
	}
}

// TrackPurchase 跟踪购买事件（发送到所有配置的平台）
func (s *TrackingService) TrackPurchase(ctx context.Context, transaction *model.PaymentTransaction) error {
	// 发送到 Facebook
	// if s.facebookClient != nil {
	// 	if err := s.facebookClient.TrackPurchase(ctx, transaction); err != nil {
	// 		fmt.Printf("Facebook Purchase 事件打点失败: %v\n", err)
	// 	} else {
	// 		fmt.Printf("Facebook Purchase 事件发送成功: %s\n", transaction.TransactionID)
	// 	}
	// }

	// 发送到数数
	if s.shenshuClient != nil {
		if err := s.shenshuClient.TrackPurchase(ctx, transaction); err != nil {
			fmt.Printf("数数 Purchase 事件打点失败: %v\n", err)
		} else {
			fmt.Printf("数数 Purchase 事件发送成功: %s\n", transaction.TransactionID)
		}
	}

	return nil
}

// ==================== Facebook Conversions API ====================

// TrackPurchase 发送 Purchase 事件到 Facebook
func (fb *FacebookClient) TrackPurchase(ctx context.Context, transaction *model.PaymentTransaction) error {
	if fb == nil {
		return fmt.Errorf("Facebook client not initialized")
	}

	// 构建 custom_data
	customData := map[string]interface{}{
		"currency": transaction.Currency,
		"value":    float64(transaction.Amount) / 100, // 数值类型，符合 Facebook 要求
		"order_id": transaction.TransactionID,
	}

	// content_ids / content_type
	if transaction.RelatedID != "" {
		customData["content_ids"] = []string{transaction.RelatedID}
		customData["content_type"] = "product"
	}

	// 从快照中提取更多信息
	if transaction.Snapshot != nil {
		if name, ok := transaction.Snapshot["name"].(string); ok {
			customData["content_name"] = name
		}
		if coinAmount, ok := transaction.Snapshot["coin_amount"].(float64); ok {
			customData["coin_amount"] = int(coinAmount)
		}
	}

	// user_data，使用 user_id 作为 external_id，提升匹配率
	userData := map[string]interface{}{
		"external_id": transaction.UserID,
	}

	// 构建事件数据，按照标准 Purchase 模板
	event := map[string]interface{}{
		"event_name":    "Purchase",
		"event_time":    transaction.CreatedAt.Unix(),
		"action_source": "website",
		"user_data":     userData,
		"custom_data":   customData,
	}

	// 构建请求数据
	requestBody := map[string]interface{}{
		"data": []interface{}{event},
	}

	// 序列化为 JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/events", fb.apiVersion, fb.datasetID)

	// 发送 HTTP POST 请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 添加 access token 到 URL
	q := req.URL.Query()
	q.Add("access_token", fb.accessToken)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")

	resp, err := fb.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Facebook API 返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}

// ==================== 数数 ====================

// Close 关闭数数客户端
func (ss *ShenshuClient) Close() error {
	// HTTP 客户端不需要关闭
	return nil
}

// TrackPurchase 发送 Purchase 事件到数数
func (ss *ShenshuClient) TrackPurchase(ctx context.Context, transaction *model.PaymentTransaction) error {
	if ss == nil {
		return fmt.Errorf("Shenshu client not initialized")
	}

	// 优先使用交易的创建时间；如果缺失则回退到当前服务器时间
	eventTime := transaction.CreatedAt
	if eventTime.IsZero() {
		eventTime = time.Now()
	}
	// 正常情况下按 America/Los_Angeles 时区展示；如果时区加载失败，则回退为手动减 16 小时
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err == nil {
		eventTime = eventTime.In(loc)
	} else {
		eventTime = eventTime.Add(-16 * time.Hour)
	}

	// 数数要求 #time 为 "yyyy-MM-dd HH:mm:ss.SSS" 或 "yyyy-MM-dd HH:mm:ss" 格式的字符串
	const shenshuTimeLayout = "2006-01-02 15:04:05.000"

	// 构建事件数据（使用新的 API 格式）
	eventData := map[string]interface{}{
		"#type":       "track",
		"#event_name": "purchase",
		"#time":       eventTime.Format(shenshuTimeLayout),
		"#account_id": transaction.UserID,
		"properties": map[string]interface{}{
			"order_id":       transaction.TransactionID,
			"product_id":     transaction.RelatedID,
			"payment_method": transaction.Provider,
			"currency":       transaction.Currency,
			"amount":         float64(transaction.Amount) / 100,
			"site_id":        transaction.SiteID,
		},
	}

	// 从快照中提取更多信息
	if transaction.Snapshot != nil {
		if name, ok := transaction.Snapshot["name"].(string); ok {
			eventData["properties"].(map[string]interface{})["product_name"] = name
		}
		if coinAmount, ok := transaction.Snapshot["coin_amount"].(float64); ok {
			eventData["properties"].(map[string]interface{})["coin_amount"] = int(coinAmount)
		}
		if price, ok := transaction.Snapshot["price"].(float64); ok {
			eventData["properties"].(map[string]interface{})["revenue"] = price / 100
			eventData["properties"].(map[string]interface{})["revenue_actual"] = price / 100
		}
	}

	// 构建请求数据
	requestBody := map[string]interface{}{
		"appid": ss.appID,
		"data":  eventData,
	}

	// 序列化为 JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %w", err)
	}

	// 发送 HTTP POST 请求
	req, err := http.NewRequestWithContext(ctx, "POST", ss.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ss.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("数数API返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}
