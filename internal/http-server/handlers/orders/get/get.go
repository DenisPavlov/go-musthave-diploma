package get

import (
	"context"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/middleware/auth"
	"github.com/DenisPavlov/go-musthave-diploma/internal/logger"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	"time"
)

type OrderGetter interface {
	GetAllOrders(ctx context.Context, username string) ([]Order, error)
}

// todo - переместить куда-от?
type Order struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    int       `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

func New(log *slog.Logger, getter OrderGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := log.With(
			slog.String("component", "orders.get"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		username := auth.GetUsername(r.Context())
		orders, err := getter.GetAllOrders(r.Context(), username)
		if err != nil {
			msg := "can not get orders"
			log.Error(msg, logger.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.PlainText(w, r, msg)
			return
		}

		if len(orders) == 0 {
			render.Status(r, http.StatusNoContent)
			render.PlainText(w, r, "nothing found")
		}

		render.JSON(w, r, orders)
	}
}
