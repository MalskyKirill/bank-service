package repository

import (
	"bank-service/internal/models"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
)

var (
	ErrCreditNotFound  = errors.New("credit not found")
	ErrCreditForbidden = errors.New("credit does not belong to user")
)

const (
	duePaymentPaid    duePaymentResult = "PAID"
	duePaymentOverdue duePaymentResult = "OVERDUE"
)

type CreditRepository struct {
	db *sql.DB
}

type duePaymentResult string

type duePayment struct {
	ID            int64
	CreditID      int64
	UserID        int64
	AccountID     int64
	Amount        float64
	PenaltyAmount float64
	Status        string
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

func (r *CreditRepository) ProcessDuePayments(ctx context.Context) (processed int, paid int, overdue int, err error) {
	paymentIDs, err := r.findDuePaymentIDs(ctx)
	if err != nil {
		return 0, 0, 0, err
	}

	for _, paymentID := range paymentIDs {
		result, err := r.processDuePaymentByID(ctx, paymentID)
		if err != nil {
			return processed, paid, overdue, fmt.Errorf("failed to process payment schedule %d: %w", paymentID, err)
		}

		processed++

		switch result {
		case duePaymentPaid:
			paid++
		case duePaymentOverdue:
			overdue++
		}
	}

	return processed, paid, overdue, nil
}

func (r *CreditRepository) findDuePaymentIDs(ctx context.Context) ([]int64, error) {
	query := `
		SELECT ps.id
		FROM payment_schedules ps
		JOIN credits c ON c.id = ps.credit_id
		WHERE ps.status IN ('PLANNED', 'OVERDUE')
		  AND ps.payment_date <= CURRENT_DATE
		  AND c.status IN ('ACTIVE', 'OVERDUE')
		ORDER BY ps.payment_date ASC, ps.id ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find due payment schedules: %w", err)
	}
	defer rows.Close()

	paymentIDs := make([]int64, 0)

	for rows.Next() {
		var id int64

		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan due payment id: %w", err)
		}

		paymentIDs = append(paymentIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("due payments rows error: %w", err)
	}

	return paymentIDs, nil
}

func (r *CreditRepository) processDuePaymentByID(ctx context.Context, paymentID int64) (duePaymentResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin due payment transaction: %w", err)
	}
	defer tx.Rollback()

	payment, err := r.findDuePaymentForUpdate(ctx, tx, paymentID)
	if err != nil {
		return "", err
	}

	accountBalance, err := r.findAccountBalanceForUpdate(ctx, tx, payment.AccountID)
	if err != nil {
		return "", err
	}

	totalToPay := roundMoneyValue(payment.Amount + payment.PenaltyAmount)

	if accountBalance >= totalToPay {
		if err := r.payCreditSchedule(ctx, tx, payment, totalToPay); err != nil {
			return "", err
		}

		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("failed to commit paid credit payment: %w", err)
		}

		return duePaymentPaid, nil
	}

	if err := r.markCreditScheduleOverdue(ctx, tx, payment); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit overdue credit payment: %w", err)
	}

	return duePaymentOverdue, nil
}

func (r *CreditRepository) findDuePaymentForUpdate(ctx context.Context, tx *sql.Tx, paymentID int64) (*duePayment, error) {
	query := `
		SELECT
			ps.id,
			ps.credit_id,
			ps.amount,
			ps.penalty_amount,
			ps.status,
			c.user_id,
			c.account_id
		FROM payment_schedules ps
		JOIN credits c ON c.id = ps.credit_id
		WHERE ps.id = $1
		  AND ps.status IN ('PLANNED', 'OVERDUE')
		  AND ps.payment_date <= CURRENT_DATE
		  AND c.status IN ('ACTIVE', 'OVERDUE')
		FOR UPDATE OF ps, c
	`

	var payment duePayment

	err := tx.QueryRowContext(ctx, query, paymentID).Scan(
		&payment.ID,
		&payment.CreditID,
		&payment.Amount,
		&payment.PenaltyAmount,
		&payment.Status,
		&payment.UserID,
		&payment.AccountID,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCreditNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find due payment for update: %w", err)
	}

	return &payment, nil
}

func (r *CreditRepository) findAccountBalanceForUpdate(ctx context.Context, tx *sql.Tx, accountID int64) (float64, error) {
	query := `
		SELECT balance
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`

	var balance float64

	err := tx.QueryRowContext(ctx, query, accountID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrAcconutNotFound
	}

	if err != nil {
		return 0, fmt.Errorf("failed to find account balance for update: %w", err)
	}

	return balance, nil
}

func (r *CreditRepository) payCreditSchedule(
	ctx context.Context,
	tx *sql.Tx,
	payment *duePayment,
	totalToPay float64,
) error {
	if err := r.subtractCreditPaymentFromAccount(ctx, tx, payment.AccountID, totalToPay); err != nil {
		return err
	}

	if err := r.markPaymentSchedulePaid(ctx, tx, payment.ID); err != nil {
		return err
	}

	if err := r.insertCreditPaymentTransaction(
		ctx,
		tx,
		payment.UserID,
		payment.AccountID,
		payment.Amount,
	); err != nil {
		return err
	}

	if payment.PenaltyAmount > 0 {
		if err := r.insertPenaltyTransaction(
			ctx,
			tx,
			payment.UserID,
			payment.AccountID,
			payment.PenaltyAmount,
			"SUCCESS",
			"Credit overdue penalty paid",
		); err != nil {
			return err
		}
	}

	if err := r.updateCreditStatusAfterPayment(ctx, tx, payment.CreditID); err != nil {
		return err
	}

	return nil
}

func (r *CreditRepository) markCreditScheduleOverdue(
	ctx context.Context,
	tx *sql.Tx,
	payment *duePayment,
) error {
	if payment.Status == "PLANNED" {
		penaltyAmount := roundMoneyValue(payment.Amount * 0.10)

		query := `
			UPDATE payment_schedules
			SET status = 'OVERDUE',
			    penalty_amount = penalty_amount + $1
			WHERE id = $2
		`

		if _, err := tx.ExecContext(ctx, query, penaltyAmount, payment.ID); err != nil {
			return fmt.Errorf("failed to mark payment schedule overdue: %w", err)
		}

		if err := r.insertPenaltyTransaction(
			ctx,
			tx,
			payment.UserID,
			payment.AccountID,
			penaltyAmount,
			"PENDING",
			"Credit overdue penalty accrued",
		); err != nil {
			return err
		}
	}

	if err := r.markCreditOverdue(ctx, tx, payment.CreditID); err != nil {
		return err
	}

	return nil
}

func (r *CreditRepository) subtractCreditPaymentFromAccount(
	ctx context.Context,
	tx *sql.Tx,
	accountID int64,
	amount float64,
) error {
	query := `
		UPDATE accounts
		SET balance = balance - $1
		WHERE id = $2
	`

	if _, err := tx.ExecContext(ctx, query, amount, accountID); err != nil {
		return fmt.Errorf("failed to subtract credit payment from account: %w", err)
	}

	return nil
}

func (r *CreditRepository) markPaymentSchedulePaid(ctx context.Context, tx *sql.Tx, paymentScheduleID int64) error {
	query := `
		UPDATE payment_schedules
		SET status = 'PAID',
		    paid_at = NOW()
		WHERE id = $1
	`

	if _, err := tx.ExecContext(ctx, query, paymentScheduleID); err != nil {
		return fmt.Errorf("failed to mark payment schedule paid: %w", err)
	}

	return nil
}

func (r *CreditRepository) markCreditOverdue(ctx context.Context, tx *sql.Tx, creditID int64) error {
	query := `
		UPDATE credits
		SET status = 'OVERDUE'
		WHERE id = $1
	`

	if _, err := tx.ExecContext(ctx, query, creditID); err != nil {
		return fmt.Errorf("failed to mark credit overdue: %w", err)
	}

	return nil
}

func (r *CreditRepository) updateCreditStatusAfterPayment(ctx context.Context, tx *sql.Tx, creditID int64) error {
	unpaidCount, err := r.countUnpaidSchedules(ctx, tx, creditID)
	if err != nil {
		return err
	}

	if unpaidCount == 0 {
		query := `
			UPDATE credits
			SET status = 'CLOSED'
			WHERE id = $1
		`

		if _, err := tx.ExecContext(ctx, query, creditID); err != nil {
			return fmt.Errorf("failed to close credit: %w", err)
		}

		return nil
	}

	overdueCount, err := r.countOverdueSchedules(ctx, tx, creditID)
	if err != nil {
		return err
	}

	status := "ACTIVE"
	if overdueCount > 0 {
		status = "OVERDUE"
	}

	query := `
		UPDATE credits
		SET status = $1
		WHERE id = $2
	`

	if _, err := tx.ExecContext(ctx, query, status, creditID); err != nil {
		return fmt.Errorf("failed to update credit status: %w", err)
	}

	return nil
}

func (r *CreditRepository) countUnpaidSchedules(ctx context.Context, tx *sql.Tx, creditID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM payment_schedules
		WHERE credit_id = $1
		  AND status <> 'PAID'
	`

	var count int

	if err := tx.QueryRowContext(ctx, query, creditID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count unpaid schedules: %w", err)
	}

	return count, nil
}

func (r *CreditRepository) countOverdueSchedules(ctx context.Context, tx *sql.Tx, creditID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM payment_schedules
		WHERE credit_id = $1
		  AND status = 'OVERDUE'
	`

	var count int

	if err := tx.QueryRowContext(ctx, query, creditID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count overdue schedules: %w", err)
	}

	return count, nil
}

func (r *CreditRepository) insertCreditPaymentTransaction(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	accountID int64,
	amount float64,
) error {
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
		VALUES ($1, $2, NULL, $3, 'CREDIT_PAYMENT', 'SUCCESS', 'Credit payment')
	`

	if _, err := tx.ExecContext(ctx, query, userID, accountID, amount); err != nil {
		return fmt.Errorf("failed to insert credit payment transaction: %w", err)
	}

	return nil
}

func (r *CreditRepository) insertPenaltyTransaction(
	ctx context.Context,
	tx *sql.Tx,
	userID int64,
	accountID int64,
	amount float64,
	status string,
	description string,
) error {
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
		VALUES ($1, $2, NULL, $3, 'PENALTY', $4, $5)
	`

	if _, err := tx.ExecContext(ctx, query, userID, accountID, amount, status, description); err != nil {
		return fmt.Errorf("failed to insert penalty transaction: %w", err)
	}

	return nil
}

func roundMoneyValue(value float64) float64 {
	return math.Round(value*100) / 100
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
