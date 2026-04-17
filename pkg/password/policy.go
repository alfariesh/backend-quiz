package password

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
)

var (
	ErrTooShort    = errors.New("password: too short")
	ErrTooLong     = errors.New("password: too long")
	ErrTooWeak     = errors.New("password: too weak")
	ErrCompromised = errors.New("password: appears in known data breaches")
)

// Policy validates password inputs.
type Policy struct {
	MinLength      int
	MaxLength      int
	RequireMixed   bool // require letters + digits
	HIBPChecker    *HIBPChecker
}

// Validate returns nil if password meets all rules.
func (p Policy) Validate(ctx context.Context, password string) error {
	if len(password) < p.MinLength {
		return ErrTooShort
	}
	if p.MaxLength > 0 && len(password) > p.MaxLength {
		return ErrTooLong
	}
	if p.RequireMixed {
		var hasLetter, hasDigit bool
		for _, r := range password {
			switch {
			case unicode.IsLetter(r):
				hasLetter = true
			case unicode.IsDigit(r):
				hasDigit = true
			}
		}
		if !hasLetter || !hasDigit {
			return ErrTooWeak
		}
	}
	if p.HIBPChecker != nil {
		pwned, err := p.HIBPChecker.IsCompromised(ctx, password)
		if err != nil {
			// Fail-open: if HIBP API is down, don't block registration.
			return nil
		}
		if pwned {
			return ErrCompromised
		}
	}
	return nil
}

// HIBPChecker queries the HaveIBeenPwned range API using k-anonymity:
// only the first 5 hex chars of SHA-1 are sent over the wire.
type HIBPChecker struct {
	client *http.Client
}

func NewHIBPChecker() *HIBPChecker {
	return &HIBPChecker{client: &http.Client{Timeout: 3 * time.Second}}
}

// IsCompromised returns true if the password appears in the HIBP breach corpus.
func (h *HIBPChecker) IsCompromised(ctx context.Context, password string) (bool, error) {
	sum := sha1.Sum([]byte(password))
	hash := strings.ToUpper(hex.EncodeToString(sum[:]))
	prefix, suffix := hash[:5], hash[5:]

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("https://api.pwnedpasswords.com/range/%s", prefix), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "backend-quiz")
	req.Header.Set("Add-Padding", "true")

	resp, err := h.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("hibp: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		// format: SUFFIX:COUNT
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(parts[0], suffix) {
			return true, nil
		}
	}
	return false, nil
}
