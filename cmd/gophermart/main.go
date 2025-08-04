package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/DenisPavlov/go-musthave-diploma/internal/config"
	addOrders "github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/orders/add"
	getOrders "github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/orders/get"
	getBalance "github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/user/balance/get"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/user/balance/withdraw"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/user/login"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/user/register"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/handlers/user/withdrawals"
	"github.com/DenisPavlov/go-musthave-diploma/internal/http-server/middleware/auth"
	mwLogger "github.com/DenisPavlov/go-musthave-diploma/internal/http-server/middleware/logger"
	"github.com/DenisPavlov/go-musthave-diploma/internal/logger"
	ordersStorage "github.com/DenisPavlov/go-musthave-diploma/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {

	// init config
	cfg := config.MustLoad()

	// init logger
	log := logger.SetupLogger(cfg.Env)
	log.Info("starting gophermart", slog.Any("config", cfg))
	log.Debug("debug was enabled")

	// init router
	router := chi.NewRouter()

	// middleware
	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	storage, err := ordersStorage.InitStorage(log, cfg)
	if err != nil {
		log.Error("error initializing storage", logger.Err(err))
		os.Exit(1)
	}

	router.Post("/api/user/register", register.New(log, storage))
	router.Post("/api/user/login", login.New(log, storage))

	router.Route("/api/user", func(r chi.Router) {
		r.Use(auth.New(log))
		r.Route("/orders", func(r chi.Router) {
			r.Post("/", addOrders.New(log, storage))
			r.Get("/", getOrders.New(log, storage))
		})
		r.Route("/balance", func(r chi.Router) {
			r.Get("/", getBalance.New(log, storage))
			r.Post("/withdraw", withdraw.New(log, storage))
		})
		r.Get("/withdrawals", withdrawals.New(log, storage))
	})

	// start server
	log.Info("server starting", slog.String("address", cfg.RunAddress))
	srv := &http.Server{
		Addr:         cfg.RunAddress,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server", logger.Err(err))
	}

}
