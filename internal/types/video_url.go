// pkg/types/media_url.go
package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

var videoHost = ""

// Set global media host
func SetVideoHostConf(conf *viper.Viper) {
	videoHost = conf.GetString("storage.local.video_host")
}

// VideoUrl is a special type that stores relative paths in the database and automatically converts to full URLs when returned by API
type VideoUrl string

// Value implements driver.Valuer interface, automatically converts before storing in database
func (m VideoUrl) Value() (driver.Value, error) {
	// Before storing in database, ensure it's a relative path
	u := string(m)
	if u == "" {
		return nil, nil
	}
	// Remove any host prefix, keeping only the path
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "//") {
		parsedURL, err := url.Parse(u)
		if err == nil {
			u = parsedURL.Path
		}
	} else if (!strings.HasPrefix(u, "/videolib/") && !strings.HasPrefix(u, "videolib/")) &&
		(!strings.HasPrefix(u, "/res/") && !strings.HasPrefix(u, "res/")) {
		return nil, errors.New("media path must start with videolib or res")
	}
	if strings.HasPrefix(u, "//") {
		u = strings.TrimPrefix(u, "/")
	}
	return u, nil
}

// Scan implements sql.Scanner interface, automatically converts when reading from database
func (m *VideoUrl) Scan(value interface{}) error {
	if value == nil {
		*m = ""
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*m = VideoUrl(v)
	case string:
		*m = VideoUrl(v)
	default:
		return errors.New("invalid scan source")
	}
	return nil
}

// MarshalJSON automatically adds host during JSON serialization
func (m VideoUrl) MarshalJSON() ([]byte, error) {
	// If empty value or already a complete URL, return directly
	if m == "" || strings.HasPrefix(string(m), "http") {
		return json.Marshal(string(m))
	}

	// Add host
	path := string(m)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	fullURL := strings.TrimSuffix(videoHost, "/") + path
	return json.Marshal(fullURL)
}
