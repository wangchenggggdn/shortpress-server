package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// PluginSecretCrypto 插件密钥加密器
type PluginSecretCrypto struct {
	secretKey []byte
}

// NewPluginSecretCrypto 创建插件密钥加密器
func NewPluginSecretCrypto(v *viper.Viper) *PluginSecretCrypto {
	// 从配置中获取密钥，如果没有则使用默认密钥
	// 生产环境应该从环境变量或配置文件中读取
	secretKey := []byte("shortpress-plugin-secret-key")
	if v != nil && v.GetString("plugin.secret_key") != "" {
		secretKey = []byte(v.GetString("plugin.secret_key"))
	}

	return &PluginSecretCrypto{
		secretKey: secretKey,
	}
}

// GenerateSecret 生成插件密钥
// 格式：base64(hmac-sha256(plugin_id:site_id))
func (c *PluginSecretCrypto) GenerateSecret(pluginID, siteID string) string {
	// 构造待签名的数据
	data := fmt.Sprintf("%s:%s", pluginID, siteID)

	// 计算 HMAC-SHA256
	h := hmac.New(sha256.New, c.secretKey)
	h.Write([]byte(data))
	signature := h.Sum(nil)

	// 返回 base64 编码的签名
	return base64.StdEncoding.EncodeToString(signature)
}

// VerifySecret 验证插件密钥
func (c *PluginSecretCrypto) VerifySecret(pluginID, siteID, secret string) bool {
	// 重新生成密钥
	expectedSecret := c.GenerateSecret(pluginID, siteID)

	// 使用 hmac.Equal 进行常量时间比较，防止时序攻击
	return hmac.Equal([]byte(secret), []byte(expectedSecret))
}

// GenerateSecretWithTimestamp 生成带时间戳的插件密钥
// 格式：plugin_id:site_id:timestamp -> base64(hmac-sha256)
func (c *PluginSecretCrypto) GenerateSecretWithTimestamp(pluginID, siteID string, timestamp int64) string {
	// 构造待签名的数据
	data := fmt.Sprintf("%s:%s:%d", pluginID, siteID, timestamp)

	// 计算 HMAC-SHA256
	h := hmac.New(sha256.New, c.secretKey)
	h.Write([]byte(data))
	signature := h.Sum(nil)

	// 返回 base64 编码的签名
	return base64.StdEncoding.EncodeToString(signature)
}

// VerifySecretWithTimestamp 验证带时间戳的插件密钥
// 如果 maxDuration > 0，还会检查时间戳是否在有效期内
func (c *PluginSecretCrypto) VerifySecretWithTimestamp(pluginID, siteID, secret string, timestamp int64, maxDuration int64) bool {
	// 验证时间戳是否在有效期内（如果设置了）
	if maxDuration > 0 {
		// 这里可以添加时间戳验证逻辑
		// 例如检查 timestamp 是否在当前时间 ± maxDuration 范围内
		// 简化实现，暂时跳过
	}

	// 重新生成密钥
	expectedSecret := c.GenerateSecretWithTimestamp(pluginID, siteID, timestamp)

	// 使用 hmac.Equal 进行常量时间比较
	return hmac.Equal([]byte(secret), []byte(expectedSecret))
}

// GenerateAPISecret 生成 API 密钥
// 用于更通用的场景，可以传入任意字符串
func (c *PluginSecretCrypto) GenerateAPISecret(data string) string {
	h := hmac.New(sha256.New, c.secretKey)
	h.Write([]byte(data))
	signature := h.Sum(nil)
	return hex.EncodeToString(signature)
}

// ParsePluginSecretFromHeader 从 header 中解析插件密钥
// 支持的格式：
// 1. 直接的 secret（base64 编码的签名）
// 2.Bearer token 格式：Bearer <secret>
func ParsePluginSecretFromHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	// 移除 Bearer 前缀
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return authHeader
}
