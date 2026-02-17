package middleware

import (
	"eventsBot/internal/utils/jwt"
	"context"
	"log/slog"
	"net/http"
	"strings"
)

func Authorization(secret string) func(next http.Handler) http.Handler {
	op := "middleware.Authorization()"
	log := slog.With(
		slog.String("op", op))
	return func(next http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			splitToken := strings.Split(authHeader, "Bearer ")
			if len(splitToken) != 2 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			tokenString := splitToken[1]
			if tokenString == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			user, err := jwt.ParseAndValidateToken[UserClaims](tokenString, secret)
			if err != nil {
				log.Error("error parse jwt token", slog.String("error", err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			type contextKey string
			const userKey contextKey = "user"
			ctx := context.WithValue(r.Context(), userKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("",
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("url", r.RequestURI),
		)
		next.ServeHTTP(w, r)
	})
}
