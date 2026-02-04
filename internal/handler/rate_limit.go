package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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

// RateLimitedLimitsHandler обрабатывает GET/POST /v1/limits/rate-limited и /api/v1/limits/rate-limited.
// При превышении лимита возвращает 429.
func RateLimitedLimitsHandler(limiter *RateLimitState) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.Allow(ip) {
			c.Header("Retry-After", "1")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "too many requests",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "rate-limited endpoint",
		})
	}
}
