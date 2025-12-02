// File: internal/middleware/middleware.go
package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"azlo-goboiler/internal/config"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

type Middleware struct {
	app *config.Application
}

func New(app *config.Application) *Middleware {
	return &Middleware{app: app}
}

// --- RESPONSE WRITER for logging ---
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// --- REQUEST ID MIDDLEWARE ---
func (mw *Middleware) RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), "request_id", requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// --- ENHANCED LOGGING MIDDLEWARE ---
func (mw *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := getRequestID(r.Context())

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		// Log request with detailed information
		logEvent := mw.app.Logger.Info()

		// Add error level for 4xx and 5xx responses
		if wrapped.statusCode >= 400 {
			if wrapped.statusCode >= 500 {
				logEvent = mw.app.Logger.Error()
			} else {
				logEvent = mw.app.Logger.Warn()
			}
		}

		logEvent.
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("query", r.URL.RawQuery).
			Int("status", wrapped.statusCode).
			Dur("duration", duration).
			Str("ip", getClientIP(r)).
			Str("user_agent", r.UserAgent()).
			Int64("content_length", r.ContentLength).
			Int("response_size", wrapped.size).
			Msg("HTTP request processed")
	})
}

// --- ENHANCED RECOVERY MIDDLEWARE ---
func (mw *Middleware) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := getRequestID(r.Context())

				mw.app.Logger.Error().
					Str("request_id", requestID).
					Str("panic", fmt.Sprintf("%v", err)).
					Bytes("stack", debug.Stack()).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("Panic recovered")

				// Return a generic error response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"success":false,"error":"Internal server error","request_id":"` + requestID + `"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// --- ENHANCED JWT MIDDLEWARE ---
func (mw *Middleware) JWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := getRequestID(r.Context())

		// Read the token from the secure cookie
		cookie, err := r.Cookie("jwt_token")
		if err != nil {
			mw.app.Logger.Warn().
				Str("request_id", requestID).
				Msg("Missing auth cookie")
			writeJSONError(w, http.StatusUnauthorized, "Auth cookie required", requestID)
			return
		}

		tokenString := cookie.Value
		claims := &jwt.RegisteredClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(mw.app.Config.App_Secret), nil
		})

		if err != nil {
			status := http.StatusUnauthorized
			msg := "Invalid token"

			if errors.Is(err, jwt.ErrTokenExpired) {
				msg = "Token has expired"
				mw.app.Logger.Warn().
					Str("request_id", requestID).
					Str("user_id", claims.Subject).
					Msg("Expired token used")
			} else {
				mw.app.Logger.Warn().
					Str("request_id", requestID).
					Err(err).
					Msg("Token validation failed")
			}

			writeJSONError(w, status, msg, requestID)
			return
		}

		if !token.Valid {
			mw.app.Logger.Warn().
				Str("request_id", requestID).
				Msg("Invalid token used")
			writeJSONError(w, http.StatusUnauthorized, "Invalid token", requestID)
			return
		}

		// Add user ID and request ID to context
		ctx := context.WithValue(r.Context(), config.UserIDKey, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// --- REDIS-BASED RATE LIMITER ---
type RedisRateLimiter struct {
	app   *config.Application
	rate  int
	burst int
}

func NewRedisRateLimiter(app *config.Application, rate, burst int) *RedisRateLimiter {
	return &RedisRateLimiter{
		app:   app,
		rate:  rate,
		burst: burst,
	}
}

func (rl *RedisRateLimiter) Allow(ip string) bool {
	ctx := context.Background()
	key := fmt.Sprintf("rate_limit:%s", ip)

	// Use Redis with sliding window algorithm
	now := time.Now().Unix()
	windowStart := now - 60 // 1-minute window

	pipe := rl.app.Redis.Pipeline()

	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart, 10))

	// Count current requests in window
	countCmd := pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, &redis.Z{Score: float64(now), Member: now})

	// Set expiration
	pipe.Expire(ctx, key, time.Minute*2)

	_, err := pipe.Exec(ctx)
	if err != nil {
		// If Redis fails, allow the request (fail open)
		rl.app.Logger.Warn().Err(err).Msg("Redis rate limiter failed, allowing request")
		return true
	}

	// Get the count
	count := countCmd.Val()
	return count <= int64(rl.rate)
}

// --- FALLBACK IN-MEMORY RATE LIMITER ---
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type MemoryRateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

func NewMemoryRateLimiter(rps int, burst int) *MemoryRateLimiter {
	rl := &MemoryRateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(rps),
		burst:    burst,
	}
	go rl.cleanupVisitors()
	return rl
}

func (rl *MemoryRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *MemoryRateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 15*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (mw *Middleware) RateLimit(next http.Handler) http.Handler {
	// Try Redis-based rate limiting first, fallback to memory-based
	var redisLimiter *RedisRateLimiter
	var memoryLimiter *MemoryRateLimiter

	if mw.app.Redis != nil {
		redisLimiter = NewRedisRateLimiter(mw.app, mw.app.Config.RateLimit, mw.app.Config.RateLimit*2)
	} else {
		memoryLimiter = NewMemoryRateLimiter(mw.app.Config.RateLimit, mw.app.Config.RateLimit*2)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := getRequestID(r.Context())
		ip := getClientIP(r)

		var allowed bool
		if redisLimiter != nil {
			allowed = redisLimiter.Allow(ip)
		} else {
			allowed = memoryLimiter.getLimiter(ip).Allow()
		}

		if !allowed {
			mw.app.Logger.Warn().
				Str("request_id", requestID).
				Str("ip", ip).
				Msg("Rate limit exceeded")
			writeJSONError(w, http.StatusTooManyRequests, "Rate limit exceeded", requestID)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- ENHANCED SECURITY MIDDLEWARE ---
func Security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")

		// Remove server information
		w.Header().Set("Server", "")

		next.ServeHTTP(w, r)
	})
}

// --- TIMEOUT MIDDLEWARE ---
func (mw *Middleware) Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)

			done := make(chan bool)
			go func() {
				next.ServeHTTP(w, r)
				done <- true
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				requestID := getRequestID(r.Context())
				mw.app.Logger.Warn().
					Str("request_id", requestID).
					Dur("timeout", timeout).
					Msg("Request timeout")
				writeJSONError(w, http.StatusRequestTimeout, "Request timeout", requestID)
				return
			}
		})
	}
}

// --- HELPER FUNCTIONS ---

func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return "unknown"
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Get the first IP (client IP)
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

func writeJSONError(w http.ResponseWriter, status int, message, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	response := fmt.Sprintf(`{"success":false,"error":"%s","request_id":"%s"}`, message, requestID)
	w.Write([]byte(response))
}
