package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"
)

type MonthlyAnalytics struct {
	Income            float64
	Expenses          float64
	TransactionsCount int

	Deposits       float64
	Withdrawals    float64
	TransfersIn    float64
	TransfersOut   float64
	CardPayments   float64
	CreditIssues   float64
	CreditPayments float64
	Penalties      float64
	CreditLoad     float64
}

type BalancePrediction struct {
	AccountID        int64
	CurrentBalance   float64
	PlannedPayments  float64
	PredictedBalance float64
	Currency         string
}

type AnalyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{
		db: db,
	}
}

func (r *AnalyticsRepository) GetMonthlyAnalytics(
	ctx context.Context,
	userID int64,
	monthStart time.Time,
	monthEnd time.Time,
) (*MonthlyAnalytics, error) {
	query := `
		WITH user_accounts AS (
			SELECT id
			FROM accounts
			WHERE user_id = $1
		),
		month_transactions AS (
			SELECT t.*
			FROM transactions t
			WHERE t.created_at >= $2
			  AND t.created_at < $3
			  AND (
			  	t.user_id = $1
			  	OR t.from_account_id IN (SELECT id FROM user_accounts)
			  	OR t.to_account_id IN (SELECT id FROM user_accounts)
			  )
		)
		SELECT
			COALESCE(SUM(CASE
				WHEN t.type = 'DEPOSIT'
					AND t.to_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				WHEN t.type = 'CREDIT_ISSUE'
					AND t.to_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				WHEN t.type = 'TRANSFER'
					AND t.to_account_id IN (SELECT id FROM user_accounts)
					AND NOT EXISTS (
						SELECT 1 FROM user_accounts ua WHERE ua.id = t.from_account_id
					)
					THEN t.amount
				ELSE 0
			END), 0) AS income,

			COALESCE(SUM(CASE
				WHEN t.type = 'WITHDRAW'
					AND t.from_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				WHEN t.type = 'CARD_PAYMENT'
					AND t.from_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				WHEN t.type = 'CREDIT_PAYMENT'
					AND t.from_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				WHEN t.type = 'PENALTY'
					AND t.from_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				WHEN t.type = 'TRANSFER'
					AND t.from_account_id IN (SELECT id FROM user_accounts)
					AND NOT EXISTS (
						SELECT 1 FROM user_accounts ua WHERE ua.id = t.to_account_id
					)
					THEN t.amount
				ELSE 0
			END), 0) AS expenses,

			COUNT(t.id) AS transactions_count,

			COALESCE(SUM(CASE
				WHEN t.type = 'DEPOSIT'
					AND t.to_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				ELSE 0
			END), 0) AS deposits,

			COALESCE(SUM(CASE
				WHEN t.type = 'WITHDRAW'
					AND t.from_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				ELSE 0
			END), 0) AS withdrawals,

			COALESCE(SUM(CASE
				WHEN t.type = 'TRANSFER'
					AND t.to_account_id IN (SELECT id FROM user_accounts)
					AND NOT EXISTS (
						SELECT 1 FROM user_accounts ua WHERE ua.id = t.from_account_id
					)
					THEN t.amount
				ELSE 0
			END), 0) AS transfers_in,

			COALESCE(SUM(CASE
				WHEN t.type = 'TRANSFER'
					AND t.from_account_id IN (SELECT id FROM user_accounts)
					AND NOT EXISTS (
						SELECT 1 FROM user_accounts ua WHERE ua.id = t.to_account_id
					)
					THEN t.amount
				ELSE 0
			END), 0) AS transfers_out,

			COALESCE(SUM(CASE
				WHEN t.type = 'CARD_PAYMENT'
					AND t.from_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				ELSE 0
			END), 0) AS card_payments,

			COALESCE(SUM(CASE
				WHEN t.type = 'CREDIT_ISSUE'
					AND t.to_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				ELSE 0
			END), 0) AS credit_issues,

			COALESCE(SUM(CASE
				WHEN t.type = 'CREDIT_PAYMENT'
					AND t.from_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				ELSE 0
			END), 0) AS credit_payments,

			COALESCE(SUM(CASE
				WHEN t.type = 'PENALTY'
					AND t.from_account_id IN (SELECT id FROM user_accounts)
					THEN t.amount
				ELSE 0
			END), 0) AS penalties,

			(
				SELECT COALESCE(SUM(ps.amount + ps.penalty_amount), 0)
				FROM payment_schedules ps
				JOIN credits c ON c.id = ps.credit_id
				WHERE c.user_id = $1
				  AND ps.payment_date >= $2::date
				  AND ps.payment_date < $3::date
				  AND ps.status <> 'PAID'
			) AS credit_load

		FROM month_transactions t
	`

	var analytics MonthlyAnalytics

	err := r.db.QueryRowContext(ctx, query, userID, monthStart, monthEnd).Scan(
		&analytics.Income,
		&analytics.Expenses,
		&analytics.TransactionsCount,
		&analytics.Deposits,
		&analytics.Withdrawals,
		&analytics.TransfersIn,
		&analytics.TransfersOut,
		&analytics.CardPayments,
		&analytics.CreditIssues,
		&analytics.CreditPayments,
		&analytics.Penalties,
		&analytics.CreditLoad,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly analytics: %w", err)
	}

	analytics.Income = roundMoney(analytics.Income)
	analytics.Expenses = roundMoney(analytics.Expenses)
	analytics.Deposits = roundMoney(analytics.Deposits)
	analytics.Withdrawals = roundMoney(analytics.Withdrawals)
	analytics.TransfersIn = roundMoney(analytics.TransfersIn)
	analytics.TransfersOut = roundMoney(analytics.TransfersOut)
	analytics.CardPayments = roundMoney(analytics.CardPayments)
	analytics.CreditIssues = roundMoney(analytics.CreditIssues)
	analytics.CreditPayments = roundMoney(analytics.CreditPayments)
	analytics.Penalties = roundMoney(analytics.Penalties)
	analytics.CreditLoad = roundMoney(analytics.CreditLoad)

	return &analytics, nil
}

