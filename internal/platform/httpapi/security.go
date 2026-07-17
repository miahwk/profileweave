package httpapi

import "net/http"

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		headers.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Cross-Origin-Resource-Policy", "same-origin")
		headers.Set("Permissions-Policy", "camera=(), geolocation=(), microphone=(), payment=(), usb=()")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("X-DNS-Prefetch-Control", "off")
		if r.URL.Path == "/api" || len(r.URL.Path) > 5 && r.URL.Path[:5] == "/api/" {
			headers.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}
