package llm

import "testing"

func TestChatCompletionsURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "base without v1",
			baseURL: "https://api.x.ai",
			want:    "https://api.x.ai/v1/chat/completions",
		},
		{
			name:    "base with v1",
			baseURL: "https://api.x.ai/v1",
			want:    "https://api.x.ai/v1/chat/completions",
		},
		{
			name:    "trailing slash",
			baseURL: "https://api.x.ai/",
			want:    "https://api.x.ai/v1/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatCompletionsURL(tt.baseURL); got != tt.want {
				t.Fatalf("chatCompletionsURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
