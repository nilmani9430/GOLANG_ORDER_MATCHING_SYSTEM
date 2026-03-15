package middleware

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/GOLANG-ORDER-MATCHING-SYSTEM/internal/logger"
	"github.com/google/uuid"
)

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(logger *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := uuid.New().String()
			ctx := context.WithValue(r.Context(), "request_id", requestID)

			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a custom ResponseWriter to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log request only if logger is provided
			if logger != nil {
				duration := time.Since(start)
				requestID := r.Context().Value("request_id")

				logger.Logger.Info("HTTP request completed",
					"request_id", requestID,
					"method", r.Method,
					"url", r.URL.String(),
					"status", wrapped.statusCode,
					"duration_ms", duration.Milliseconds(),
					"user_agent", r.UserAgent(),
					"remote_addr", r.RemoteAddr)
			}
		})
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			// w.Header().Set("Access-Control-Max-Age", "86400")

			// Handle preflight requests
			// Handle OPTIONS requests for CORS preflight
			// Browsers send OPTIONS requests before certain cross-origin requests
			// to check if the actual request will be allowed
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK) // Return 200 OK to indicate CORS is allowed
				return // Stop here, no need to process the preflight request further
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware recovers from panics and logs them
func RecoveryMiddleware(logger *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := r.Context().Value("request_id")

					logger.Logger.Error("Panic recovered",
						"request_id", requestID,
						"error", err,
						"stack", string(debug.Stack()),
						"method", r.Method,
						"url", r.URL.String())

					// Write error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, `{"error": "Internal server error"}`)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware adds a timeout to requests
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a channel to signal completion
			done := make(chan struct{})

			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				// Request completed normally
			case <-ctx.Done():
				// Request timed out
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestTimeout)
				fmt.Fprintf(w, `{"error": "Request timeout"}`)
			}
		})
	}
}

// SecurityMiddleware adds security headers
func SecurityMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add security headers
			// Prevent MIME-type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")
			
			// Prevent clickjacking by disabling iframe embedding
			w.Header().Set("X-Frame-Options", "DENY")
			
			// Enable browser's XSS filter
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			
			// Enforce HTTPS for a specified time period
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			
			// Control how much referrer information is included with requests
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
