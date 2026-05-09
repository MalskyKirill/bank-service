package handler

import (
	"bank-service/internal/apperror"
	"bank-service/internal/dto"
	"bank-service/internal/response"
	"bank-service/internal/service"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"
)

type AuthHandler struct {
	authService *service.AuthService
	logger      *logrus.Logger
}

func NewAuthHandler(authService *service.AuthService, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.authService.Registration(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, result)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.authService.Login(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) handleError(w http.ResponseWriter, err error) {
	var appErr *apperror.AppError

	if errors.As(err, &appErr) {
		response.WriteError(w, appErr.StatusCode, appErr.Message)
		return
	}

	h.logger.Errorf("unexpected error: %v", err)

	response.WriteError(w, http.StatusInternalServerError, "internal server error")
}
