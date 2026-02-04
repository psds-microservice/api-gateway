package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimitState — простой in-memory rate limiter по IP (скользящее окно, max N запросов).
type RateLimitState struct {
	mu        sync.Mutex
	perIP     map[string]*rateWindow
	limit     int
	windowSec time.Duration
}

type rateWindow struct {
	count       int
	windowStart time.Time
}

// NewRateLimitState создаёт лимитер: limit запросов на windowSec (например 5 на 1 сек).
func NewRateLimitState(limit int, windowSec time.Duration) *RateLimitState {
	s := &RateLimitState{
		perIP:     make(map[string]*rateWindow),
		limit:     limit,
		windowSec: windowSec,
	}
	go s.cleanup()
	return s
}

func (s *RateLimitState) cleanup() {
	ticker := time.NewTicker(2 * s.windowSec)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		for ip, w := range s.perIP {
			if time.Since(w.windowStart) > 2*s.windowSec {
				delete(s.perIP, ip)
			}
		}
		s.mu.Unlock()
	}
}

// Allow возвращает true, если запрос разрешён, false если лимит превышен.
func (s *RateLimitState) Allow(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	w, ok := s.perIP[ip]
	if !ok {
		s.perIP[ip] = &rateWindow{count: 1, windowStart: now}
		return true
	}
	if now.Sub(w.windowStart) >= s.windowSec {
		w.count = 1
		w.windowStart = now
		return true
	}
	w.count++
	return w.count <= s.limit
}

// RateLimitedLimitsHandler возвращает http.HandlerFunc для /v1/limits/rate-limited (net/http).
func RateLimitedLimitsHandler(limiter *RateLimitState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !limiter.Allow(ip) {
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded", "message": "too many requests",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok", "message": "rate-limited endpoint",
		})
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host != "" {
		return host
	}
	return r.RemoteAddr
}
