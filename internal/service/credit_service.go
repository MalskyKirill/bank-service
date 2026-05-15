package service

import (
	"context"
	"errors"
	"math"
	"net/http"
	"time"

	"bank-service/internal/apperror"
	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repository"
)

type CreditRateProvider interface {
	GetCreditRate(ctx context.Context) (float64, error)
}

type CreditService struct {
	creditRepository *repository.CreditRepository
	rateProvider     CreditRateProvider
}

func NewCreditService(
	creditRepository *repository.CreditRepository,
	rateProvider CreditRateProvider,
) *CreditService {
	return &CreditService{
		creditRepository: creditRepository,
		rateProvider:     rateProvider,
	}
}

func (s *CreditService) CreateCredit(ctx context.Context, userId int64, req dto.CreateCreditRequest) (*dto.CreateCreditResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	if req.AccountID <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "account_id is required")
	}

	if req.Amount <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "amount must be greater than zero")
	}

	if req.TermMonths <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "term_months must be greater than zero")
	}

	if req.TermMonths > 120 {
		return nil, apperror.New(http.StatusBadRequest, "term_months must not be greater than 120")
	}

	interestRate, err := s.rateProvider.GetCreditRate(ctx)
	if err != nil {
		return nil, apperror.New(http.StatusServiceUnavailable, "failed to get central bank rate")
	}

	monthlyPayment := calculateAnnuityPayment(req.Amount, interestRate, req.TermMonths)

	schedule, err := generatePaymentSchedule(req.Amount, interestRate, req.TermMonths, monthlyPayment)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to generate payment schedule")
	}

	credit, createdSchedule, err := s.creditRepository.Create(
		ctx,
		userId,
		req.AccountID,
		req.Amount,
		interestRate,
		req.TermMonths,
		monthlyPayment,
		schedule,
	)
	if err != nil {
		return nil, mapCreditRepositoryError(err, "failed to create credit")
	}

	return &dto.CreateCreditResponse{
		Credit:   *toCreditResponse(credit),
		Schedule: toPaymentScheduleResponses(createdSchedule),
	}, nil
}

func (s *CreditService) GetCredits(ctx context.Context, userId int64) ([]dto.CreditResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	credits, err := s.creditRepository.FindAllByUserID(ctx, userId)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to get credits")
	}

	response := make([]dto.CreditResponse, 0, len(credits))

	for _, credit := range credits {
		creditCopy := credit
		response = append(response, *toCreditResponse(&creditCopy))
	}

	return response, nil
}

func (s *CreditService) GetCredit(ctx context.Context, userId int64, creditId int64) (*dto.CreditResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	if creditId <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "invalid credit id")
	}

	credit, err := s.creditRepository.FindByID(ctx, userId, creditId)
	if err != nil {
		return nil, mapCreditRepositoryError(err, "failed to get credit")
	}

	return toCreditResponse(credit), nil
}

func (s *CreditService) GetCreditSchedule(ctx context.Context, userId int64, creditId int64) ([]dto.PaymentScheduleResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	if creditId <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "invalid credit id")
	}

	schedule, err := s.creditRepository.FindScheduleByCreditID(ctx, userId, creditId)
	if err != nil {
		return nil, mapCreditRepositoryError(err, "failed to get credit schedule")
	}

	return toPaymentScheduleResponses(schedule), nil
}

func calculateAnnuityPayment(amount float64, annualRate float64, termMonths int) float64 {
	monthlyRate := annualRate / 12 / 100

	if monthlyRate == 0 {
		return roundMoney(amount / float64(termMonths))
	}

	pow := math.Pow(1+monthlyRate, float64(termMonths))

	payment := amount * monthlyRate * pow / (pow - 1)

	return roundMoney(payment)
}

func generatePaymentSchedule(amount float64, annualRate float64, termMonths int, monthlyPayment float64) ([]models.PaymentSchedule, error) {
	monthlyRate := annualRate / 12 / 100
	remainingDebt := amount

	schedule := make([]models.PaymentSchedule, 0, termMonths)
	now := time.Now()

	for month := 1; month <= termMonths; month++ {
		interestAmount := roundMoney(remainingDebt * monthlyRate)
		principalAmount := roundMoney(monthlyPayment - interestAmount)
		paymentAmount := monthlyPayment

		if month == termMonths {
			principalAmount = roundMoney(remainingDebt)
			paymentAmount = roundMoney(principalAmount + interestAmount)
		}

		if principalAmount < 0 {
			return nil, errors.New("principal amount is negative")
		}

		remainingDebt = roundMoney(remainingDebt - principalAmount)

		schedule = append(schedule, models.PaymentSchedule{
			PaymentDate:     now.AddDate(0, month, 0),
			Amount:          paymentAmount,
			PrincipalAmount: principalAmount,
			InterestAmount:  interestAmount,
			PenaltyAmount:   0,
			Status:          "PLANNED",
		})
	}

	return schedule, nil
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func mapCreditRepositoryError(err error, defaultMessage string) error {
	if errors.Is(err, repository.ErrAcconutNotFound) {
		return apperror.New(http.StatusNotFound, "account not found")
	}

	if errors.Is(err, repository.ErrAccountForbidden) {
		return apperror.New(http.StatusForbidden, "access denied to account")
	}

	if errors.Is(err, repository.ErrCreditNotFound) {
		return apperror.New(http.StatusNotFound, "credit not found")
	}

	if errors.Is(err, repository.ErrCreditForbidden) {
		return apperror.New(http.StatusForbidden, "access denied to credit")
	}

	return apperror.New(http.StatusInternalServerError, defaultMessage)
}

func toCreditResponse(credit *models.Credit) *dto.CreditResponse {
	return &dto.CreditResponse{
		ID:             credit.ID,
		UserID:         credit.UserID,
		AccountID:      credit.AccountID,
		Amount:         credit.Amount,
		InterestRate:   credit.InterestRate,
		TermMonths:     credit.TermMonths,
		MonthlyPayment: credit.MonthlyPayment,
		Status:         credit.Status,
		CreatedAt:      credit.CreatedAt,
	}
}

func toPaymentScheduleResponses(schedule []models.PaymentSchedule) []dto.PaymentScheduleResponse {
	response := make([]dto.PaymentScheduleResponse, 0, len(schedule))

	for _, payment := range schedule {
		response = append(response, dto.PaymentScheduleResponse{
			ID:              payment.ID,
			CreditID:        payment.CreditID,
			PaymentDate:     payment.PaymentDate,
			Amount:          payment.Amount,
			PrincipalAmount: payment.PrincipalAmount,
			InterestAmount:  payment.InterestAmount,
			PenaltyAmount:   payment.PenaltyAmount,
			Status:          payment.Status,
			PaidAt:          payment.PaidAt,
		})
	}

	return response
}
