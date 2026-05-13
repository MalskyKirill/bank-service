package models

import "time"

type Card struct {
	ID         int64
	UserID     int64
	AccountId  int64
	Number     string
	Expiry     string
	CVVHash    string
	NumberHMAC string
	CreatedAt  time.Time
}
