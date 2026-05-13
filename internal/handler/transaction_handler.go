package handler

import (
	"bank-service/internal/apperror"
	"bank-service/internal/middleware"
	"bank-service/internal/response"
	"bank-service/internal/service"
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"
)

type TransactionHandler struct {
	transactionService *service.TransactionService
	logger             *logrus.Logger
}

func NewTransactionHandler(transactionService *service.TransactionService, logger *logrus.Logger) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
		logger:             logger,
	}
}

func (h *TransactionHandler) GetUserTransactions(w http.ResponseWriter, r *http.Request) {
	userId, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.transactionService.GetUserTransactions(r.Context(), userId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *TransactionHandler) GetAccountTransactions(w http.ResponseWriter, r *http.Request) {
	userId, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	accountId, err := parseAccountId(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	result, err := h.transactionService.GetAccountTransactions(r.Context(), userId, accountId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *TransactionHandler) handleError(w http.ResponseWriter, err error) {
	var appErr *apperror.AppError

	if errors.As(err, &appErr) {
		response.WriteError(w, appErr.StatusCode, appErr.Message)
		return
	}

	h.logger.Errorf("unexpected transaction error: %v", err)
	response.WriteError(w, http.StatusInternalServerError, "internal server error")
}
