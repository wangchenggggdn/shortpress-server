// pkg/types/media_url.go
package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

var imageHost = ""

// Set global media host
func SetImageHostConf(conf *viper.Viper) {
	imageHost = conf.GetString("storage.local.image_host")
}

// MediaURL is a special type that stores relative paths in the database and automatically converts to full URLs when returned by API
type ImageURL string

// Value implements driver.Valuer interface, automatically converts before storing in database
func (m ImageURL) Value() (driver.Value, error) {
	// Before storing in database, ensure it's a relative path
	u := string(m)
	if u == "" {
		return nil, nil
	}
	// Remove any host prefix, keeping only the path
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "//") {
		if strings.HasPrefix(u, imageHost) {
			// If URL starts with the imageHost, remove that prefix
			u = strings.TrimPrefix(u, imageHost)
			fmt.Println("Removing image host prefix:", u)
		} else {
			// Otherwise, parse the URL to get the path
			return nil, errors.New("media path must start with image host")
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
func (m *ImageURL) Scan(value interface{}) error {
	if value == nil {
		*m = ""
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*m = ImageURL(v)
	case string:
		*m = ImageURL(v)
	default:
		return errors.New("invalid scan source")
	}
	return nil
}

// MarshalJSON automatically adds host during JSON serialization
func (m ImageURL) MarshalJSON() ([]byte, error) {
	// If empty value or already a complete URL, return directly
	if m == "" || strings.HasPrefix(string(m), "http") {
		return json.Marshal(string(m))
	}

	// // Add host
	// path := string(m)
	// if !strings.HasPrefix(path, "/") {
	//     path = "/" + path
	// }

	fullURL := strings.TrimRight(imageHost, "/") + "/" + strings.TrimLeft(string(m), "/")
	return json.Marshal(fullURL)
}
