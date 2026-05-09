package router

import (
	"bank-service/internal/config"
	"bank-service/internal/handler"
	"bank-service/internal/repository"
	"bank-service/internal/security"
	"bank-service/internal/service"
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func NewRouter(database *sql.DB, cfg *config.Config, logger *logrus.Logger) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/health", handler.Health).Methods(http.MethodGet)

	userRepository := repository.NewUserRepository(database)
	jwtService := security.NewJWTService(cfg.JWTSecret, cfg.JWTTTLHours)
	authService := service.NewAuthService(userRepository, jwtService)
	authHandler := handler.NewAuthHandler(authService, logger)

	r.HandleFunc("/register", authHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)

	return r
}
