package router

import (
	"bank-service/internal/config"
	"bank-service/internal/handler"
	"bank-service/internal/middleware"
	"bank-service/internal/repository"
	"bank-service/internal/security"
	"bank-service/internal/service"
	"database/sql"
	"net/http"

	cbrintegration "bank-service/internal/integration/cbr"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	smtpintegration "bank-service/internal/integration/smtp"
)

func NewRouter(database *sql.DB, cfg *config.Config, logger *logrus.Logger) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/health", handler.Health).Methods(http.MethodGet)

	userRepository := repository.NewUserRepository(database)
	jwtService := security.NewJWTService(cfg.JWTSecret, cfg.JWTTTLHours)
	authService := service.NewAuthService(userRepository, jwtService)
	authHandler := handler.NewAuthHandler(authService, logger)

	smtpClient := smtpintegration.NewClient(
		cfg.SMTPEnabled,
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPass,
		cfg.SMTPFrom,
		logger,
	)

	notificationService := service.NewNotificationService(smtpClient)

	r.HandleFunc("/register", authHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)

	accountRepository := repository.NewAccountRepository(database)
	accountService := service.NewAccountService(accountRepository)
	accountHandler := handler.NewAccountHandler(accountService, logger)

	transactionRepository := repository.NewTransactionRepository(database)
	transactionService := service.NewTransactionService(transactionRepository)
	transactionHandler := handler.NewTransactionHandler(transactionService, logger)

	cardRepository := repository.NewCardRepository(database, cfg.PGPSecret)
	cardService := service.NewCardService(
		cardRepository,
		userRepository,
		notificationService,
		cfg.HMACSecret,
		logger,
	)
	cardHandler := handler.NewCardHandler(cardService, logger)

	creditRepository := repository.NewCreditRepository(database)

	cbrClient := cbrintegration.NewClient(
		cfg.CBRURL,
		cfg.CBRRateMargin,
		cfg.CBRLookbackDays,
	)

	creditService := service.NewCreditService(
		creditRepository,
		cbrClient,
		userRepository,
		notificationService,
		logger,
	)
	creditHandler := handler.NewCreditHandler(creditService, logger)

	authRouter := r.PathPrefix("/").Subrouter()
	authRouter.Use(middleware.AuthMiddleware(jwtService))

	authRouter.HandleFunc("/accounts", accountHandler.CreateAccount).Methods(http.MethodPost)
	authRouter.HandleFunc("/accounts", accountHandler.GetAccounts).Methods(http.MethodGet)

	authRouter.HandleFunc("/accounts/{accountId:[0-9]+}/deposit", accountHandler.Deposit).Methods(http.MethodPost)
	authRouter.HandleFunc("/accounts/{accountId:[0-9]+}/withdraw", accountHandler.Withdraw).Methods(http.MethodPost)

	authRouter.HandleFunc("/transfer", accountHandler.Transfer).Methods(http.MethodPost)

	authRouter.HandleFunc("/transactions", transactionHandler.GetUserTransactions).Methods(http.MethodGet)
	authRouter.HandleFunc("/accounts/{accountId:[0-9]+}/transactions", transactionHandler.GetAccountTransactions).Methods(http.MethodGet)

	authRouter.HandleFunc("/cards", cardHandler.CreateCard).Methods(http.MethodPost)
	authRouter.HandleFunc("/cards", cardHandler.GetCards).Methods(http.MethodGet)
	authRouter.HandleFunc("/cards/{cardId:[0-9]+}", cardHandler.GetCard).Methods(http.MethodGet)
	authRouter.HandleFunc("/cards/{cardId:[0-9]+}/pay", cardHandler.Pay).Methods(http.MethodPost)

	authRouter.HandleFunc("/credits", creditHandler.CreateCredit).Methods(http.MethodPost)
	authRouter.HandleFunc("/credits", creditHandler.GetCredits).Methods(http.MethodGet)
	authRouter.HandleFunc("/credits/{creditId:[0-9]+}", creditHandler.GetCredit).Methods(http.MethodGet)
	authRouter.HandleFunc("/credits/{creditId:[0-9]+}/schedule", creditHandler.GetCreditSchedule).Methods(http.MethodGet)

	return r
}
