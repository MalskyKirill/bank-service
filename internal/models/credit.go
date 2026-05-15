package models

import "time"

type Credit struct {
	ID             int64
	UserID         int64
	AccountID      int64
	Amount         float64
	InterestRate   float64
	TermMonths     int
	MonthlyPayment float64
	Status         string
	CreatedAt      time.Time
}

type PaymentSchedule struct {
	ID              int64
	CreditID        int64
	PaymentDate     time.Time
	Amount          float64
	PrincipalAmount float64
	InterestAmount  float64
	PenaltyAmount   float64
	Status          string
	PaidAt          *time.Time
}
