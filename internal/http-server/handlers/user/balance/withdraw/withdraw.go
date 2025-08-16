package withdraw

import (
	"context"
	"errors"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/middleware/auth"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/util"
	"github.com/DenisPavlov/go-musthave-diploma/internal/logger"
	"github.com/DenisPavlov/go-musthave-diploma/internal/storage"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	"strings"
)

type Withdraw interface {
	Withdraw(ctx context.Context, username string, orderNum string, sum float32) error
}

type RequestData struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

func New(log *slog.Logger, withdraw Withdraw) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := log.With(
			slog.String("component", "balance.withdraw"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		username := auth.GetUsername(r.Context())

		var req RequestData
		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			msg := "invalid request body"
			log.Error(msg, logger.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.PlainText(w, r, msg)
			return
		}

		errs := util.ValidateOrderNum(req.Order)
		if errs != nil {
			msg := strings.Join(errs, ", ")
			log.Error("invalid order number", slog.String("errors", msg))
			render.Status(r, http.StatusUnprocessableEntity)
			render.PlainText(w, r, msg)
			return
		}

		err = withdraw.Withdraw(r.Context(), username, req.Order, req.Sum)

		if err != nil {
			log.Error("failed to withdraw", logger.Err(err))

			if errors.Is(err, storage.ErrNotEnough) {
				msg := "not enough coins to withdraw"
				log.Error(msg, logger.Err(err))
				render.Status(r, http.StatusPaymentRequired)
				render.PlainText(w, r, msg)
				return
			} else if errors.Is(err, storage.ErrWrongOrderNumber) {
				msg := "wrong order number"
				log.Error(msg, slog.String("errors", msg))
				render.Status(r, http.StatusUnprocessableEntity)
				render.PlainText(w, r, msg)
				return
			} else {
				util.LogAngSendStatusInternalServerError(log, w, r, "can not withdraw", err)
				return
			}
		}
		render.Status(r, http.StatusOK)
	}
}
