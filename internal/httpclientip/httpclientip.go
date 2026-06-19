// Package httpclientip extracts the normalized client IP from HTTP requests.
package httpclientip

import (
	"net"
	"net/http"
	"net/netip"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// FromRequest returns chi's configured client IP, falling back to RemoteAddr.
func FromRequest(r *http.Request) string {
	if ip := chimiddleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return parseIP(host)
	}
	return parseIP(r.RemoteAddr)
}

func parseIP(value string) string {
	ip, err := netip.ParseAddr(value)
	if err != nil {
		return ""
	}
	return ip.String()
}
