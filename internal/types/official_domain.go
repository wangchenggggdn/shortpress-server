package types

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	baseDomainOnce sync.Once
	baseDomain     string
)

// BaseDomain is the base domain for all official domains
// const BaseDomain = ""

// GetBaseDomain returns the base domain from configuration or default
func GetBaseDomain() string {
	return baseDomain
}

// InitBaseDomain initializes the base domain from configuration
func InitBaseDomain(conf *viper.Viper) {
	baseDomainOnce.Do(func() {
		if conf != nil {
			baseDomain = conf.GetString("domain.hosting")
		}
	})
}

// OfficialDomain represents a subdomain for shortpress.com
type OfficialDomain struct {
	Subdomain string
}

// String returns the complete domain name (subdomain.shortpress.com)
func (od *OfficialDomain) String() string {
	if od.Subdomain == "" {
		return GetBaseDomain()
	}
	return fmt.Sprintf("%s.%s", od.Subdomain, GetBaseDomain())
}

// GetSubdomain returns the subdomain part only
func (od *OfficialDomain) GetSubdomain() string {
	return od.Subdomain
}

// SetSubdomain sets a new subdomain
func (od *OfficialDomain) SetSubdomain(subdomain string) {
	// Clean the subdomain - remove any existing domain suffix and trim spaces
	cleaned := strings.TrimSpace(subdomain)
	cleaned = strings.TrimSuffix(cleaned, "."+GetBaseDomain())
	od.Subdomain = cleaned
}

// MarshalJSON implements json.Marshaler interface
func (od *OfficialDomain) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, od.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler interface
func (od *OfficialDomain) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	str := strings.Trim(string(data), `"`)

	// Extract subdomain from full domain
	domainSuffix := "." + GetBaseDomain()
	if strings.HasSuffix(str, domainSuffix) {
		od.Subdomain = strings.TrimSuffix(str, domainSuffix)
	} else {
		od.Subdomain = str
	}

	return nil
}

// ContainsBaseDomain checks if a string contains the BaseDomain
func ContainsBaseDomain(str string) bool {
	return strings.Contains(str, GetBaseDomain())
}

// IsOfficialDomain checks if a string is an official domain (ends with BaseDomain)
func IsOfficialDomain(str string) bool {
	baseDomain := GetBaseDomain()
	if str == baseDomain {
		return false
	}
	return strings.HasSuffix(str, "."+baseDomain)
}

// ExtractSubdomain extracts the subdomain part before BaseDomain
// For example: "xxx.myshortpress.com" returns "xxx"
// Returns empty string if the input doesn't end with BaseDomain
func ExtractSubdomain(str string) string {
	baseDomain := GetBaseDomain()
	if str == baseDomain || !strings.HasSuffix(str, "."+baseDomain) {
		return ""
	}

	parts := strings.Split(str, ".")
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}
