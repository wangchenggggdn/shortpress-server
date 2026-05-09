package paypal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// PayPalClient 定义 PayPal 客户端接口
type PayPalClient interface {
	SetCredentials(clientID, clientSecret string, isSandbox bool)
	GetAccessToken() (string, error)
	ListWebhooks() ([]map[string]interface{}, error)
	CreateWebhook(url string, events []string) (string, error)
	DeleteWebhook(webhookID string) error
	VerifyWebhook(webhookID string) (bool, error)
	// 订阅相关方法
	CreateProduct(name, description string) (string, error)
	CreateSubscriptionPlan(productID, name, description string, amount int64, currency, interval string) (string, error)
	CreateSubscription(planID, successURL, cancelURL string, metadata map[string]string) (string, string, error)
	CancelSubscription(subscriptionID string) error
	GetSubscriptionDetails(subscriptionID string) (map[string]interface{}, error)
}

// paypalClient 实现 PayPal 客户端接口
type paypalClient struct {
	clientID     string
	clientSecret string
	isSandbox    bool
	httpClient   *http.Client
	mutex        sync.Mutex
	accessToken  string
}

// NewClient 创建新的 PayPal 客户端
func NewClient() PayPalClient {
	return &paypalClient{
		httpClient: &http.Client{},
	}
}

// SetCredentials 设置 PayPal 凭证
func (c *paypalClient) SetCredentials(clientID, clientSecret string, isSandbox bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.clientID = clientID
	c.clientSecret = clientSecret
	c.isSandbox = isSandbox
	c.accessToken = "" // 清除旧的 access token
}

// getBaseURL 获取 PayPal API 基础 URL
func (c *paypalClient) getBaseURL() string {
	if c.isSandbox {
		return "https://api-m.sandbox.paypal.com"
	}
	return "https://api-m.paypal.com"
}

