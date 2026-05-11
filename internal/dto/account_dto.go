package dto

import "time"

type CreateAccountRequest struct {
	Currency string `json:"currency"`
}

type AccountResponse struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

type MoneyOperationRequest struct {
	Amount float64 `json:"amount"`
}

type AccountOperationResponse struct {
	Message string          `json:"message"`
	Account AccountResponse `json:"account"`
}

type TransferRequest struct {
	FromAccountID int64   `json:"from_account_id"`
	ToAccountID   int64   `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}

type TransferResponce struct {
	Message        string          `json:"message"`
	FromAccountID  AccountResponse `json:"from_account_id"`
	ToAccountID    int64           `json:"to_account_id"`
	TransferAmount float64         `json:"transfer_amount"`
}
