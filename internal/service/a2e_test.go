package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"shortpress-server/internal/api"
	"shortpress-server/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestA2EServiceInvokeWan27ProxiesGenerateRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotAuth string
	var gotSiteID string
	var gotReq a2eGenerateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generate" {
			t.Fatalf("path = %q, want /generate", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		gotSiteID = r.Header.Get(middleware.SiteIDHeader)
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":{"task_id":"task-1","model":"a2eWan2.7"}}`))
	}))
	defer server.Close()

	ctx := testGinContextWithA2EHeaders()
	svc := &a2eService{
		client:     server.Client(),
		serviceURL: server.URL,
		wan27Model: a2eWan27Model,
	}

	resp, err := svc.InvokeWan27(ctx, &api.A2EWan27InvokeRequest{
		VideoID:  "vid-1",
		Number:   2,
		ImageURL: "https://example.com/source.jpg",
		Name:     "fitness video",
		Prompt:   "make a video",
		Args: map[string]any{
			"video_time": "5",
		},
	})
	if err != nil {
		t.Fatalf("InvokeWan27() error = %v", err)
	}

	if gotAuth != "Bearer user-token" {
		t.Fatalf("Authorization header = %q", gotAuth)
	}
	if gotSiteID != "site-1" {
		t.Fatalf("%s header = %q", middleware.SiteIDHeader, gotSiteID)
	}
	if gotReq.Model != a2eWan27Model {
		t.Fatalf("model = %q", gotReq.Model)
	}
	if gotReq.VideoID != "vid-1" || gotReq.Number != 2 {
		t.Fatalf("request metadata = %#v", gotReq)
	}
	if gotReq.Args["prompt"] != "make a video" {
		t.Fatalf("prompt arg = %#v", gotReq.Args["prompt"])
	}
	if gotReq.Args["image_url"] != "https://example.com/source.jpg" {
		t.Fatalf("image_url arg = %#v", gotReq.Args["image_url"])
	}
	if gotReq.Args["name"] != "fitness video" {
		t.Fatalf("name arg = %#v", gotReq.Args["name"])
	}
	if gotReq.Args["model"] != a2eWan27ProviderModel {
		t.Fatalf("model arg = %#v", gotReq.Args["model"])
	}
	if gotReq.Args["video_time"] != "5" {
		t.Fatalf("video_time arg = %#v", gotReq.Args["video_time"])
	}
	if resp.TaskID != "task-1" || resp.Model != a2eWan27Model || resp.TraceID == "" {
		t.Fatalf("response = %#v", resp)
	}
}

func TestBuildWan27ArgsPreservesExplicitPromptArg(t *testing.T) {
	args := buildWan27Args(&api.A2EWan27InvokeRequest{
		ImageURL: "https://example.com/source.jpg",
		Name:     "top-level name",
		Prompt:   "top-level prompt",
		Args: map[string]any{
			"name":   "args name",
			"prompt": "args prompt",
			"model":  "wan2.6-i2v",
		},
	})

	if args["name"] != "args name" {
		t.Fatalf("name = %#v", args["name"])
	}
	if args["prompt"] != "args prompt" {
		t.Fatalf("prompt = %#v", args["prompt"])
	}
	if args["image_url"] != "https://example.com/source.jpg" {
		t.Fatalf("image_url = %#v", args["image_url"])
	}
	if args["model"] != "wan2.6-i2v" {
		t.Fatalf("model = %#v", args["model"])
	}
}

func TestA2EServiceInvokeWan27RequiresOfficialFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx := testGinContextWithA2EHeaders()
	svc := &a2eService{
		client:     http.DefaultClient,
		serviceURL: "http://example.com",
		wan27Model: a2eWan27Model,
	}

	_, err := svc.InvokeWan27(ctx, &api.A2EWan27InvokeRequest{
		ImageURL: "https://example.com/source.jpg",
		Prompt:   "make a video",
	})
	if err == nil {
		t.Fatal("InvokeWan27() error = nil, want missing name error")
	}
	if err.Error() != "name is required" {
		t.Fatalf("error = %q", err.Error())
	}
}

func testGinContextWithA2EHeaders() *gin.Context {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/a2e/wan2_7/invoke", nil)
	req.Header.Set("Authorization", "Bearer user-token")
	req.Header.Set(middleware.SiteIDHeader, "site-1")
	ctx.Request = req
	return ctx
}
