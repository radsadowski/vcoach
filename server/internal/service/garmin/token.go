package garmin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// tokenPath exchanges a Personal API key for a short-lived bearer JWT.
const tokenPath = "/personalapitoken-service/personal-api/personal-api-key/token"

// BearerToken is the JWT the token endpoint returns, with its expiry.
type BearerToken struct {
	AccessToken string
	ExpiresAt   time.Time
}

// ErrNoAPIKey is returned when no Personal API key is configured.
var ErrNoAPIKey = fmt.Errorf("garmin: no personal API key configured")

// bearerToken returns a valid bearer token, exchanging the API key when the
// cached JWT is missing or within one hour of expiry.
func (c *HTTPClient) bearerToken(ctx context.Context) (*BearerToken, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.apiKey == "" {
		return nil, ErrNoAPIKey
	}
	if c.token != nil && time.Until(c.token.ExpiresAt) > time.Hour {
		return c.token, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+tokenPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "ApiKey "+c.apiKey)
	req.Header.Set("Accept", "*/*")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("garmin token exchange: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &UnauthorizedError{}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPStatusError{Path: tokenPath, StatusCode: resp.StatusCode}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("garmin token exchange: read body: %w", err)
	}
	accessToken := strings.TrimSpace(string(body))
	if accessToken == "" {
		return nil, fmt.Errorf("garmin token exchange: empty token response")
	}

	expiresAt, err := jwtExpiration(accessToken)
	if err != nil {
		return nil, err
	}
	c.token = &BearerToken{AccessToken: accessToken, ExpiresAt: expiresAt}
	return c.token, nil
}

// jwtExpiration reads the exp claim without verifying the signature.
func jwtExpiration(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("garmin token exchange: response is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("garmin token exchange: decode JWT payload: %w", err)
	}
	var claims struct {
		ExpiresAt float64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, fmt.Errorf("garmin token exchange: parse JWT payload: %w", err)
	}
	if claims.ExpiresAt == 0 {
		return time.Time{}, fmt.Errorf("garmin token exchange: JWT has no exp claim")
	}
	return time.Unix(int64(claims.ExpiresAt), 0), nil
}
