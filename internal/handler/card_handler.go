package handler

import (
	"bank-service/internal/apperror"
	"bank-service/internal/dto"
	"bank-service/internal/middleware"
	"bank-service/internal/response"
	"bank-service/internal/service"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type CardHandler struct {
	cardService *service.CardService
	logger      *logrus.Logger
}

func NewCardHandler(cardService *service.CardService, logger *logrus.Logger) *CardHandler {
	return &CardHandler{
		cardService: cardService,
		logger:      logger,
	}
}

func (h *CardHandler) CreateCard(w http.ResponseWriter, r *http.Request) {
	userId, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.CreateCardRequest

	if err := decodeJsonBody(r, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.cardService.CreateCard(r.Context(), userId, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, result)
}

func (h *CardHandler) GetCards(w http.ResponseWriter, req *http.Request) {
	userId, ok := middleware.UserIDFromContext(req.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.cardService.GetUserCards(req.Context(), userId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *CardHandler) GetCard(w http.ResponseWriter, req *http.Request) {
	userId, ok := middleware.UserIDFromContext(req.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cardId, err := parseCardId(req)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	result, err := h.cardService.GetCard(req.Context(), userId, cardId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *CardHandler) Pay(w http.ResponseWriter, req *http.Request) {
	userId, ok := middleware.UserIDFromContext(req.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cardId, err := parseCardId(req)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid card id")
		return
	}

	var request dto.CardPaymentRequest
	if err := decodeJsonBody(req, &request); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.cardService.Pay(req.Context(), userId, cardId, request)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *CardHandler) handleError(w http.ResponseWriter, err error) {
	var appErr *apperror.AppError

	if errors.As(err, &appErr) {
		response.WriteError(w, appErr.StatusCode, appErr.Message)
		return
	}

	h.logger.Errorf("unenspected card error, %v", err)
	response.WriteError(w, http.StatusInternalServerError, "internal server error")
}

func parseCardId(r *http.Request) (int64, error) {
	vars := mux.Vars(r)

	cardIdRaw := vars["cardId"]

	if cardIdRaw == "" {
		return 0, errors.New("card id is empty")
	}

	cardId, err := strconv.ParseInt(cardIdRaw, 10, 64)
	if err != nil {
		return 0, err
	}

	if cardId <= 0 {
		return 0, errors.New("card id must be positive")
	}

	return cardId, nil
}
