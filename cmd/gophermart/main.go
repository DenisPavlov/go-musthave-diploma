package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DenisPavlov/go-musthave-diploma/internal/client/accrual"
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
	"github.com/DenisPavlov/go-musthave-diploma/internal/service/order"
	ordersStorage "github.com/DenisPavlov/go-musthave-diploma/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"golang.org/x/sync/errgroup"
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

	router.Get("/api/orders/{val}", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.Header().Set("Retry-After", "10")
		render.Status(r, http.StatusTooManyRequests)
		render.PlainText(w, r, "hello")
	})

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

	accrualClient := accrual.NewClient(log, cfg)
	orderProcessor := order.NewProcessor(log, accrualClient, storage)

	appCtx, cancelApp := context.WithCancel(context.Background())

	g, gCtx := errgroup.WithContext(appCtx)
	g.Go(func() error {
		return orderProcessor.ProcessNewOrders(gCtx, 10, 5)
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
	g.Go(func() error {
		return srv.ListenAndServe()
	})

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	g.Go(func() error {
		select {
		case sig := <-done:
			defer cancelApp()
			log.Info("received shutdown signal", slog.String("signal", sig.String()))
			accrualClient.Shutdown()
			if err := storage.Close(); err != nil {
				log.Error("error closing storage", logger.Err(err))
			}
			if err := srv.Shutdown(appCtx); err != nil {
				log.Error("failed to stop server", logger.Err(err))
			}
			log.Info("server stopped")
			return nil
		case <-gCtx.Done():
			return nil
		}
	})

	if err := g.Wait(); err != nil {
		log.Error("error running service", logger.Err(err))
		os.Exit(1)
	}
}
