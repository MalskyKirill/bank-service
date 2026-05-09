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
