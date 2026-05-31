package middleware

import (
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// IdempotencyMiddleware intercepts incoming requests to enforce exact-once processing.
func IdempotencyMiddleware(redisClient *redis.Client, lockTTL time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract the unique key from the header
			idempotencyKey := r.Header.Get("X-Idempotency-Key")

			// If the client didn't provide a key, bypass middleware or enforce it.
			// For a critical webhook API, we enforce it.
			if idempotencyKey == "" {
				http.Error(w, "Missing required X-Idempotency-Key header", http.StatusBadRequest)
				return
			}

			ctx := r.Context()
			lockKey := "lock:" + idempotencyKey

			// Atomic Set-if-Not-Exists (NX) lock acquisition
			success, err := redisClient.SetNX(ctx, lockKey, "processing", lockTTL).Result()
			if err != nil {
				http.Error(w, "Internal cache validation failure", http.StatusInternalServerError)
				return
			}

			if !success {
				// The key already exists in Redis, indicating a duplicate request
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"status": "ignored", "message": "Duplicate event transaction detected via middleware"}`))
				return
			}

			// If successful, pass the request down to the actual handler
			next.ServeHTTP(w, r)
		})
	}
}
