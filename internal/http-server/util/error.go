package util

import (
	"github.com/DenisPavlov/go-musthave-diploma/internal/logger"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
)

func LogAngSendStatusInternalServerError(log *slog.Logger, w http.ResponseWriter, r *http.Request, msg string, err error) {
	log.Error(msg, logger.Err(err))
	render.Status(r, http.StatusInternalServerError)
	render.PlainText(w, r, msg)
}
