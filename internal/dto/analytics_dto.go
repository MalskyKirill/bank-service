package dto

type MonthlyAnalyticsResponse struct {
	Month             string  `json:"month"`
	Income            float64 `json:"income"`
	Expenses          float64 `json:"expenses"`
	Net               float64 `json:"net"`
	TransactionsCount int     `json:"transactions_count"`

	Deposits       float64 `json:"deposits"`
	Withdrawals    float64 `json:"withdrawals"`
	TransfersIn    float64 `json:"transfers_in"`
	TransfersOut   float64 `json:"transfers_out"`
	CardPayments   float64 `json:"card_payments"`
	CreditIssues   float64 `json:"credit_issues"`
	CreditPayments float64 `json:"credit_payments"`
	Penalties      float64 `json:"penalties"`
	CreditLoad     float64 `json:"credit_load"`
}

type BalancePredictionResponse struct {
	AccountID        int64   `json:"account_id"`
	Days             int     `json:"days"`
	CurrentBalance   float64 `json:"current_balance"`
	PlannedPayments  float64 `json:"planned_payments"`
	PredictedBalance float64 `json:"predicted_balance"`
	Currency         string  `json:"currency"`
}
