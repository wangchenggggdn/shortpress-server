package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"

	"github.com/spf13/viper"
)

var subtitleHost = ""

// SetSubtitleHostConf sets the subtitle host from configuration
func SetSubtitleHostConf(conf *viper.Viper) {
	subtitleHost = conf.GetString("storage.local.video_host")
}

// SubtitleItem represents a single subtitle track
type SubtitleItem struct {
	Path string `json:"path"` // Relative path to subtitle file
	Desc string `json:"desc"` // Description/language description
}

// SubtitleTracks is a custom type that stores subtitle tracks as JSON in database
// JSON structure: {"en": {"path": "path/to/en.srt", "desc": "English"}, "zh": {"path": "path/to/zh.srt", "desc": "中文"}}
type SubtitleTracks map[string]SubtitleItem

// Value implements driver.Valuer interface for database storage
func (s SubtitleTracks) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}

	// Clean paths before storing - ensure they are relative paths
	cleaned := make(map[string]SubtitleItem)
	for lang, item := range s {
		cleanedPath := item.Path

		// Remove any host prefix, keeping only the path
		if strings.HasPrefix(cleanedPath, "http://") || strings.HasPrefix(cleanedPath, "https://") || strings.HasPrefix(cleanedPath, "//") {
			if strings.HasPrefix(cleanedPath, subtitleHost) {
				cleanedPath = strings.TrimPrefix(cleanedPath, subtitleHost)
			} else {
				// Extract path from URL
				parts := strings.SplitN(cleanedPath, "/", 4)
				if len(parts) >= 4 {
					cleanedPath = parts[3] // Get path part after domain
				}
			}
		}

		// Validate path format
		if cleanedPath != "" && !strings.HasPrefix(cleanedPath, "/videolib/") && !strings.HasPrefix(cleanedPath, "videolib/") &&
			!strings.HasPrefix(cleanedPath, "/res/") && !strings.HasPrefix(cleanedPath, "res/") {
			// return nil, errors.New("subtitle path must start with videolib or res")
			continue
		}

		if strings.HasPrefix(cleanedPath, "//") {
			cleanedPath = strings.TrimPrefix(cleanedPath, "/")
		}

		cleaned[lang] = SubtitleItem{
			Path: cleanedPath,
			Desc: item.Desc,
		}
	}

	return json.Marshal(cleaned)
}

// Scan implements sql.Scanner interface for reading from database
func (s *SubtitleTracks) Scan(value interface{}) error {
	if value == nil {
		*s = make(SubtitleTracks)
		return nil
	}

	var jsonData []byte
	switch v := value.(type) {
	case []byte:
		jsonData = v
	case string:
		jsonData = []byte(v)
	default:
		return errors.New("invalid scan source for SubtitleTracks")
	}

	var tracks map[string]SubtitleItem
	if err := json.Unmarshal(jsonData, &tracks); err != nil {
		return err
	}

	*s = tracks
	return nil
}

// MarshalJSON automatically adds host during JSON serialization for API responses
func (s SubtitleTracks) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return json.Marshal(map[string]SubtitleItem{})
	}

	// Convert relative paths to full URLs for API response
	result := make(map[string]SubtitleItem)
	for lang, item := range s {
		fullPath := item.Path

		// If it's already a full URL, keep it as is
		if !strings.HasPrefix(fullPath, "http") && fullPath != "" {
			if !strings.HasPrefix(fullPath, "/") {
				fullPath = "/" + fullPath
			}
			fullPath = strings.TrimSuffix(subtitleHost, "/") + fullPath
		}

		result[lang] = SubtitleItem{
			Path: fullPath,
			Desc: item.Desc,
		}
	}

	return json.Marshal(result)
}

// UnmarshalJSON handles JSON unmarshaling for API requests
func (s *SubtitleTracks) UnmarshalJSON(data []byte) error {
	var tracks map[string]SubtitleItem
	if err := json.Unmarshal(data, &tracks); err != nil {
		return err
	}

	*s = tracks
	return nil
}

// AddSubtitle adds or updates a subtitle track
func (s *SubtitleTracks) AddSubtitle(lang, path, desc string) {
	if *s == nil {
		*s = make(SubtitleTracks)
	}
	(*s)[lang] = SubtitleItem{
		Path: path,
		Desc: desc,
	}
}

// RemoveSubtitle removes a subtitle track
func (s *SubtitleTracks) RemoveSubtitle(lang string) {
	if *s != nil {
		delete(*s, lang)
	}
}

// GetSubtitle gets a subtitle track by language
func (s SubtitleTracks) GetSubtitle(lang string) (SubtitleItem, bool) {
	if s == nil {
		return SubtitleItem{}, false
	}
	item, exists := s[lang]
	return item, exists
}

// HasSubtitles returns true if there are any subtitles
func (s SubtitleTracks) HasSubtitles() bool {
	return len(s) > 0
}

// GetLanguages returns all available subtitle languages
func (s SubtitleTracks) GetLanguages() []string {
	languages := make([]string, 0, len(s))
	for lang := range s {
		languages = append(languages, lang)
	}
	return languages
}
