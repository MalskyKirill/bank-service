package service

import (
	"bank-service/internal/apperror"
	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repository"
	"context"
	"net/http"
	"strings"
)

type AccountService struct {
	accountRepository *repository.AccountRepository
}

func NewAccountService(accountRepository *repository.AccountRepository) *AccountService {
	return &AccountService{
		accountRepository: accountRepository,
	}
}

func (s *AccountService) CreateAccount(ctx context.Context, userID int64, req dto.CreateAccountRequest) (*dto.AccountResponse, error) {
	if userID <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "RUB"
	}

	if currency != "RUB" {
		return nil, apperror.New(http.StatusBadRequest, "only RUB currency is supported")
	}

	account, err := s.accountRepository.Create(ctx, userID, currency)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, err.Error())
	}

	return toAccountResponce(account), nil
}

func (s *AccountService) GetUserAccount(ctx context.Context, userID int64) ([]dto.AccountResponse, error) {
	if userID <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	accounts, err := s.accountRepository.FindAllByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to get accounts")
	}

	response := make([]dto.AccountResponse, 0, len(accounts))

	for _, account := range accounts {
		accountCopy := account
		response = append(response, *toAccountResponce(&accountCopy))
	}

	return response, nil

}

func toAccountResponce(account *models.Account) *dto.AccountResponse {
	return &dto.AccountResponse{
		ID:        account.ID,
		UserID:    account.UserID,
		Balance:   account.Balance,
		Currency:  account.Currency,
		CreatedAt: account.CreatedAt,
	}
}
