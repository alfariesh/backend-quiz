package middleware

import (
	"net/http"
	"strings"
)

// CORS returns a middleware that only allows listed origins. Pass one or more
// origins (e.g. "https://app.example.com"). Pass a single "*" to allow any
// origin (dev only — disables credentials). Empty list means: no CORS headers
// emitted; browser calls from any origin will be blocked.
func CORS(allowedOrigins ...string) func(http.Handler) http.Handler {
	allowAny := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"
	allowed := map[string]struct{}{}
	for _, o := range allowedOrigins {
		if o == "" {
			continue
		}
		allowed[strings.TrimRight(o, "/")] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			h := w.Header()

			switch {
			case allowAny:
				h.Set("Access-Control-Allow-Origin", "*")
			case origin != "":
				if _, ok := allowed[strings.TrimRight(origin, "/")]; ok {
					h.Set("Access-Control-Allow-Origin", origin)
					h.Set("Access-Control-Allow-Credentials", "true")
					h.Add("Vary", "Origin")
				}
			}

			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID, X-Captcha-Token")
			h.Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
