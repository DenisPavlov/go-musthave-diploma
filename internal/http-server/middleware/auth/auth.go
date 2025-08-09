package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/DenisPavlov/go-musthave-diploma/internal/utils"
	"github.com/go-chi/render"
)

type ctxUsernameKey int

const UsernameKey ctxUsernameKey = 0

func New(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log := log.With(
			slog.String("component", "middleware/auth"),
		)
		log.Info("auth middleware enabled")

		fn := func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				msg := "Authorization header is required"
				log.Debug(msg)
				render.Status(r, http.StatusUnauthorized)
				render.PlainText(w, r, msg)
				return
			}

			tokenString := authHeader[len("Bearer "):]
			claims, err := utils.ParseJWT(tokenString)
			if err != nil {
				msg := "Invalid token"
				log.Debug(msg)
				render.Status(r, http.StatusUnauthorized)
				render.PlainText(w, r, msg)
				return
			}

			ctx := context.WithValue(r.Context(), UsernameKey, claims["sub"])
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

func GetUsername(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	username, ok := ctx.Value(UsernameKey).(string)
	if ok {
		return username
	} else {
		return ""
	}
}
