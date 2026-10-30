package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TI-Wegen/api-pdf-go/internal/browser"
	"github.com/TI-Wegen/api-pdf-go/internal/config"
	"github.com/TI-Wegen/api-pdf-go/internal/converter"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	pool := browser.NewPool(cfg.MaxConcurrentPages, cfg.RecycleAfter, cfg.QueueTimeout, cfg.ChromePath, logger)
	converterHandler := converter.NewHandler(pool, cfg, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /CreatePdfFromUrl", converterHandler.HandleConvert)
	mux.HandleFunc("POST /CreatePdfFromHtml", converterHandler.HandleConvertHTML)
	mux.HandleFunc("GET /health", converterHandler.HandleHealth)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second, // HTML grande pode levar mais que 5s para subir
		WriteTimeout:      cfg.QueueTimeout + cfg.RenderTimeout + 15*time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("servidor escutando",
			"endereço", server.Addr,
			"paginas_simultaneas", cfg.MaxConcurrentPages,
			"reciclar_apos", cfg.RecycleAfter,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("servidor encontrou um erro fatal", "erro", err)
			os.Exit(1)
		}
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	<-shutdownChan
	logger.Info("sinal de desligamento recebido, iniciando graceful shutdown...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.RenderTimeout+15*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("falha no graceful shutdown", "erro", err)
	} else {
		logger.Info("servidor desligado com sucesso")
	}

	pool.Close()
}
