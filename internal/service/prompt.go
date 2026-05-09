package service

import (
	"fmt"
	"strings"
	"time"

	"shortpress-server/internal/api"
	"shortpress-server/internal/common"
	"shortpress-server/internal/types"
	"shortpress-server/pkg/llm"
	"shortpress-server/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const defaultPromptOptimizeTemplate = `Please optimize the following prompt for clarity, specificity, and better generation quality.
Return only the optimized prompt text itself.
Do not include labels, headings, explanations, markdown, or surrounding quotes.

User prompt:
{{user_input}}`

var promptOptimizeLabels = []string{
	"optimized prompt:",
	"optimized prompt",
	"prompt:",
	"prompt",
}

type PromptService interface {
	OptimizePrompt(ctx *gin.Context, req *api.PromptOptimizeRequest) (*api.PromptOptimizeResponse, error)
}

type promptService struct {
	*Service
	client llm.LLM
}

func NewPromptService(service *Service) PromptService {
	grok := types.GetGrokConfig()
	return &promptService{
		Service: service,
		client: llm.LLM{
			BaseURL: grok.BaseURL,
			Model:   grok.Model,
			APIKey:  grok.APIKey,
		},
	}
}

func (s *promptService) OptimizePrompt(ctx *gin.Context, req *api.PromptOptimizeRequest) (*api.PromptOptimizeResponse, error) {
	userInput := strings.TrimSpace(req.UserInput)
	if userInput == "" {
		return nil, common.ErrBadRequest
	}
	if s.client.BaseURL == "" || s.client.Model == "" || s.client.APIKey == "" {
		return nil, fmt.Errorf("grok config is incomplete")
	}

	traceID := uuid.NewString()
	prompt := buildPromptOptimizePrompt(userInput)
	start := time.Now()

	log.AddNotice(ctx, "prompt_optimize_trace_id", traceID)
	log.AddNotice(ctx, "prompt_optimize_scene", normalizePromptScene(req.Scene))
	log.AddNotice(ctx, "prompt_optimize_input_len", len(userInput))

	response, err := s.client.Chat(prompt, 45*time.Second)
	elapsed := time.Since(start)
	log.AddNotice(ctx, "prompt_optimize_elapsed_ms", elapsed.Milliseconds())
	if err != nil {
		log.AddNotice(ctx, "prompt_optimize_success", false)
		return nil, fmt.Errorf("optimize prompt failed: %w", err)
	}

	optimizedPrompt := cleanOptimizedPrompt(string(response))
	if optimizedPrompt == "" {
		log.AddNotice(ctx, "prompt_optimize_success", false)
		return nil, fmt.Errorf("optimize prompt returned empty result")
	}

	log.AddNotice(ctx, "prompt_optimize_success", true)
	return &api.PromptOptimizeResponse{
		OptimizedPrompt: optimizedPrompt,
		TraceID:         traceID,
	}, nil
}

func buildPromptOptimizePrompt(userInput string) string {
	return strings.ReplaceAll(defaultPromptOptimizeTemplate, "{{user_input}}", userInput)
}

func cleanOptimizedPrompt(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`")
	value = strings.TrimSpace(value)

	plain := strings.Trim(value, "*_ ")
	lowerPlain := strings.ToLower(plain)
	for _, label := range promptOptimizeLabels {
		if lowerPlain == label {
			return ""
		}
		if strings.HasPrefix(lowerPlain, label) {
			value = strings.Trim(plain[len(label):], "*_ \n\t\r")
			break
		}
	}

	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"'")
	value = strings.TrimSpace(value)
	return value
}

func normalizePromptScene(scene string) string {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return "default"
	}
	return scene
}
