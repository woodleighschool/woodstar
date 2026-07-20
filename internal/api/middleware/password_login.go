package middleware

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/time/rate"
)

const (
	passwordLoginAttemptsPerMinute = 10
	passwordLoginBurst             = 4
)

// NewPasswordLoginLimiter returns the process-local limiter for password login.
func NewPasswordLoginLimiter() *rate.Limiter {
	return rate.NewLimiter(
		rate.Every(time.Minute/passwordLoginAttemptsPerMinute),
		passwordLoginBurst,
	)
}

// LimitPasswordLogin rejects password-login requests that exceed limiter.
func LimitPasswordLogin(limiter *rate.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter.Allow() {
				next.ServeHTTP(w, r)
				return
			}

			const message = "too many login attempts; try again shortly"
			w.Header().Set("Content-Type", "application/problem+json")
			w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(limiter)))
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(huma.NewError(http.StatusTooManyRequests, message))
		})
	}
}

func retryAfterSeconds(limiter *rate.Limiter) int {
	missingTokens := 1 - limiter.Tokens()
	if missingTokens <= 0 {
		return 1
	}

	seconds := int(math.Ceil(missingTokens / float64(limiter.Limit())))
	return max(seconds, 1)
}
