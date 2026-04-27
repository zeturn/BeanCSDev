package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

func SaveToken(profile string, token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return SaveRaw(profile, string(data))
}

func LoadToken(profile string) (*oauth2.Token, error) {
	raw, err := LoadRaw(profile)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal([]byte(raw), &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func Claims(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return map[string]any{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return map[string]any{}
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return map[string]any{}
	}
	return claims
}

func ExpiryString(token *oauth2.Token) string {
	if token == nil || token.Expiry.IsZero() {
		return "unknown"
	}
	return token.Expiry.Local().Format(time.RFC3339)
}
