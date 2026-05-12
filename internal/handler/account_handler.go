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
	"strconv"

	"github.com/gorilla/mux"
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

	if err := decodeJsonBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
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

func (h *AccountHandler) Deposit(w http.ResponseWriter, r *http.Request) {
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

	var req dto.MoneyOperationRequest

	if err := decodeJsonBody(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.accountService.Deposit(r.Context(), userId, accountId, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *AccountHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userId, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	accoundId, err := parseAccountId(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	var req dto.MoneyOperationRequest
	if err := decodeJsonBody(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.accountService.Withdraw(r.Context(), userId, accoundId, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *AccountHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	userId, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.TransferRequest

	if err := decodeJsonBody(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.accountService.Transfer(r.Context(), userId, req)
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

func parseAccountId(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	accountIdRaws := vars["accountId"]
	if accountIdRaws == "" {
		return 0, errors.New("account id is empty")
	}

	accountId, err := strconv.ParseInt(accountIdRaws, 10, 64)
	if err != nil {
		return 0, err
	}

	if accountId <= 0 {
		return 0, errors.New("account id must be positive")
	}

	return accountId, nil
}

func decodeJsonBody(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	return decoder.Decode(target)
}
