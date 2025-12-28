package authMiddlware

import (
	"context"
	"net/http"
	"strings"

	resp "main_service/internal/lib/api/response"
	"main_service/internal/lib/jwt"

	"github.com/go-chi/render"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func AuthMiddleware(jwtParser jwt.JWTParser) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, resp.Error("Missing authorization"))
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			userID, err := jwtParser.ParseToken(token)
			if err != nil {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, resp.Error("Invalid token"))
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
