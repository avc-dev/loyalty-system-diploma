package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const (
	serverReadTimeout  = 15 * time.Second
	serverWriteTimeout = 15 * time.Second
	serverIdleTimeout  = 60 * time.Second
	shutdownTimeout    = 10 * time.Second
)

// createServer создает HTTP сервер
func createServer(addr string, handler *chi.Mux) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}
}

// runServer запускает HTTP сервер и ожидает сигнала завершения
func (a *App) runServer(ctx context.Context) error {
	// Запуск HTTP сервера в горутине
	go func() {
		a.logger.Info("starting HTTP server", zap.String("address", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	return nil
}

// shutdown выполняет graceful shutdown приложения
func (a *App) shutdown(cancel context.CancelFunc) {
	a.logger.Info("shutting down server...")

	// Останавливаем прием новых запросов
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("server shutdown error", zap.Error(err))
	}

	// Останавливаем worker pool
	cancel()
	a.workerPool.Stop()
	a.logger.Info("worker pool stopped")

	// Закрываем соединение с БД
	a.db.Close()
	a.logger.Info("database connection closed")

	a.logger.Info("server stopped gracefully")
}
