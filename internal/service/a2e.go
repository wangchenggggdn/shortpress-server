package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"shortpress-server/internal/api"
	"shortpress-server/internal/middleware"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

const (
	a2eWan27Model         = "a2eWan2.7"
	a2eWan27ProviderModel = "wan2.7-i2v"
)

type a2eHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type A2EService interface {
	InvokeWan27(ctx *gin.Context, req *api.A2EWan27InvokeRequest) (*api.A2EWan27InvokeResponse, error)
}

type a2eService struct {
	*Service
	client     a2eHTTPClient
	serviceURL string
	timeout    time.Duration
	retryCount int
	wan27Model string
}

type a2eGenerateRequest struct {
	Model   string         `json:"model"`
	VideoID string         `json:"video_id,omitempty"`
	Number  int            `json:"number,omitempty"`
	Args    map[string]any `json:"args"`
}

type a2eUpstreamResponse struct {
	Code    int            `json:"code"`
	Message string         `json:"message,omitempty"`
	Error   string         `json:"error,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

func NewA2EService(service *Service, conf *viper.Viper) A2EService {
	timeout := conf.GetDuration("a2e.timeout")
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	retryCount := conf.GetInt("a2e.retry_count")
	if retryCount < 0 {
		retryCount = 0
	}

	return &a2eService{
		Service:    service,
		client:     &http.Client{Timeout: timeout},
		serviceURL: strings.TrimRight(conf.GetString("a2e.generate_service_url"), "/"),
		timeout:    timeout,
		retryCount: retryCount,
		wan27Model: a2eWan27Model,
	}
}

func (s *a2eService) InvokeWan27(ctx *gin.Context, req *api.A2EWan27InvokeRequest) (*api.A2EWan27InvokeResponse, error) {
	if s.serviceURL == "" {
		return nil, fmt.Errorf("a2e generate service url is not configured")
	}

	traceID := uuid.NewString()
	args := buildWan27Args(req)
	if missing := missingWan27RequiredArgs(args); len(missing) > 0 {
		return nil, fmt.Errorf("%s is required", strings.Join(missing, ", "))
	}

	payload := a2eGenerateRequest{
		Model:   s.wan27Model,
		VideoID: strings.TrimSpace(req.VideoID),
		Number:  req.Number,
		Args:    args,
	}

	log.AddNotice(ctx, "a2e_trace_id", traceID)
	log.AddNotice(ctx, "a2e_model", s.wan27Model)
	log.AddNotice(ctx, "a2e_action", "invoke")
	log.AddNotice(ctx, "a2e_args_keys", strings.Join(mapKeys(args), ","))

	upstream, err := s.doJSON(ctx, http.MethodPost, s.serviceURL+"/generate", payload)
	if err != nil {
		return nil, err
	}
	if upstream.Data == nil {
		return nil, fmt.Errorf("a2e generate service returned empty data")
	}

	return &api.A2EWan27InvokeResponse{
		TaskID:      stringFromMap(upstream.Data, "task_id"),
		Model:       stringFromMap(upstream.Data, "model"),
		TraceID:     traceID,
		RawResponse: upstream.Data,
	}, nil
}

func (s *a2eService) doJSON(ctx *gin.Context, method, endpoint string, payload any) (*a2eUpstreamResponse, error) {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal a2e request failed: %w", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt <= s.retryCount; attempt++ {
		start := time.Now()
		upstream, retryable, err := s.doJSONOnce(ctx, method, endpoint, body)
		log.AddNotice(ctx, "a2e_elapsed_ms", time.Since(start).Milliseconds())
		if err == nil {
			log.AddNotice(ctx, "a2e_success", true)
			return upstream, nil
		}
		lastErr = err
		log.AddNotice(ctx, "a2e_success", false)
		if !retryable || attempt == s.retryCount {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
	}
	return nil, lastErr
}

func (s *a2eService) doJSONOnce(ctx *gin.Context, method, endpoint string, body []byte) (*a2eUpstreamResponse, bool, error) {
	req, err := http.NewRequestWithContext(ctx.Request.Context(), method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("create a2e request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", ctx.GetHeader("Authorization"))
	req.Header.Set(middleware.SiteIDHeader, ctx.GetHeader(middleware.SiteIDHeader))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("send a2e request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("read a2e response failed: %w", err)
	}

	retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, retryable, fmt.Errorf("a2e generate service returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var upstream a2eUpstreamResponse
	if err = json.Unmarshal(respBody, &upstream); err != nil {
		return nil, false, fmt.Errorf("parse a2e response failed: %w", err)
	}
	if upstream.Error != "" {
		return nil, false, fmt.Errorf("a2e generate service error: %s", firstNonEmpty(upstream.Message, upstream.Error))
	}
	return &upstream, false, nil
}

func buildWan27Args(req *api.A2EWan27InvokeRequest) map[string]any {
	args := make(map[string]any, len(req.Args)+4)
	for k, v := range req.Args {
		args[k] = v
	}
	if imageURL := strings.TrimSpace(req.ImageURL); imageURL != "" {
		if _, exists := args["image_url"]; !exists {
			args["image_url"] = imageURL
		}
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		if _, exists := args["name"]; !exists {
			args["name"] = name
		}
	}
	if prompt := strings.TrimSpace(req.Prompt); prompt != "" {
		if _, exists := args["prompt"]; !exists {
			args["prompt"] = prompt
		}
	}
	if model := strings.TrimSpace(req.Model); model != "" {
		if _, exists := args["model"]; !exists {
			args["model"] = model
		}
	}
	if _, exists := args["model"]; !exists {
		args["model"] = a2eWan27ProviderModel
	}
	return args
}

func missingWan27RequiredArgs(args map[string]any) []string {
	required := []string{"image_url", "name", "prompt"}
	missing := make([]string, 0, len(required))
	for _, key := range required {
		value, ok := args[key]
		if !ok || strings.TrimSpace(fmt.Sprint(value)) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func stringFromMap(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
