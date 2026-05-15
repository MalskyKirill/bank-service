package dto

import "time"

type CreateCreditRequest struct {
	AccountID  int64   `json:"account_id"`
	Amount     float64 `json:"amount"`
	TermMonths int     `json:"term_months"`
}

type CreditResponse struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	AccountID      int64     `json:"account_id"`
	Amount         float64   `json:"amount"`
	InterestRate   float64   `json:"interest_rate"`
	TermMonths     int       `json:"term_months"`
	MonthlyPayment float64   `json:"monthly_payment"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type PaymentScheduleResponse struct {
	ID              int64      `json:"id"`
	CreditID        int64      `json:"credit_id"`
	PaymentDate     time.Time  `json:"payment_date"`
	Amount          float64    `json:"amount"`
	PrincipalAmount float64    `json:"principal_amount"`
	InterestAmount  float64    `json:"interest_amount"`
	PenaltyAmount   float64    `json:"penalty_amount"`
	Status          string     `json:"status"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
}

type CreateCreditResponse struct {
	Credit   CreditResponse            `json:"credit"`
	Schedule []PaymentScheduleResponse `json:"shedule"`
}