func (r *AnalyticsRepository) PredictBalance(
	ctx context.Context,
	userID int64,
	accountID int64,
	days int,
) (*BalancePrediction, error) {
	account, err := r.findAccountForPrediction(ctx, accountID)
	if err != nil {
		return nil, err
	}

	if account.UserID != userID {
		return nil, ErrAccountForbidden
	}

	plannedPayments, err := r.sumPlannedPayments(ctx, userID, accountID, days)
	if err != nil {
		return nil, err
	}

	predictedBalance := roundMoney(account.Balance - plannedPayments)

	return &BalancePrediction{
		AccountID:        account.ID,
		CurrentBalance:   roundMoney(account.Balance),
		PlannedPayments:  roundMoney(plannedPayments),
		PredictedBalance: predictedBalance,
		Currency:         account.Currency,
	}, nil
}

func (r *AnalyticsRepository) findAccountForPrediction(ctx context.Context, accountID int64) (*accountPredictionData, error) {
	query := `
		SELECT id, user_id, balance, currency
		FROM accounts
		WHERE id = $1
	`

	var account accountPredictionData

	err := r.db.QueryRowContext(ctx, query, accountID).Scan(
		&account.ID,
		&account.UserID,
		&account.Balance,
		&account.Currency,
	)

	if err == sql.ErrNoRows {
		return nil, ErrAcconutNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find account for prediction: %w", err)
	}

	return &account, nil
}

func (r *AnalyticsRepository) sumPlannedPayments(
	ctx context.Context,
	userID int64,
	accountID int64,
	days int,
) (float64, error) {
	query := `
		SELECT COALESCE(SUM(ps.amount + ps.penalty_amount), 0)
		FROM payment_schedules ps
		JOIN credits c ON c.id = ps.credit_id
		WHERE c.user_id = $1
		  AND c.account_id = $2
		  AND ps.status IN ('PLANNED', 'OVERDUE')
		  AND ps.payment_date <= CURRENT_DATE + ($3::int * INTERVAL '1 day')
	`

	var total float64

	if err := r.db.QueryRowContext(ctx, query, userID, accountID, days).Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to sum planned payments: %w", err)
	}

	return roundMoney(total), nil
}

type accountPredictionData struct {
	ID       int64
	UserID   int64
	Balance  float64
	Currency string
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}
