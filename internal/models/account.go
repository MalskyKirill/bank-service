package models

import "time"

type Account struct {
	ID        int64
	UserID    int64
	Balance   float64
	Currency  string
	CreatedAt time.Time
}
