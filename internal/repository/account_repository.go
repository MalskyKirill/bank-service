package repository

import (
	"bank-service/internal/models"
	"context"
	"database/sql"
	"fmt"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{
		db: db,
	}
}

func (r *AccountRepository) Create(ctx context.Context, userID int64, currency string) (*models.Account, error) {
	query := `
		INSERT INTO accounts (user_id, currency)
		VALUE ($1, $2)
		RETURNING id, user_id, balance, currency, created_at`

	var account models.Account

	err := r.db.QueryRowContext(ctx, query, userID, currency).Scan(
		&account.ID,
		&account.UserID,
		&account.Balance,
		&account.Currency,
		&account.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create account, %w", err)
	}

	return &account, nil
}

func (r *AccountRepository) FindAllByUserID(ctx context.Context, userId int64) ([]models.Account, error) {
	query := `
		SELECT id, user_id, balance, currency, created_at
		FROM accounts
		WHERE user_id = $1
		ORDER BY id	
	`

	rows, err := r.db.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to find accounts: %w", err)
	}

	defer rows.Close()

	accounts := make([]models.Account, 0)

	for rows.Next() {
		var account models.Account

		err := rows.Scan(
			&account.ID,
			&account.UserID,
			&account.Balance,
			&account.Currency,
			&account.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to find accounts: %w", err)
		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("accounts rows error: %w", err)
	}

	return accounts, nil
}
