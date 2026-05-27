package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db/site"
	"shortpress-server/internal/repository/db/user"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// FacebookClient Facebook Conversions API 客户端（按请求使用站点 token / pixel）
type FacebookClient struct {
	apiVersion string
	client     *http.Client
}

// ShenshuClient 数数（ThinkingData）客户端
type ShenshuClient struct {
	appID  string
	url    string
	client *http.Client
}

// TrackingService 统一的打点服务
type TrackingService struct {
	siteRepository site.SiteRepository
	userRepository user.UserRepository
	shenshuClient  *ShenshuClient
	apiVersion     string
	testEventCode  string
}

// NewTrackingService 创建打点服务
func NewTrackingService(
	conf *viper.Viper,
	siteRepository site.SiteRepository,
	userRepository user.UserRepository,
) *TrackingService {
	fbApiVersion := conf.GetString("analytics.facebook.apiVersion")
	if fbApiVersion == "" {
		fbApiVersion = "v21.0"
	}
	shenshuAppID := conf.GetString("analytics.shenshu.appId")
	if shenshuAppID == "" {
		shenshuAppID = "f1b5c41b749442e889414f227eb17a75"
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
		siteRepository: siteRepository,
		userRepository: userRepository,
		shenshuClient:  ssClient,
		apiVersion:     fbApiVersion,
		testEventCode:  conf.GetString("analytics.facebook.testEventCode"),
	}
}

// TrackPurchase 跟踪购买事件（Facebook CAPI 按站点 + 数数）
func (s *TrackingService) TrackPurchase(ctx context.Context, transaction *model.PaymentTransaction) error {
	if transaction == nil {
		return nil
	}

	meta, u := s.resolvePurchaseAttribution(ctx, transaction)

	if err := s.trackFacebookPurchase(ctx, transaction, meta, u); err != nil {
		fmt.Printf("Facebook Purchase CAPI failed: %v\n", err)
	}

	if s.shenshuClient != nil {
		if err := s.shenshuClient.TrackPurchase(ctx, transaction, meta); err != nil {
			fmt.Printf("数数 Purchase 事件打点失败: %v\n", err)
			return err
		}
		fmt.Printf("数数 Purchase 事件发送成功: %s (fbc=%v fbp=%v)\n",
			transaction.TransactionID, meta.Fbc != "", meta.Fbp != "")
	}

	return nil
}

func addTrackingContextProperties(properties map[string]interface{}, snapshot model.JSONMap) {
	if snapshot == nil {
		return
	}

	rawContext, ok := snapshot["tracking_context"]
	if !ok {
		return
	}

	context, ok := rawContext.(map[string]interface{})
	if !ok {
		if jsonBytes, err := json.Marshal(rawContext); err == nil {
			_ = json.Unmarshal(jsonBytes, &context)
		}
	}

	if context == nil {
		return
	}

	allowedKeys := map[string]struct{}{
		"utm_source0":     {},
		"page0":           {},
		"utm_source":      {},
		"utm_medium":      {},
		"utm_campaign":    {},
		"utm_content":     {},
		"utm_term":        {},
		"fbclid":          {},
		"page_url":        {},
		"referrer":        {},
		"page_title":      {},
		"visitor_id":      {},
		"session_id":      {},
		"user_agent":      {},
		"language":        {},
		"platform":        {},
		"screen_height":   {},
		"screen_width":    {},
		"timezone_offset": {},
		"video_id":        {},
		"playlist_id":     {},
		"video_name":      {},
		"playlist_name":   {},
		"pay_type":        {},
		"page":            {},
	}

	for key, value := range context {
		if _, ok := allowedKeys[key]; !ok {
			continue
		}
		if _, exists := properties[key]; exists {
			continue
		}

		switch value.(type) {
		case string, bool, float64, float32, int, int64, int32, uint, uint64, uint32:
			properties[key] = value
		}
	}
}

func (s *TrackingService) resolvePurchaseAttribution(ctx context.Context, transaction *model.PaymentTransaction) (ResolvedMetaClick, *model.User) {
	var u *model.User
	if s.userRepository != nil && transaction.UserID != "" {
		u, _ = s.userRepository.GetByUserID(ctx, transaction.UserID)
	}
	return ResolveMetaClick(transaction.Snapshot, u), u
}

func (s *TrackingService) trackFacebookPurchase(ctx context.Context, transaction *model.PaymentTransaction, meta ResolvedMetaClick, u *model.User) error {
	if s.siteRepository == nil {
		return fmt.Errorf("site repository not configured")
	}
	siteRow, err := s.siteRepository.GetBySiteID(ctx, transaction.SiteID)
	if err != nil {
		return err
	}
	if siteRow == nil {
		return fmt.Errorf("site not found: %s", transaction.SiteID)
	}
	pixelID := ""
	if siteRow.FacebookPixelID != nil {
		pixelID = strings.TrimSpace(*siteRow.FacebookPixelID)
	}
	accessToken := ""
	if siteRow.FacebookCapiAccessToken != nil {
		accessToken = strings.TrimSpace(*siteRow.FacebookCapiAccessToken)
	}
	if pixelID == "" || accessToken == "" {
		return nil
	}

	return sendFacebookPurchase(ctx, &FacebookClient{
		apiVersion: s.apiVersion,
		client:     &http.Client{Timeout: 10 * time.Second},
	}, pixelID, accessToken, s.testEventCode, transaction, meta, u)
}

