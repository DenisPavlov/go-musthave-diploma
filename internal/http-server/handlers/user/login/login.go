package login

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/user/model"
	"github.com/DenisPavlov/go-musthave-diploma/internal/logger"
	"github.com/DenisPavlov/go-musthave-diploma/internal/storage"
	"github.com/DenisPavlov/go-musthave-diploma/internal/utils"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type UserLogin interface {
	Login(ctx context.Context, username string, password string) error
}

func New(log *slog.Logger, login UserLogin) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := log.With(
			slog.String("component", "user.login"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		errs := validateRequest(r)
		if errs != nil {
			msg := strings.Join(errs, ", ")
			log.Error("invalid request", slog.String("errors", msg))
			render.Status(r, http.StatusBadRequest)
			render.PlainText(w, r, msg)
			return
		}

		var req model.User
		err := render.DecodeJSON(r.Body, &req)
		if err != nil || req.Login == "" || req.Password == "" {
			msg := "invalid request body"
			log.Error(msg, logger.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.PlainText(w, r, msg)
			return
		}

		if err := login.Login(r.Context(), req.Login, req.Password); err != nil {
			if errors.Is(err, storage.ErrUserNotFound) || errors.Is(err, storage.ErrWrongPassword) {
				msg := "wrong username or password"
				log.Error(msg, logger.Err(err), slog.String("login", req.Login))
				render.Status(r, http.StatusUnauthorized)
				render.PlainText(w, r, msg)
				return
			} else {
				msg := "could not login user"
				log.Error(msg, logger.Err(err), slog.String("login", req.Login))
				render.Status(r, http.StatusInternalServerError)
				render.PlainText(w, r, msg)
				return
			}
		}

		token, err := utils.GenerateJWT(req.Login)
		if err != nil {
			msg := "could not create token"
			log.Error(msg, logger.Err(err), slog.String("login", req.Login))
			render.Status(r, http.StatusInternalServerError)
			render.PlainText(w, r, msg)
			return
		}

		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))

	}
}

// todo - подумать как лучше сделать валидацию
func validateRequest(r *http.Request) []string {
	var errs []string

	contentType := render.GetRequestContentType(r)
	if contentType != render.ContentTypeJSON {
		errs = append(errs, "invalid Content-Type")
	}

	return errs
}
