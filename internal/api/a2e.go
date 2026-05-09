package api

// A2EWan27InvokeRequest represents a wan2.7 generation request proxied to the
// video generation service.
type A2EWan27InvokeRequest struct {
	VideoID  string         `json:"video_id,omitempty" example:"vid-123456"`
	Number   int            `json:"number,omitempty" example:"1"`
	ImageURL string         `json:"image_url" binding:"required" example:"https://example.com/source.jpg"`
	Name     string         `json:"name" binding:"required" example:"Fitness intro"`
	Prompt   string         `json:"prompt" binding:"required" example:"Create a cinematic fitness intro video"`
	Model    string         `json:"model,omitempty" example:"wan2.7-i2v"`
	Args     map[string]any `json:"args,omitempty"`
}

// A2EWan27InvokeResponse represents the standardized invoke response.
type A2EWan27InvokeResponse struct {
	TaskID      string `json:"task_id"`
	Model       string `json:"model"`
	TraceID     string `json:"trace_id"`
	RawResponse any    `json:"raw_response,omitempty"`
}
