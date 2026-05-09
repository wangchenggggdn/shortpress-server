package translate

import (
	"encoding/json"
	"fmt"
	"shortpress-server/pkg/llm"
)

type Translator struct {
	llm            llm.LLM
	TargetLanguage string
}

func NewTranslator(baseURL, model, apiKey, targetLanguage string) Translator {
	return Translator{
		llm: llm.LLM{
			BaseURL:    baseURL,
			Model:      model,
			APIKey:     apiKey,
			JSONFormat: true,
		},
		TargetLanguage: targetLanguage,
	}
}

func (t *Translator) TranslatePlaylist(content PlaylistTranslateReq) ([]PlaylistTranslateResp, error) {
	// 构建提示词
	prompt := fmt.Sprintf(
		playlistTranslatePrompt,
		t.TargetLanguage,
		content.Title,
		content.Description,
		content.Tags,
		content.SeoTitle,
		content.SeoDescription,
		content.SeoKeywords,
	)

	// 调用 LLM 获取 JSON 响应
	response, err := t.llm.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("handle translation error: %w", err)
	}

	// 解析 JSON 响应
	var multiLangData []PlaylistTranslateResp
	if err = json.Unmarshal(response, &multiLangData); err != nil {
		return nil, fmt.Errorf("validate translate result failed: %w, raw_response: %s", err, string(response))
	}

	return multiLangData, nil
}

func (t *Translator) TranslatePage(content PageTranslateReq) ([]PageTranslateResp, error) {
	// 构建提示词
	prompt := fmt.Sprintf(
		pageTranslatePrompt,
		t.TargetLanguage,
		content.Name,
		content.Title,
		content.Description,
		content.Keywords,
	)

	// 调用 LLM 获取 JSON 响应
	response, err := t.llm.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("handle translate error: %w", err)
	}

	// 解析 JSON 响应
	var multiLangData []PageTranslateResp
	if err = json.Unmarshal(response, &multiLangData); err != nil {
		return nil, fmt.Errorf("validate translate result failed: %w, raw_response: %s", err, string(response))
	}

	return multiLangData, nil
}
