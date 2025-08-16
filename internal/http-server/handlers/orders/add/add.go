package add

import (
	"context"
	"errors"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/middleware/auth"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/util"
	"github.com/DenisPavlov/go-musthave-diploma/internal/logger"
	"github.com/DenisPavlov/go-musthave-diploma/internal/storage"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type OrderAdder interface {
	AddOrder(ctx context.Context, orderNum string, login string) error
}

func New(log *slog.Logger, adder OrderAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := log.With(
			slog.String("component", "orders.add"),
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

		bytesBody, err := io.ReadAll(r.Body)
		if err != nil {
			msg := "error reading request body"
			log.Error(msg, logger.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.PlainText(w, r, msg)
			return
		}
		orderNumber := string(bytesBody)
		log.Info("adding order", slog.String("orderNumber", orderNumber))

		errs = util.ValidateOrderNum(orderNumber)
		if errs != nil {
			msg := strings.Join(errs, ", ")
			log.Error("invalid order number", slog.String("errors", msg))
			render.Status(r, http.StatusUnprocessableEntity)
			render.PlainText(w, r, msg)
			return
		}

		login := auth.GetUsername(r.Context())

		addErr := adder.AddOrder(r.Context(), orderNumber, login)
		if addErr != nil {
			log.Error("can not add order", logger.Err(addErr))

			if errors.Is(addErr, storage.ErrOrderWasAddedByCurrentUser) {
				render.Status(r, http.StatusOK)
				render.PlainText(w, r, "order was added by current user")
				return
			} else if errors.Is(addErr, storage.ErrOrderWasAddedByOtherUsers) {
				render.Status(r, http.StatusConflict)
				render.PlainText(w, r, "order was added by other users")
				return
			} else {
				render.Status(r, http.StatusInternalServerError)
				render.PlainText(w, r, "can not add order")
				return
			}
		}

		render.Status(r, http.StatusAccepted)
		render.PlainText(w, r, "order added successfully")
	}
}

// todo -подумать над структурой ответа функции
func validateRequest(r *http.Request) []string {
	var errs []string

	contentType := render.GetRequestContentType(r)
	if contentType != render.ContentTypePlainText {
		errs = append(errs, "invalid Content-Type")
	}

	return errs
}
