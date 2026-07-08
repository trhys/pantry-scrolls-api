package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/trhys/Recipe-Repo-2/internal/auth"
	"github.com/trhys/Recipe-Repo-2/internal/metrics"
)

func (cfg *ApiConfig) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		cookie, err := r.Cookie("jwt")
		if err == nil {
			tokenString = cookie.Value
		}

		if tokenString == "" {
			token, err := auth.GetBearerToken(r.Header)
			if err == nil {
				tokenString = token
			}
		}

		if tokenString == "" {
			ctx := context.WithValue(r.Context(), "userID", "")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		subject, err := auth.ValidateJWT(tokenString, cfg.Secret)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token", error_description="token is expired"`)
				respondFail(r, w, 401, "Expired token", nil)
				return
			}
			ip := getClientIP(r)
			respondFail(r, w, 401, "Unauthorized", fmt.Errorf("Unauthorized access attempt from IP: %s - ERROR: %v", ip, err))
			return
		}

		ctx := context.WithValue(r.Context(), "userID", subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (cfg *ApiConfig) MetricsMiddleware(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			path := r.Pattern
			if path == "" {
				path = "unknown"
			}

			next.ServeHTTP(w, r)

			duration := time.Since(start).Seconds()
			m.Latency.WithLabelValues(path).Observe(duration)
			m.ServerHits.WithLabelValues(path).Inc()
		})
	}
}
