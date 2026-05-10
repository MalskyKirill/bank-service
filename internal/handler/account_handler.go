package handler

import (
	"bank-service/internal/apperror"
	"bank-service/internal/dto"
	"bank-service/internal/middleware"
	"bank-service/internal/response"
	"bank-service/internal/service"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

type AccountHandler struct {
	accountService *service.AccountService
	logger         *logrus.Logger
}

func NewAccountHandler(accountService *service.AccountService, logger *logrus.Logger) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
		logger:         logger,
	}
}

func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.CreateAccountRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&req)
	if err != nil && !errors.Is(err, io.EOF) {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.accountService.CreateAccount(r.Context(), userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, result)
}

func (h *AccountHandler) GetAccounts(w http.ResponseWriter, r *http.Request) {
	userId, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.accountService.GetUserAccount(r.Context(), userId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *AccountHandler) handleError(w http.ResponseWriter, err error) {
	var appError *apperror.AppError

	if errors.As(err, &appError) {
		response.WriteError(w, appError.StatusCode, appError.Message)
		return
	}

	h.logger.Errorf("unexpected account error: %v", err)

	response.WriteError(w, http.StatusInternalServerError, "internal server error")
}
