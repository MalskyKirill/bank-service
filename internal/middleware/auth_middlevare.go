package middleware

import (
	"bank-service/internal/response"
	"bank-service/internal/security"
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userIDKey contextKey = "userID"

func AuthMiddleware(jwtService *security.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.WriteError(w, http.StatusUnauthorized, "authorization header is required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.WriteError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			tokenString := strings.TrimSpace(parts[1])
			if tokenString == "" {
				response.WriteError(w, http.StatusUnauthorized, "token is required")
				return
			}

			userId, err := jwtService.ParceToken(tokenString)
			if err != nil {
				response.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userId)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func userIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}
