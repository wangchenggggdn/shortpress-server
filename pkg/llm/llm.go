package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ChatRequest OpenAI 兼容的聊天请求结构
type ChatRequest struct {
	Model          string              `json:"model"`
	Messages       []ReqChatMessage    `json:"messages"`
	ResponseFormat *ChatResponseFormat `json:"response_format,omitempty"`
}

// ReqChatMessage 聊天消息结构
type ReqChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RespChatMessage 聊天消息结构
type RespChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponseFormat 响应格式结构
type ChatResponseFormat struct {
	Type string `json:"type"`
}

// ChatResponse OpenAI 兼容的聊天响应结构
type ChatResponse struct {
	Choices []struct {
		Message RespChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

type LLM struct {
	APIKey     string
	BaseURL    string
	Model      string
	JSONFormat bool
}

// Chat 调用 LLM API 进行对话
func (l *LLM) Chat(prompt string, timeout ...time.Duration) ([]byte, error) {
	// 构建请求体
	reqBody := ChatRequest{
		Model: l.Model,
		Messages: []ReqChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// 根据 jsonFormat 字段设置 response_format
	if l.JSONFormat {
		reqBody.ResponseFormat = &ChatResponseFormat{
			Type: "json_object",
		}
	}

	// 序列化请求体
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 构建 HTTP 请求
	url := chatCompletionsURL(l.BaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.APIKey)

	// 发送请求
	var client *http.Client
	if len(timeout) != 0 {
		client = &http.Client{
			Timeout: timeout[0],
		}
	} else {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误: %s, status_code: %d", string(body), resp.StatusCode)
	}

	// 解析响应
	var chatResp ChatResponse
	if err = json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, raw_response: %s", err, string(body))
	}

	// 检查 API 错误
	if chatResp.Error != nil {
		return nil, fmt.Errorf("%s, type: %s, code: %s", chatResp.Error.Message, chatResp.Error.Type, chatResp.Error.Code)
	}

	// 返回消息内容（转换为 []byte）
	if len(chatResp.Choices) > 0 {
		return []byte(chatResp.Choices[0].Message.Content), nil
	}

	return nil, fmt.Errorf("API 返回空响应")
}

func chatCompletionsURL(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL + "/chat/completions"
	}
	return baseURL + "/v1/chat/completions"
}
