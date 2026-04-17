package middleware

import "net/http"

type SecurityHeadersOptions struct {
	// HSTSMaxAgeSeconds > 0 adds Strict-Transport-Security. Set 0 to disable (e.g. dev over HTTP).
	HSTSMaxAgeSeconds int
	// HSTSIncludeSubdomains adds "includeSubDomains" when HSTS is enabled.
	HSTSIncludeSubdomains bool
	// HSTSPreload adds "preload" when HSTS is enabled.
	HSTSPreload bool
	// ContentSecurityPolicy — empty string disables.
	ContentSecurityPolicy string
}

func SecurityHeaders(opts SecurityHeadersOptions) func(http.Handler) http.Handler {
	hstsValue := ""
	if opts.HSTSMaxAgeSeconds > 0 {
		hstsValue = "max-age=" + itoa(opts.HSTSMaxAgeSeconds)
		if opts.HSTSIncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		if opts.HSTSPreload {
			hstsValue += "; preload"
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Cross-Origin-Resource-Policy", "same-site")
			if hstsValue != "" {
				h.Set("Strict-Transport-Security", hstsValue)
			}
			if opts.ContentSecurityPolicy != "" {
				h.Set("Content-Security-Policy", opts.ContentSecurityPolicy)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
