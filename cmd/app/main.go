package main

import (
	"bank-service/internal/config"
	"bank-service/internal/db"
	"bank-service/internal/router"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	cfg, err := config.Load()

	if err != nil {
		logger.Fatalf("failed to load config: %v", err)
	}

	database, err := db.OpenPostgres(cfg)
	if err != nil {
		logger.Fatalf("failed to connect to database: %v", err)
	}

	defer database.Close()

	logger.Info("connected to PostgreSQL")

	if err := db.InitSchema(database, "migrations/schema.sql"); err != nil {
		logger.Fatalf("failed to initialize database schema: %v", err)
	}

	logger.Info("database schema initialized")

	appRouter := router.NewRouter(database, cfg, logger)

	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      appRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server started on port %s", cfg.ServerPort)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit

	logger.Info("server stopped")

}
