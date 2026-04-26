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

			// Ensure user is active (Skip for verification route)
			if !user.IsActive && !strings.Contains(r.URL.Path, "/verify-signup") {
				http.Error(w, "account is deactivated", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", user.ID)
			ctx = context.WithValue(ctx, "role", user.Role)
			ctx = context.WithValue(ctx, "permissions", user.Permissions)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminMiddleware() func(http.Handler) http.Handler {
	return RoleMiddleware("admin")
}

func RoleMiddleware(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleVal := r.Context().Value("role")
			if roleVal == nil {
				http.Error(w, "forbidden: role not found", http.StatusForbidden)
				return
			}

			userRole := roleVal.(string)
			isAllowed := false
			for _, role := range allowedRoles {
				if userRole == role {
					isAllowed = true
					break
				}
			}

			if !isAllowed {
				http.Error(w, "forbidden: insufficient permissions", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func PermissionMiddleware(requiredPermission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleVal := r.Context().Value("role")
			if roleVal == nil {
				http.Error(w, "forbidden: role not found", http.StatusForbidden)
				return
			}

			userRole := roleVal.(string)
			// Admins bypass all permission checks
			if userRole == "admin" {
				next.ServeHTTP(w, r)
				return
			}

			if userRole != "moderator" {
				http.Error(w, "forbidden: role must be moderator or admin", http.StatusForbidden)
				return
			}

			permissionsVal := r.Context().Value("permissions")
			if permissionsVal == nil {
				http.Error(w, "forbidden: permissions not found", http.StatusForbidden)
				return
			}

			permissions := permissionsVal.([]string)
			hasPermission := false
			for _, p := range permissions {
				if p == requiredPermission {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				http.Error(w, "forbidden: missing required permission: "+requiredPermission, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
