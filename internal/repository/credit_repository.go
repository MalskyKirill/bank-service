package repository

import (
	"bank-service/internal/models"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrCreditNotFound  = errors.New("credit not found")
	ErrCreditForbidden = errors.New("credit does not belong to user")
)

type CreditRepository struct {
	db *sql.DB
}

func NewCreditRepository(db *sql.DB) *CreditRepository {
	return &CreditRepository{
		db: db,
	}
}

func (r *CreditRepository) Create(
	ctx context.Context,
	userID int64,
	accountID int64,
	amount float64,
	interestRate float64,
	termMonths int,
	monthlyPayment float64,
	schedule []models.PaymentSchedule,
) (*models.Credit, []models.PaymentSchedule, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("filed to begin creating transaction, %w", err)
	}
	defer tx.Rollback()

	if err := r.ensureAccountBelongsToUserForUpdate(ctx, tx, userID, accountID); err != nil {
		return nil, nil, err
	}

	credit, err := r.insertCredit(ctx, tx, userID, accountID, amount, interestRate, termMonths, monthlyPayment)
	if err != nil {
		return nil, nil, err
	}

	createdSchedule := make([]models.PaymentSchedule, 0, len(schedule))

	for _, payment := range schedule {
		payment.CreditID = credit.ID

		createPayment, err := r.insertPaymentSchedule(ctx, tx, payment)
		if err != nil {
			return nil, nil, err
		}

		createdSchedule = append(createdSchedule, *createPayment)
	}

	if err := r.addCreditAmountToAccount(ctx, tx, accountID, amount); err != nil {
		return nil, nil, err
	}

	if err := r.insertCreditIssueTransaction(ctx, tx, userID, accountID, amount); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit credit transaction: %w", err)
	}

	return credit, createdSchedule, nil
}

