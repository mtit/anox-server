package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const accessTokenTTL = 12 * time.Hour

type accessTokenClaims struct {
	IssuedAt  int64 `json:"iat"`
	ExpiresAt int64 `json:"exp"`
}

func (s *HTTPServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader("Authorization"))
		if token == "" {
			c.JSON(401, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
		settings, err := s.currentAnoxSettings()
		if err != nil || !validateAccessToken(token, settings.Secret, time.Now()) {
			c.JSON(401, gin.H{"error": "invalid credentials"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (s *HTTPServer) handleLogin(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	settings, err := s.currentAnoxSettings()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to load settings"})
		return
	}

	if req.Password != settings.Pass {
		c.JSON(401, gin.H{"error": "invalid password"})
		return
	}

	token, err := issueAccessToken(settings.Secret, time.Now())
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to issue token"})
		return
	}

	c.JSON(200, gin.H{
		"token":      token,
		"expires_in": int64(accessTokenTTL.Seconds()),
	})
}

func (s *HTTPServer) currentAnoxSettings() (settingsSecretGetter, error) {
	settings, err := s.configStore.GetAnoxSettings()
	if err != nil {
		return settingsSecretGetter{}, err
	}
	return settingsSecretGetter{
		Pass:   settings.Pass,
		Secret: settings.Secret,
	}, nil
}

type settingsSecretGetter struct {
	Pass   string
	Secret string
}

func issueAccessToken(secret string, now time.Time) (string, error) {
	claims := accessTokenClaims{
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(accessTokenTTL).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := signTokenPayload(encodedPayload, secret)
	return encodedPayload + "." + signature, nil
}

func validateAccessToken(token, secret string, now time.Time) bool {
	payload, signature, ok := strings.Cut(token, ".")
	if !ok || payload == "" || signature == "" {
		return false
	}

	expectedSignature := signTokenPayload(payload, secret)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return false
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return false
	}

	var claims accessTokenClaims
	if err := json.Unmarshal(rawPayload, &claims); err != nil {
		return false
	}
	return claims.ExpiresAt > now.Unix()
}

func signTokenPayload(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
