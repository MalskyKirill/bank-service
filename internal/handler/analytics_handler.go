package handler

import (
	"errors"
	"net/http"
	"strconv"

	"bank-service/internal/apperror"
	"bank-service/internal/middleware"
	"bank-service/internal/response"
	"bank-service/internal/service"

	"github.com/sirupsen/logrus"
)

type AnalyticsHandler struct {
	analyticsService *service.AnalyticsService
	logger           *logrus.Logger
}

func NewAnalyticsHandler(
	analyticsService *service.AnalyticsService,
	logger *logrus.Logger,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
		logger:           logger,
	}
}

func (h *AnalyticsHandler) GetMonthlyAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	month := r.URL.Query().Get("month")

	result, err := h.analyticsService.GetMonthlyAnalytics(r.Context(), userID, month)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *AnalyticsHandler) PredictBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	accountID, err := parseAccountId(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	days := 30

	daysRaw := r.URL.Query().Get("days")
	if daysRaw != "" {
		parsedDays, err := strconv.Atoi(daysRaw)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, "invalid days")
			return
		}

		days = parsedDays
	}

	result, err := h.analyticsService.PredictBalance(r.Context(), userID, accountID, days)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}

func (h *AnalyticsHandler) handleError(w http.ResponseWriter, err error) {
	var appErr *apperror.AppError

	if errors.As(err, &appErr) {
		response.WriteError(w, appErr.StatusCode, appErr.Message)
		return
	}

	h.logger.Errorf("unexpected analytics error: %v", err)
	response.WriteError(w, http.StatusInternalServerError, "internal server error")
}
