package repository

import (
	"bank-service/internal/models"
	"context"
	"database/sql"
	"fmt"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{
		db: db,
	}
}

func (r *TransactionRepository) FindAllByUserId(ctx context.Context, userId int64) ([]models.Transaction, error) {
	query := `
		SELECT 
			t.id,
			t.user_id,
			t.from_account_id,
			t.to_account_id,
			t.amount,
			t.type,
			t.status,
			t.description,
			t.created_at
		FROM transactions as t
		WHERE t.user_id = $1 OR t.from_account_id IN (SELECT id FROM accounts WHERE user_id = $1) OR t.to_account_id IN (SELECT id FROM accounts WHERE user_id = $1)
		ORDER BY t.created_at DESC, t.id DESC
		`

	rows, err := r.db.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to find transactions, %w", err)
	}
	defer rows.Close()

	return scanTransactions(rows)
}

func (r *TransactionRepository) FindByAccountId(ctx context.Context, userId int64, accountId int64) ([]models.Transaction, error) {
	if err := r.ensureAccountBelongsToUser(ctx, userId, accountId); err != nil {
		return nil, err
	}

	query := `
		SELECT 
			id,
			user_id,
			from_account_id,
			to_account_id,
			amount,
			type,
			status,
			description,
			created_at
		FROM transactions
		WHERE from_account_id = $1 or to_account_id = $1
		ORDER BY created_at DESC, id DESC
	`

	rows, err := r.db.QueryContext(ctx, query, accountId)
	if err != nil {
		return nil, fmt.Errorf("failed to account transactions, %w", err)
	}
	defer rows.Close()

	return scanTransactions(rows)
}

func (r *TransactionRepository) ensureAccountBelongsToUser(ctx context.Context, userId int64, accountId int64) error {
	query := `SELECT user_id FROM accounts WHERE id = $1`

	var ownerId int64

	err := r.db.QueryRowContext(ctx, query, accountId).Scan(&ownerId)
	if err == sql.ErrNoRows {
		return ErrAcconutNotFound
	}

	if err != nil {
		return fmt.Errorf("failed to check account owner, %w", err)
	}

	if ownerId != userId {
		return ErrAccountForbidden
	}

	return nil
}

func scanTransactions(rows *sql.Rows) ([]models.Transaction, error) {
	transactions := make([]models.Transaction, 0)

	for rows.Next() {
		var transaction models.Transaction

		var fromAccountID sql.NullInt64
		var toAccountID sql.NullInt64
		var description sql.NullString

		err := rows.Scan(
			&transaction.ID,
			&transaction.UserID,
			&fromAccountID,
			&toAccountID,
			&transaction.Amount,
			&transaction.Type,
			&transaction.Status,
			&description,
			&transaction.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("falied to scan transaction, %w", err)
		}

		if fromAccountID.Valid {
			transaction.FromAccountID = &fromAccountID.Int64
		}

		if toAccountID.Valid {
			transaction.ToAccountID = &toAccountID.Int64
		}

		if description.Valid {
			transaction.Description = &description.String
		}

		transactions = append(transactions, transaction)

	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("transactions rows error, %w", err)
	}

	return transactions, nil
}
