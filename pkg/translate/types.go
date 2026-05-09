package translate

// PlaylistI18nItem 单个语言的播放列表 i18n 数据
type PlaylistI18nItem struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Tags           string `json:"tags"`
	SeoTitle       string `json:"seo_title"`
	SeoDescription string `json:"seo_description"`
	SeoKeywords    string `json:"seo_keywords"`
}

// PlaylistTranslateReq 播放列表翻译请求参数
type PlaylistTranslateReq struct {
	PlaylistI18nItem
}

// PlaylistTranslateResp 播放列表翻译响应参数
type PlaylistTranslateResp struct {
	Language string `json:"language"`
	PlaylistI18nItem
}

// PageI18nItem 单个语言的播放列表 i18n 数据
type PageI18nItem struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
}

// PageTranslateReq 页面翻译请求参数
type PageTranslateReq struct {
	PageI18nItem
}

// PageTranslateResp 页面翻译响应参数
type PageTranslateResp struct {
	Language string `json:"language"`
	PageI18nItem
}
