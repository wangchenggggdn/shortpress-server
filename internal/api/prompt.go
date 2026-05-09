package api

// PromptOptimizeRequest represents a prompt optimization request.
type PromptOptimizeRequest struct {
	Scene     string `json:"scene,omitempty" example:"default"`
	UserInput string `json:"user_input" binding:"required" example:"Create a landing page for my fitness course"`
}

// PromptOptimizeResponse represents a prompt optimization response.
type PromptOptimizeResponse struct {
	OptimizedPrompt string `json:"optimized_prompt" example:"Create a clear, conversion-focused landing page for a fitness course..."`
	TraceID         string `json:"trace_id,omitempty" example:"1f2c9d2a-86c5-4b76-bb5b-a5d3fd5600c5"`
}
