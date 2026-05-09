package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"shortpress-server/internal/api"

	"github.com/gin-gonic/gin"
)

type fakePromptService struct {
	called bool
}

func (s *fakePromptService) OptimizePrompt(ctx *gin.Context, req *api.PromptOptimizeRequest) (*api.PromptOptimizeResponse, error) {
	s.called = true
	return &api.PromptOptimizeResponse{
		OptimizedPrompt: "optimized " + req.UserInput,
		TraceID:         "trace-test",
	}, nil
}

func TestPromptOptimizeAllowsUserTokenWithoutCreatorID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &fakePromptService{}
	handler := &PromptHandler{promptService: service}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("user_id", "user-123")
	ctx.Set("creator_id", "")
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/prompt/optimize",
		strings.NewReader(`{"user_input":"create a product image prompt"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.Optimize(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !service.called {
		t.Fatal("prompt service was not called")
	}
}

func TestPromptOptimizeRejectsMissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &fakePromptService{}
	handler := &PromptHandler{promptService: service}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("creator_id", "creator-123")
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/prompt/optimize",
		strings.NewReader(`{"user_input":"create a product image prompt"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.Optimize(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body: %s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
	if service.called {
		t.Fatal("prompt service should not be called")
	}
}
