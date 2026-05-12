package dto

import "time"

type TransactionResponse struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	FromAccountID *int64    `json:"from_account_id"`
	ToAccountID   *int64    `json:"to_account_id"`
	Amount        float64   `json:"amount"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	Discription   *string   `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}
