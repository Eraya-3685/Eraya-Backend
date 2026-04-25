package middleware

import (
	"context"
	"eraya/user"
	"eraya/util"
	"net/http"
	"strings"
)

func AuthMiddleware(secret string, userSvc user.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			tokenString := ""

			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				// Try query param for WebSockets
				tokenString = r.URL.Query().Get("token")
			}

			if tokenString == "" {
				http.Error(w, "missing or invalid token", http.StatusUnauthorized)
				return
			}

			claims, err := util.ValidateJWT(tokenString, secret)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Verify user still exists in DB
			user, err := userSvc.GetProfile(r.Context(), claims.UserID)
			if err != nil || user == nil {
				http.Error(w, "user no longer exists or inactive", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "role", claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleVal := r.Context().Value("role")
			if roleVal == nil || roleVal.(string) != "admin" {
				http.Error(w, "forbidden: admin access required", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