func (r *CreditRepository) FindAllByUserID(ctx context.Context, userID int64) ([]models.Credit, error) {
	query := `
		SELECT 
			id,
			user_id,
			account_id,
			amount,
			interest_rate,
			term_months,
			monthly_payment,
			status,
			created_at
		FROM credits
		WHERE user_id = $1
		ORDER BY id DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find credits: %w", err)
	}
	defer rows.Close()

	credits := make([]models.Credit, 0)

	for rows.Next() {
		var credit models.Credit

		if err := rows.Scan(
			&credit.ID,
			&credit.UserID,
			&credit.AccountID,
			&credit.Amount,
			&credit.InterestRate,
			&credit.TermMonths,
			&credit.MonthlyPayment,
			&credit.Status,
			&credit.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan credit: %w", err)
		}

		credits = append(credits, credit)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("credits rows error: %w", err)
	}

	return credits, nil
}

func (r *CreditRepository) FindByID(ctx context.Context, userID int64, creditID int64) (*models.Credit, error) {
	query := `
		SELECT 
			id,
			user_id,
			account_id,
			amount,
			interest_rate,
			term_months,
			monthly_payment,
			status,
			created_at
		FROM credits
		WHERE id = $1
	`

	var credit models.Credit

	err := r.db.QueryRowContext(ctx, query, creditID).Scan(
		&credit.ID,
		&credit.UserID,
		&credit.AccountID,
		&credit.Amount,
		&credit.InterestRate,
		&credit.TermMonths,
		&credit.MonthlyPayment,
		&credit.Status,
		&credit.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCreditNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find credit: %w", err)
	}

	if credit.UserID != userID {
		return nil, ErrCreditForbidden
	}

	return &credit, nil
}

func (r *CreditRepository) FindScheduleByCreditID(ctx context.Context, userID int64, creditID int64) ([]models.PaymentSchedule, error) {
	if _, err := r.FindByID(ctx, userID, creditID); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id,
			credit_id,
			payment_date,
			amount,
			principal_amount,
			interest_amount,
			penalty_amount,
			status,
			paid_at
		FROM payment_schedules
		WHERE credit_id = $1
		ORDER BY payment_date ASC, id ASC
	`

	rows, err := r.db.QueryContext(ctx, query, creditID)
	if err != nil {
		return nil, fmt.Errorf("failed to find payment schedule: %w", err)
	}
	defer rows.Close()

	schedule := make([]models.PaymentSchedule, 0)

	for rows.Next() {
		var payment models.PaymentSchedule
		var paidAt sql.NullTime

		if err := rows.Scan(
			&payment.ID,
			&payment.CreditID,
			&payment.PaymentDate,
			&payment.Amount,
			&payment.PrincipalAmount,
			&payment.InterestAmount,
			&payment.PenaltyAmount,
			&payment.Status,
			&paidAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan payment schedule: %w", err)
		}

		if paidAt.Valid {
			payment.PaidAt = &paidAt.Time
		}

		schedule = append(schedule, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("payment schedule rows error: %w", err)
	}

	return schedule, nil
}

func (r *CreditRepository) insertPaymentSchedule(ctx context.Context, tx *sql.Tx, payment models.PaymentSchedule) (*models.PaymentSchedule, error) {
	query := `
		INSERT INTO payment_schedules (
			credit_id,
			payment_date,
			amount,
			principal_amount,
			interest_amount,
			penalty_amount,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING
			id,
			credit_id,
			payment_date,
			amount,
			principal_amount,
			interest_amount,
			penalty_amount,
			status,
			paid_at
	`

	var createdPayment models.PaymentSchedule
	var paidAt sql.NullTime

	err := tx.QueryRowContext(
		ctx,
		query,
		payment.CreditID,
		payment.PaymentDate,
		payment.Amount,
		payment.PrincipalAmount,
		payment.InterestAmount,
		payment.PenaltyAmount,
		payment.Status,
	).Scan(
		&createdPayment.ID,
		&createdPayment.CreditID,
		&createdPayment.PaymentDate,
		&createdPayment.Amount,
		&createdPayment.PrincipalAmount,
		&createdPayment.InterestAmount,
		&createdPayment.PenaltyAmount,
		&createdPayment.Status,
		&paidAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert payment schedule: %w", err)
	}

	if paidAt.Valid {
		createdPayment.PaidAt = &paidAt.Time
	}

	return &createdPayment, nil
}

func (r *CreditRepository) ensureAccountBelongsToUserForUpdate(ctx context.Context, tx *sql.Tx, userId int64, accountId int64) error {
	query := `
		SELECT user_id
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`

	var ownerId int64

	err := tx.QueryRowContext(ctx, query, accountId).Scan(&ownerId)
	if err != nil {
		return fmt.Errorf("filed to check account owner, %w", err)
	}

	if ownerId != userId {
		return ErrAccountForbidden
	}

	return nil
}

func (r *CreditRepository) insertCredit(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	accountID int64,
	amount float64,
	interestRate float64,
	termMonths int,
	monthlyPayment float64,
) (*models.Credit, error) {
	query := `
		INSERT INTO credits (
			user_id,
			account_id,
			amount,
			interest_rate,
			term_months,
			monthly_payment
		)
		VALUES ($1, $2,  $3, $4, $5, $6)
		RETURNING 
			id,
			user_id,
			account_id,
			amount,
			interest_rate,
			term_months,
			monthly_payment,
			status,
			created_at
	`

	var credit models.Credit

	err := tx.QueryRowContext(
		ctx,
		query,
		userID,
		accountID,
		amount,
		interestRate,
		termMonths,
		monthlyPayment,
	).Scan(
		&credit.ID,
		&credit.UserID,
		&credit.AccountID,
		&credit.Amount,
		&credit.InterestRate,
		&credit.TermMonths,
		&credit.MonthlyPayment,
		&credit.Status,
		&credit.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("filed to insert credit, %w", err)
	}

	return &credit, nil
}

func (r *CreditRepository) addCreditAmountToAccount(ctx context.Context, tx *sql.Tx, accountId int64, amount float64) error {
	query := `
		UPDATE accounts
		SET balance = balance + $1
		WHERE id = $2
	`

	if _, err := tx.ExecContext(ctx, query, amount, accountId); err != nil {
		return fmt.Errorf("failed to add credit amount to account: %w", err)
	}

	return nil
}

func (r *CreditRepository) insertCreditIssueTransaction(ctx context.Context, tx *sql.Tx, userID int64, accountID int64, amount float64) error {
	query := `
		INSERT INTO transactions (
			user_id,
			from_account_id,
			to_account_id,
			amount,
			type,
			status,
			description
		)
		VALUES ($1, NULL, $2, $3, 'CREDIT_ISSUE', 'SUCCESS', 'Credit issued')
	`

	if _, err := tx.ExecContext(ctx, query, userID, accountID, amount); err != nil {
		return fmt.Errorf("failed to insert credit issue transaction: %w", err)
	}

	return nil
}
