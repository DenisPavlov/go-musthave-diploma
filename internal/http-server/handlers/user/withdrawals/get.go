package withdrawals

import (
	"context"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/middleware/auth"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/util"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	"time"
)

type WithdrawalGetter interface {
	GetWithdrawals(ctx context.Context, username string) ([]Withdrawal, error)
}

type Withdrawal struct {
	Order       string    `json:"order"`
	Sum         float32   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

func New(log *slog.Logger, getter WithdrawalGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := log.With(
			slog.String("component", "withdrawals.get"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		username := auth.GetUsername(r.Context())

		withdrawals, err := getter.GetWithdrawals(r.Context(), username)
		if err != nil {
			util.LogAngSendStatusInternalServerError(log, w, r, "can not get all withdrawals", err)
			return
		}

		if len(withdrawals) == 0 {
			render.Status(r, http.StatusNoContent)
			render.PlainText(w, r, "there are no withdrawal")
			return
		}

		render.JSON(w, r, withdrawals)
	}
}
