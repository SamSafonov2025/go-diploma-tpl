package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/auth"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/helpers"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
)

type contextKey string

const userContextKey contextKey = "user"

// ParseUserCredentials middleware parses user credentials from request body
func ParseUserCredentials(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userCreds models.User
		if err := json.NewDecoder(r.Body).Decode(&userCreds); err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		if err := userCreds.ValidateCredentials(); err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, userCreds)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthRequired middleware validates JWT token
func AuthRequired() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Bearer token
			authHeader := r.Header.Get("Authorization")
			tokenParts := strings.Split(authHeader, "Bearer ")

			if len(tokenParts) != 2 {
				helpers.RespondWithError(w, models.ErrInvalidTokenFormat)
				return
			}

			// Validate token and get user ID
			userID, err := auth.ExtractUserIDFromToken(tokenParts[1])
			if err != nil {
				helpers.RespondWithError(w, models.ErrUnauthorized)
				return
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), userContextKey, models.User{ID: userID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ExtractUserFromContext retrieves user from request context
func ExtractUserFromContext(ctx context.Context) (models.User, error) {
	userInfo, ok := ctx.Value(userContextKey).(models.User)
	if !ok {
		return models.User{}, models.ErrUnauthorized
	}
	return userInfo, nil
}
