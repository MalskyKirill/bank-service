package models

import "time"

type Transaction struct {
	ID            int64
	UserID        int64
	FromAccountID *int64
	ToAccountID   *int64
	Amount        float64
	Type          string
	Status        string
	Description   *string
	CreatedAt     time.Time
}
