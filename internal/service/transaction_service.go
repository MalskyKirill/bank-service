package service

import (
	"bank-service/internal/apperror"
	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repository"
	"context"
	"errors"
	"net/http"
)

type TransactionService struct {
	transactionRepository *repository.TransactionRepository
}

func NewTransactionService(transactionRepository *repository.TransactionRepository) *TransactionService {
	return &TransactionService{
		transactionRepository: transactionRepository,
	}
}

func (s *TransactionService) GetUserTransactions(ctx context.Context, userId int64, accountId int64) ([]dto.TransactionResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unathorized")
	}

	transactions, err := s.transactionRepository.FindAllByUserId(ctx, userId)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to get transactions")
	}

	return toTransactionResponse(transactions), nil
}

func (s *TransactionService) GetAccountTransactions(ctx context.Context, userId int64, accountId int64) ([]dto.TransactionResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unathorized")
	}

	if accountId <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "invalid account id")
	}

	transactions, err := s.transactionRepository.FindByAccountId(ctx, userId, accountId)
	if err != nil {
		return nil, mapTransactionRepositoryError(err)
	}

	return toTransactionResponse(transactions), nil
}

func mapTransactionRepositoryError(err error) error {
	if errors.Is(err, repository.ErrAcconutNotFound) {
		return apperror.New(http.StatusNotFound, "account not found")
	}

	if errors.Is(err, repository.ErrAccountForbidden) {
		return apperror.New(http.StatusForbidden, "access denied to account")
	}

	return apperror.New(http.StatusInternalServerError, "failed to get transactions")
}

func toTransactionResponse(transactions []models.Transaction) []dto.TransactionResponse {
	response := make([]dto.TransactionResponse, 0, len(transactions))

	for _, transaction := range transactions {
		response = append(response, dto.TransactionResponse{
			ID:            transaction.ID,
			UserID:        transaction.UserID,
			FromAccountID: transaction.FromAccountID,
			ToAccountID:   transaction.ToAccountID,
			Amount:        transaction.Amount,
			Type:          transaction.Type,
			Status:        transaction.Status,
			Description:   transaction.Description,
			CreatedAt:     transaction.CreatedAt,
		})
	}

	return response
}
