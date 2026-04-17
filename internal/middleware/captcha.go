package middleware

import (
	"net/http"

	"github.com/alfariesh/backend-quiz/pkg/captcha"
)

// Captcha returns a middleware that requires a valid captcha token from the
// X-Captcha-Token header. Passing nil (or NoopVerifier) disables the check.
func Captcha(v captcha.Verifier, trustForwarded bool) func(http.Handler) http.Handler {
	if v == nil {
		v = captcha.NoopVerifier{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Captcha-Token")
			if err := v.Verify(r.Context(), token, ClientIP(r, trustForwarded)); err != nil {
				http.Error(w, `{"error":"captcha verification failed"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
