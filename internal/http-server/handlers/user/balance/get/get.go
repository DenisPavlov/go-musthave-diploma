package get

import (
	"context"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/middleware/auth"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/util"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
)

type UserBalanceGetter interface {
	GetBalance(ctx context.Context, username string) (Balance, error)
}

type Balance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

func New(log *slog.Logger, balanceGetter UserBalanceGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := log.With(
			slog.String("component", "balance.get"),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		username := auth.GetUsername(r.Context())

		if balance, err := balanceGetter.GetBalance(r.Context(), username); err != nil {
			util.LogAngSendStatusInternalServerError(log, w, r, "can not get balance", err)
			return
		} else {
			render.JSON(w, r, balance)
		}
	}
}
