package handler

import (
	"errors"
	"net/http"
	"strconv"

	"bank-service/internal/apperror"
	"bank-service/internal/dto"
	"bank-service/internal/middleware"
	"bank-service/internal/response"
	"bank-service/internal/service"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type CreditHandler struct {
	creditService *service.CreditService
	logger        *logrus.Logger
}

func NewCreditHandler(creditService *service.CreditService, logger *logrus.Logger) *CreditHandler {
	return &CreditHandler{
		creditService: creditService,
		logger:        logger,
	}
}

func (h *CreditHandler) CreateCredit(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.CreateCreditRequest

	if err := decodeJsonBody(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.creditService.CreateCredit(r.Context(), userID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, result)
}

func (h *CreditHandler) GetCredits(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.creditService.GetCredits(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *CreditHandler) GetCredit(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	creditID, err := parseCreditID(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid credit id")
		return
	}

	result, err := h.creditService.GetCredit(r.Context(), userID, creditID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *CreditHandler) GetCreditSchedule(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	creditID, err := parseCreditID(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid credit id")
		return
	}

	result, err := h.creditService.GetCreditSchedule(r.Context(), userID, creditID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *CreditHandler) handleError(w http.ResponseWriter, err error) {
	var appErr *apperror.AppError

	if errors.As(err, &appErr) {
		response.WriteError(w, appErr.StatusCode, appErr.Message)
		return
	}

	h.logger.Errorf("unexpected credit error: %v", err)
	response.WriteError(w, http.StatusInternalServerError, "internal server error")
}

func parseCreditID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)

	creditIDRaw := vars["creditId"]
	if creditIDRaw == "" {
		return 0, errors.New("credit id is empty")
	}

	creditID, err := strconv.ParseInt(creditIDRaw, 10, 64)
	if err != nil {
		return 0, err
	}

	if creditID <= 0 {
		return 0, errors.New("credit id must be positive")
	}

	return creditID, nil
}
