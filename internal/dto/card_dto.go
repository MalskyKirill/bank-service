package dto

import "time"

type CreateCardRequest struct {
	AccountID int64 `json:"account_id"`
}

type CreateCardResponse struct {
	ID        int64     `json:"id"`
	AccountId int64     `json:"account_id"`
	Number    string    `json:"number"`
	Expiry    string    `json:"expiry"`
	CVV       string    `json:"cvv"`
	CreatedAt time.Time `json:"created_at"`
}

type CardResponse struct {
	ID        int64     `json:"id"`
	AccountId int64     `json:"account_id"`
	Number    string    `json:"number"`
	Expiry    string    `json:"expiry"`
	CreatedAt time.Time `json:"created_at"`
}

type CardPaymentRequest struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

type CardPaymentResponse struct {
	Message string          `json:"message"`
	CardId  int64           `json:"card_id"`
	Amount  float64         `json:"amount"`
	Account AccountResponse `json:"account"`
}
