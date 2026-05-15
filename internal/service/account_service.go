package service

import (
	"bank-service/internal/apperror"
	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repository"
	"context"
	"errors"
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

	return toAccountResponse(account), nil
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
		response = append(response, *toAccountResponse(&accountCopy))
	}

	return response, nil

}

func (s *AccountService) Deposit(ctx context.Context, userId int64, accountId int64, req dto.MoneyOperationRequest) (*dto.AccountOperationResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	if accountId <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "invalid account id")
	}

	if req.Amount <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "amount mast be more 0")
	}

	account, err := s.accountRepository.Deposit(ctx, userId, accountId, req.Amount)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, err.Error())
	}

	return &dto.AccountOperationResponse{
		Message: "deposit completed successfully",
		Account: *toAccountResponse(account),
	}, nil
}

func (s *AccountService) Withdraw(ctx context.Context, userId int64, accountId int64, req dto.MoneyOperationRequest) (*dto.AccountOperationResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "invalid account id")
	}

	if accountId <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "invalid account id")
	}

	if req.Amount <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "amount mast be more 0")
	}

	account, err := s.accountRepository.Withdraw(ctx, userId, accountId, req.Amount)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, err.Error())
	}

	return &dto.AccountOperationResponse{
		Message: "withdraw completed successfully",
		Account: *toAccountResponse(account),
	}, nil
}

func (s *AccountService) Transfer(ctx context.Context, userId int64, req dto.TransferRequest) (*dto.TransferResponce, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "invalid account id")
	}

	if req.FromAccountID <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "from_account_id is required")
	}

	if req.ToAccountID <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "to_account_id is required")
	}

	if req.Amount <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "amount mast be more 0")
	}

	account, err := s.accountRepository.Transfer(ctx, userId, req.FromAccountID, req.ToAccountID, req.Amount)
	if err != nil {
		return nil, mapAccountRepositotyError(err, "failed to transfer money")
	}

	return &dto.TransferResponce{
		Message:        "transfer completed successfully",
		FromAccountID:  *toAccountResponse(account),
		ToAccountID:    req.ToAccountID,
		TransferAmount: req.Amount,
	}, nil
}

func mapAccountRepositotyError(err error, defaultMessage string) error {
	if errors.Is(err, repository.ErrAcconutNotFound) {
		return apperror.New(http.StatusNotFound, "account not found")
	}

	if errors.Is(err, repository.ErrAccountForbidden) {
		return apperror.New(http.StatusForbidden, "access denied to account")
	}

	if errors.Is(err, repository.ErrInsufficientFunds) {
		return apperror.New(http.StatusBadRequest, "insufficient funds")
	}

	if errors.Is(err, repository.ErrSameAccountTransfer) {
		return apperror.New(http.StatusBadRequest, "cannot transfer to the same account")
	}

	return apperror.New(http.StatusInternalServerError, defaultMessage)
}

func toAccountResponse(account *models.Account) *dto.AccountResponse {
	return &dto.AccountResponse{
		ID:        account.ID,
		UserID:    account.UserID,
		Balance:   account.Balance,
		Currency:  account.Currency,
		CreatedAt: account.CreatedAt,
	}
}
