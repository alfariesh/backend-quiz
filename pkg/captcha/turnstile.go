package captcha

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// Verifier verifies human-response tokens.
type Verifier interface {
	Verify(ctx context.Context, token, clientIP string) error
}

// Turnstile is a Cloudflare Turnstile verifier.
type Turnstile struct {
	secret string
	client *http.Client
}

func NewTurnstile(secret string) *Turnstile {
	return &Turnstile{
		secret: secret,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

type turnstileResponse struct {
	Success     bool     `json:"success"`
	ErrorCodes  []string `json:"error-codes"`
	Hostname    string   `json:"hostname"`
	Action      string   `json:"action"`
	ChallengeTS string   `json:"challenge_ts"`
}

// Verify returns nil when the token is valid.
func (t *Turnstile) Verify(ctx context.Context, token, clientIP string) error {
	if token == "" {
		return ErrMissingToken
	}
	form := url.Values{}
	form.Set("secret", t.secret)
	form.Set("response", token)
	if clientIP != "" {
		form.Set("remoteip", clientIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var tr turnstileResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return err
	}
	if !tr.Success {
		if len(tr.ErrorCodes) > 0 {
			return &Error{Code: strings.Join(tr.ErrorCodes, ",")}
		}
		return ErrInvalidToken
	}
	return nil
}

// Error is returned when verification fails with a specific upstream code.
type Error struct {
	Code string
}

func (e *Error) Error() string { return "captcha: " + e.Code }

var (
	ErrMissingToken = errors.New("captcha: missing token")
	ErrInvalidToken = errors.New("captcha: invalid token")
)

// NoopVerifier accepts every token (for dev/tests when not configured).
type NoopVerifier struct{}

func (NoopVerifier) Verify(ctx context.Context, token, clientIP string) error { return nil }
