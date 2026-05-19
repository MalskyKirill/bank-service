package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"bank-service/internal/apperror"
	"bank-service/internal/dto"
	"bank-service/internal/repository"
)

type AnalyticsService struct {
	analyticsRepository *repository.AnalyticsRepository
}

func NewAnalyticsService(analyticsRepository *repository.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepository: analyticsRepository,
	}
}

func (s *AnalyticsService) GetMonthlyAnalytics(
	ctx context.Context,
	userID int64,
	month string,
) (*dto.MonthlyAnalyticsResponse, error) {
	if userID <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	monthStart, monthEnd, normalizedMonth, err := parseAnalyticsMonth(month)
	if err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid month format, use YYYY-MM")
	}

	analytics, err := s.analyticsRepository.GetMonthlyAnalytics(ctx, userID, monthStart, monthEnd)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to get analytics")
	}

	return &dto.MonthlyAnalyticsResponse{
		Month:             normalizedMonth,
		Income:            analytics.Income,
		Expenses:          analytics.Expenses,
		Net:               roundMoney(analytics.Income - analytics.Expenses),
		TransactionsCount: analytics.TransactionsCount,

		Deposits:       analytics.Deposits,
		Withdrawals:    analytics.Withdrawals,
		TransfersIn:    analytics.TransfersIn,
		TransfersOut:   analytics.TransfersOut,
		CardPayments:   analytics.CardPayments,
		CreditIssues:   analytics.CreditIssues,
		CreditPayments: analytics.CreditPayments,
		Penalties:      analytics.Penalties,
		CreditLoad:     analytics.CreditLoad,
	}, nil
}

func (s *AnalyticsService) PredictBalance(
	ctx context.Context,
	userID int64,
	accountID int64,
	days int,
) (*dto.BalancePredictionResponse, error) {
	if userID <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	if accountID <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "invalid account id")
	}

	if days <= 0 {
		days = 30
	}

	if days > 365 {
		return nil, apperror.New(http.StatusBadRequest, "days must not be greater than 365")
	}

	prediction, err := s.analyticsRepository.PredictBalance(ctx, userID, accountID, days)
	if err != nil {
		return nil, mapAnalyticsRepositoryError(err)
	}

	return &dto.BalancePredictionResponse{
		AccountID:        prediction.AccountID,
		Days:             days,
		CurrentBalance:   prediction.CurrentBalance,
		PlannedPayments:  prediction.PlannedPayments,
		PredictedBalance: prediction.PredictedBalance,
		Currency:         prediction.Currency,
	}, nil
}

func parseAnalyticsMonth(month string) (time.Time, time.Time, string, error) {
	if month == "" {
		now := time.Now()

		monthStart := time.Date(
			now.Year(),
			now.Month(),
			1,
			0,
			0,
			0,
			0,
			time.UTC,
		)

		monthEnd := monthStart.AddDate(0, 1, 0)

		return monthStart, monthEnd, monthStart.Format("2006-01"), nil
	}

	parsed, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}

	monthStart := time.Date(
		parsed.Year(),
		parsed.Month(),
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	monthEnd := monthStart.AddDate(0, 1, 0)

	return monthStart, monthEnd, monthStart.Format("2006-01"), nil
}

func mapAnalyticsRepositoryError(err error) error {
	if errors.Is(err, repository.ErrAcconutNotFound) {
		return apperror.New(http.StatusNotFound, "account not found")
	}

	if errors.Is(err, repository.ErrAccountForbidden) {
		return apperror.New(http.StatusForbidden, "access denied to account")
	}

	return apperror.New(http.StatusInternalServerError, "failed to build balance prediction")
}