// GetAccessToken 获取 PayPal 访问令牌
func (c *paypalClient) GetAccessToken() (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 如果已有有效的 token，直接返回
	if c.accessToken != "" {
		return c.accessToken, nil
	}

	url := c.getBaseURL() + "/v1/oauth2/token"
	data := "grant_type=client_credentials"

	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(c.clientID, c.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get access token: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	c.accessToken = tokenResp.AccessToken
	return c.accessToken, nil
}

// ListWebhooks 列出所有已配置的 webhooks
func (c *paypalClient) ListWebhooks() ([]map[string]interface{}, error) {
	token, err := c.GetAccessToken()
	if err != nil {
		return nil, err
	}

	url := c.getBaseURL() + "/v1/notifications/webhooks"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list webhooks: %s", string(body))
	}

	var result struct {
		Webhooks []map[string]interface{} `json:"webhooks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Webhooks, nil
}

// CreateWebhook 创建新的 webhook
func (c *paypalClient) CreateWebhook(url string, events []string) (string, error) {
	token, err := c.GetAccessToken()
	if err != nil {
		return "", err
	}

	apiURL := c.getBaseURL() + "/v1/notifications/webhooks"

	// 构建请求体
	requestBody := map[string]interface{}{
		"url": url,
		"event_types": make([]map[string]string, len(events)),
	}

	for i, event := range events {
		requestBody["event_types"].([]map[string]string)[i] = map[string]string{
			"name": event,
		}
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create webhook: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	webhookID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("webhook ID not found in response")
	}

	return webhookID, nil
}

// DeleteWebhook 删除指定的 webhook
func (c *paypalClient) DeleteWebhook(webhookID string) error {
	token, err := c.GetAccessToken()
	if err != nil {
		return err
	}

	url := c.getBaseURL() + "/v1/notifications/webhooks/" + webhookID

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete webhook: %s", string(body))
	}

	return nil
}

// VerifyWebhook 验证指定的 webhook 是否存在
func (c *paypalClient) VerifyWebhook(webhookURL string) (bool, error) {
	webhooks, err := c.ListWebhooks()
	if err != nil {
		return false, err
	}

	for _, webhook := range webhooks {
		if url, ok := webhook["url"].(string); ok && url == webhookURL {
			return true, nil
		}
	}

	return false, nil
}

// CreateProduct 创建 PayPal 产品
func (c *paypalClient) CreateProduct(name, description string) (string, error) {
	token, err := c.GetAccessToken()
	if err != nil {
		return "", err
	}

	url := c.getBaseURL() + "/v1/catalogs/products"

	// 如果描述为空，使用默认描述
	if description == "" {
		description = name + " Subscription"
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"name":        name,
		"description": description,
		"type":        "SERVICE",
		"category":    "SOFTWARE",
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Prefer", "return=representation")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create product: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	productID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("product ID not found in response")
	}

	return productID, nil
}

// CreateSubscriptionPlan 创建 PayPal 订阅计划
func (c *paypalClient) CreateSubscriptionPlan(productID, name, description string, amount int64, currency, interval string) (string, error) {
	token, err := c.GetAccessToken()
	if err != nil {
		return "", err
	}

	url := c.getBaseURL() + "/v1/billing/plans"

	// 定义 PayPal 间隔映射
	paypalInterval := c.mapToPayPalInterval(interval)
	if paypalInterval == "" {
		return "", fmt.Errorf("invalid interval: %s", interval)
	}

	// 如果描述为空，使用默认描述
	if description == "" {
		description = name + " Subscription"
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"product_id":    productID,
		"name":          name,
		"description":   description,
		"status":        "ACTIVE",
		"billing_cycles": []map[string]interface{}{
			{
				"frequency": map[string]interface{}{
					"interval_unit":  paypalInterval,
					"interval_count": 1,
				},
				"tenure_type": "REGULAR",
				"sequence":     1,
				"total_cycles": 0, // 0 表示无限循环
				"pricing_scheme": map[string]interface{}{
					"fixed_price": map[string]interface{}{
						"value":         fmt.Sprintf("%.2f", float64(amount)/100.0),
						"currency_code": currency,
					},
				},
			},
		},
		"payment_preferences": map[string]interface{}{
			"auto_bill_outstanding":     true,
			"setup_fee_failure_action":  "CONTINUE",
			"payment_failure_threshold": 3,
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Prefer", "return=representation")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create subscription plan: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	planID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("plan ID not found in response")
	}

	return planID, nil
}

// CreateSubscription 创建 PayPal 订阅
func (c *paypalClient) CreateSubscription(planID, successURL, cancelURL string, metadata map[string]string) (string, string, error) {
	token, err := c.GetAccessToken()
	if err != nil {
		return "", "", err
	}

	url := c.getBaseURL() + "/v1/billing/subscriptions"

	// 构建请求体
	requestBody := map[string]interface{}{
		"plan_id": planID,
		"application_context": map[string]interface{}{
			"brand_name":  "ShortPress",
			"locale":      "en-US",
			"shipping_preference": "NO_SHIPPING",
			"user_action":          "SUBSCRIBE_NOW",
			"payment_method": map[string]interface{}{
				"payer_selected": "PAYPAL",
				"payee_preferred": "IMMEDIATE_PAYMENT_REQUIRED",
			},
			"return_url": successURL,
			"cancel_url": cancelURL,
		},
		"custom_id": metadata["transaction_id"],
	}

	// 添加元数据
	if len(metadata) > 0 {
		requestBody["custom_id"] = metadata["transaction_id"]
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Prefer", "return=representation")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to create subscription: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	subscriptionID, ok := result["id"].(string)
	if !ok {
		return "", "", fmt.Errorf("subscription ID not found in response")
	}

	// 提取审批链接
	var approvalURL string
	if links, ok := result["links"].([]interface{}); ok {
		for _, link := range links {
			linkMap, ok := link.(map[string]interface{})
			if !ok {
				continue
			}
			if rel, ok := linkMap["rel"].(string); ok && rel == "approve" {
				if href, ok := linkMap["href"].(string); ok {
					approvalURL = href
					break
				}
			}
		}
	}

	if approvalURL == "" {
		return "", "", fmt.Errorf("approval URL not found in response")
	}

	return subscriptionID, approvalURL, nil
}

// CancelSubscription 取消 PayPal 订阅
func (c *paypalClient) CancelSubscription(subscriptionID string) error {
	token, err := c.GetAccessToken()
	if err != nil {
		return err
	}

	url := c.getBaseURL() + "/v1/billing/subscriptions/" + subscriptionID + "/cancel"

	// 构建请求体
	requestBody := map[string]interface{}{
		"reason": "User requested cancellation",
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to cancel subscription: %s", string(body))
	}

	return nil
}

// GetSubscriptionDetails 获取订阅详情
func (c *paypalClient) GetSubscriptionDetails(subscriptionID string) (map[string]interface{}, error) {
	token, err := c.GetAccessToken()
	if err != nil {
		return nil, err
	}

	url := c.getBaseURL() + "/v1/billing/subscriptions/" + subscriptionID

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get subscription details: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// mapToPayPalInterval 将内部间隔映射到 PayPal 间隔
func (c *paypalClient) mapToPayPalInterval(interval string) string {
	switch interval {
	case "week":
		return "WEEK"
	case "month":
		return "MONTH"
	case "year":
		return "YEAR"
	default:
		return ""
	}
}
