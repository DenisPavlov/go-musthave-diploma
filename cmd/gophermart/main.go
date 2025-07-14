package main

import (
	"github.com/DenisPavlov/go-musthave-diploma/internal/config"
	"github.com/DenisPavlov/go-musthave-diploma/internal/logger"
	"log/slog"
)

func main() {

	// init config
	cfg := config.MustLoad()

	// init logger
	log := logger.SetupLogger(cfg.Env)
	log.Info("starting gophermart", slog.Any("config", cfg))

}