func sendFacebookPurchase(
	ctx context.Context,
	fb *FacebookClient,
	datasetID, accessToken, testEventCode string,
	transaction *model.PaymentTransaction,
	meta ResolvedMetaClick,
	u *model.User,
) error {
	customData := map[string]interface{}{
		"currency": transaction.Currency,
		"value":    float64(transaction.Amount) / 100,
		"order_id": transaction.TransactionID,
	}
	if transaction.RelatedID != "" {
		customData["content_ids"] = []string{transaction.RelatedID}
		customData["content_type"] = "product"
	}
	if transaction.Snapshot != nil {
		if name, ok := transaction.Snapshot["name"].(string); ok {
			customData["content_name"] = name
		}
	}

	userData := map[string]interface{}{}
	if transaction.UserID != "" {
		userData["external_id"] = []string{HashMetaPII(transaction.UserID)}
	}
	if u != nil && strings.TrimSpace(u.Email) != "" {
		if hashed := HashMetaPII(u.Email); hashed != "" {
			userData["em"] = []string{hashed}
		}
	}
	if meta.Fbc != "" {
		userData["fbc"] = meta.Fbc
	}
	if meta.Fbp != "" {
		userData["fbp"] = meta.Fbp
	}

	eventTime := transaction.CreatedAt.Unix()
	if eventTime <= 0 {
		eventTime = time.Now().Unix()
	}

	event := map[string]interface{}{
		"event_name":    "Purchase",
		"event_time":    eventTime,
		"event_id":      fmt.Sprintf("purchase_%s", transaction.TransactionID),
		"action_source": "website",
		"user_data":     userData,
		"custom_data":   customData,
	}
	if meta.EventSourceURL != "" {
		event["event_source_url"] = meta.EventSourceURL
	}

	requestBody := map[string]interface{}{
		"data": []interface{}{event},
	}
	if testEventCode != "" {
		requestBody["test_event_code"] = testEventCode
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %w", err)
	}

	// 本地/测试：token 以 FAKE_ 开头时不请求 Meta，仅打印完整 CAPI 载荷
	if strings.HasPrefix(accessToken, "FAKE_") {
		pretty, _ := json.MarshalIndent(requestBody, "", "  ")
		fmt.Printf("[Meta CAPI dry-run] pixel=%s url=https://graph.facebook.com/%s/%s/events\n%s\n",
			datasetID, fb.apiVersion, datasetID, string(pretty))
		return nil
	}

	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/events", fb.apiVersion, datasetID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	q := req.URL.Query()
	q.Add("access_token", accessToken)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")

	resp, err := fb.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Facebook API 返回 %d: %s", resp.StatusCode, string(body))
	}
	fmt.Printf("Facebook Purchase CAPI success: %s\n", transaction.TransactionID)
	return nil
}

// Close 关闭数数客户端
func (ss *ShenshuClient) Close() error {
	return nil
}

// TrackPurchase 发送 Purchase 事件到数数（含 Meta 归因参数，与 CAPI 同源解析）
func (ss *ShenshuClient) TrackPurchase(ctx context.Context, transaction *model.PaymentTransaction, meta ResolvedMetaClick) error {
	if ss == nil {
		return fmt.Errorf("Shenshu client not initialized")
	}

	eventTime := transaction.CreatedAt
	if eventTime.IsZero() {
		eventTime = time.Now()
	}
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err == nil {
		eventTime = eventTime.In(loc)
	} else {
		eventTime = eventTime.Add(-16 * time.Hour)
	}

	const shenshuTimeLayout = "2006-01-02 15:04:05.000"

	props := map[string]interface{}{
		"order_id":       transaction.TransactionID,
		"transaction_id": transaction.TransactionID,
		"product_id":     transaction.RelatedID,
		"payment_method": transaction.Provider,
		"provider":       transaction.Provider,
		"currency":       transaction.Currency,
		"amount":         float64(transaction.Amount) / 100,
		"value":          float64(transaction.Amount) / 100,
		"site_id":        transaction.SiteID,
	}
	ApplyMetaAttributionProperties(props, meta, transaction.Snapshot)

	eventData := map[string]interface{}{
		"#type":       "track",
		"#event_name": "purchase",
		"#time":       eventTime.Format(shenshuTimeLayout),
		"#account_id": transaction.UserID,
		"properties":  props,
	}

	if transaction.Snapshot != nil {
		if name, ok := transaction.Snapshot["name"].(string); ok {
			props["product_name"] = name
		}
		if coinAmount, ok := transaction.Snapshot["coin_amount"].(float64); ok {
			props["coin_amount"] = int(coinAmount)
		}
		if price, ok := transaction.Snapshot["price"].(float64); ok {
			props["revenue"] = price / 100
			props["revenue_actual"] = price / 100
		}
		if v, ok := transaction.Snapshot["is_subscription_renewal"].(bool); ok && v {
			props["is_subscription_renewal"] = true
		}
	}
	if transaction.PaymentType == model.PaymentTypeSubscription {
		billing := "initial"
		if transaction.Snapshot != nil {
			if v, ok := transaction.Snapshot["stripe_subscription_billing"].(string); ok && v != "" {
				billing = v
			}
		}
		props["stripe_subscription_billing"] = billing
	}
	addTrackingContextProperties(props, transaction.Snapshot)

	requestBody := map[string]interface{}{
		"appid": ss.appID,
		"data":  eventData,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %w", err)
	}

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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("数数API返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}
