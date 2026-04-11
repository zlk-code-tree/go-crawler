package utility

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrTokenExpired = errors.New("token has expired")
	ErrTokenInvalid = errors.New("token is invalid")
)

// JwtClaims represents the claims in a JWT token.
type JwtClaims struct {
	UserId    int64  `json:"userId"`
	Username  string `json:"username"`
	CompanyId int64  `json:"companyId"`
	Exp       int64  `json:"exp"`
	Iat       int64  `json:"iat"`
}

// GenerateToken creates a JWT token.
func GenerateToken(userId int64, username string, companyId int64, secret string, expireSeconds int64) (string, error) {
	now := time.Now().Unix()
	claims := JwtClaims{
		UserId:    userId,
		Username:  username,
		CompanyId: companyId,
		Exp:       now + expireSeconds,
		Iat:       now,
	}

	headerJSON := `{"alg":"HS256","typ":"JWT"}`
	header := base64UrlEncode([]byte(headerJSON))

	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64UrlEncode(claimsBytes)

	signingInput := header + "." + payload
	signature := signHMAC(signingInput, secret)

	return signingInput + "." + signature, nil
}

// ParseToken validates and parses a JWT token.
func ParseToken(tokenStr string, secret string) (*JwtClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, ErrTokenInvalid
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSig := signHMAC(signingInput, secret)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, ErrTokenInvalid
	}

	payloadBytes, err := base64UrlDecode(parts[1])
	if err != nil {
		return nil, ErrTokenInvalid
	}

	var claims JwtClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, ErrTokenInvalid
	}

	if time.Now().Unix() > claims.Exp {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}

func signHMAC(input, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return base64UrlEncode(mac.Sum(nil))
}

func base64UrlEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func base64UrlDecode(s string) ([]byte, error) {
	remainder := len(s) % 4
	if remainder == 2 {
		s += "=="
	} else if remainder == 3 {
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
