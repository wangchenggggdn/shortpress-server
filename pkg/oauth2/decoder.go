package oauth2

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"math/big"
	"net/http"
)

type Keys struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type Claims struct {
	jwt.RegisteredClaims
	Email   string `json:"email,omitempty"`
	Name    string `json:"name,omitempty"`
	Picture string `json:"picture,omitempty"`
}

type Decoder struct {
	keyURL string
	cache  map[string]Keys
}

func NewDecoder(url string) *Decoder {
	return &Decoder{
		keyURL: url,
		cache:  make(map[string]Keys),
	}
}

// Decode 首先获取公钥信息，缓存公钥信息，再解码
func (d *Decoder) Decode(token string) (*Claims, error) {
	resp, err := http.Get(d.keyURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			Use string `json:"use"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	for _, key := range data.Keys {
		d.cache[key.Kid] = key
	}
	return d.getSubFromToken(token)
}

// 获取userID
func (d *Decoder) getSubFromToken(idToken string) (*Claims, error) {
	// 使用 ParseWithClaims 解析并验证 token
	token, err := jwt.ParseWithClaims(idToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 获取正确的公钥来验证签名
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("invalid kid in token header")
		}
		return d.getPublicKey(kid), nil
	})

	// 错误处理
	if err != nil {
		return nil, err
	} else if token == nil {
		return nil, errors.New("nil token")
	}

	// 从解析后的 token 中提取 Claims
	claims, ok := token.Claims.(*Claims)
	if ok && token.Valid {
		return claims, nil
	}

	// 如果 token 无效
	return nil, fmt.Errorf("get userID err , ok: %v valid: %v", ok, token.Valid)
}

func (d *Decoder) getPublicKey(keyId string) *rsa.PublicKey {
	//获取验证所需要的公钥
	var pubKey rsa.PublicKey
	var keys Keys
	if key, ok := d.cache[keyId]; ok {
		keys = key
		nBin, _ := base64.RawURLEncoding.DecodeString(keys.N)
		nData := new(big.Int).SetBytes(nBin)

		eBin, _ := base64.RawURLEncoding.DecodeString(keys.E)
		eData := new(big.Int).SetBytes(eBin)

		pubKey.N = nData
		pubKey.E = int(eData.Uint64())
		return &pubKey
	}
	return nil
}
