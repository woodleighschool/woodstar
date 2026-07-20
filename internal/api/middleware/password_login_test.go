package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/time/rate"
)

func TestPasswordLoginLimiterAllowsConfiguredBurst(t *testing.T) {
	limiter := NewPasswordLoginLimiter()
	if got, want := limiter.Limit(), rate.Every(time.Minute/10); got != want {
		t.Fatalf("rate = %v, want %v", got, want)
	}
	if got := limiter.Burst(); got != 4 {
		t.Fatalf("burst = %d, want 4", got)
	}
	now := time.Now()
	for attempt := 1; attempt <= 4; attempt++ {
		if !limiter.AllowN(now, 1) {
			t.Fatalf("attempt %d was rejected inside the burst", attempt)
		}
	}
	if limiter.AllowN(now, 1) {
		t.Fatal("fifth immediate attempt was admitted")
	}
}

func TestPasswordLoginRateLimitStopsBeforeHandler(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(time.Hour), 1)
	handlerCalls := 0
	handler := LimitPasswordLogin(limiter)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalls++
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/login", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusNoContent)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/login", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if handlerCalls != 1 {
		t.Fatalf("handler calls = %d, want 1", handlerCalls)
	}
	retryAfter, err := strconv.Atoi(second.Header().Get("Retry-After"))
	if err != nil || retryAfter < 1 {
		t.Fatalf("Retry-After = %q, want positive integer seconds", second.Header().Get("Retry-After"))
	}
	if got := second.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", got)
	}
	var problem huma.ErrorModel
	if err := json.Unmarshal(second.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem response: %v", err)
	}
	if problem.Status != http.StatusTooManyRequests || problem.Detail == "" {
		t.Fatalf("problem = %+v, want 429 with detail", problem)
	}
}
