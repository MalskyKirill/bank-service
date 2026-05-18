package repository

import (
	"bank-service/internal/models"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrAcconutNotFound     = errors.New("account not found")
	ErrAccountForbidden    = errors.New("accounts does not belong to user")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrSameAccountTransfer = errors.New("cannot transfer to the same account")
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
		VALUES ($1, $2)
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

func (r *AccountRepository) Deposit(ctx context.Context, userID int64, accountID int64, amount float64) (*models.Account, error) {
	tx, err := r.db.BeginTx(ctx, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback()

	account, err := r.findByIDForUpdate(ctx, tx, accountID)
	if err != nil {
		return nil, err
	}

	if account.UserID != userID {
		return nil, ErrAccountForbidden
	}

	updateAccount, err := r.addBalance(ctx, tx, accountID, amount)
	if err != nil {
		return nil, err
	}

	err = r.insertTransaction(ctx, tx, userID, nil, &accountID, amount, "DEPOSIT", "Account deposit")
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit deposit transaction: %w", err)
	}

	return updateAccount, nil
}

func (r *AccountRepository) Withdraw(ctx context.Context, userId int64, accountId int64, amount float64) (*models.Account, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback()

	account, err := r.findByIDForUpdate(ctx, tx, accountId)
	if err != nil {
		return nil, err
	}

	if account.UserID != userId {
		return nil, ErrAccountForbidden
	}

	if account.Balance < amount {
		return nil, ErrInsufficientFunds
	}

	updateAccount, err := r.substractBalanse(ctx, tx, accountId, amount)
	if err != nil {
		return nil, err
	}

	err = r.insertTransaction(
		ctx, tx, userId, &accountId, nil, amount, "WITHDRAW", "Account withdrawal",
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit substract transaction: %w", err)
	}

	return updateAccount, nil
}

func (r *AccountRepository) Transfer(ctx context.Context, userId int64, fromAccountId int64, toAccountId int64, amount float64) (*models.Account, error) {
	if fromAccountId == toAccountId {
		return nil, ErrSameAccountTransfer
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback()

	fromAccount, toAccount, err := r.lockTransferAccounts(ctx, tx, fromAccountId, toAccountId)
	if err != nil {
		return nil, err
	}

	if fromAccount.ID != userId {
		return nil, ErrAccountForbidden
	}

	if fromAccount.Balance < amount {
		return nil, ErrInsufficientFunds
	}

	updateFromAccount, err := r.substractBalanse(ctx, tx, fromAccount.ID, amount)
	if err != nil {
		return nil, err
	}

	if _, err := r.addBalance(ctx, tx, toAccount.ID, amount); err != nil {
		return nil, err
	}

	err = r.insertTransaction(ctx, tx, userId, &fromAccountId, &toAccountId, amount, "TRANSFER", "Transfer between accounts")
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transfer transaction: %w", err)
	}

	return updateFromAccount, nil
}

func (r *AccountRepository) findByIDForUpdate(ctx context.Context, tx *sql.Tx, accountID int64) (*models.Account, error) {
	query := `
		SELECT id, user_id, balance, currency, created_at
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`

	var account models.Account

	err := tx.QueryRowContext(ctx, query, accountID).Scan(
		&account.ID,
		&account.UserID,
		&account.Balance,
		&account.Currency,
		&account.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAcconutNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find account for update: %w", err)
	}

	return &account, nil
}

func (r *AccountRepository) lockTransferAccounts(ctx context.Context, tx *sql.Tx, fromAccountID int64, toAccountID int64) (*models.Account, *models.Account, error) {
	firstId := fromAccountID
	secondId := toAccountID

	if firstId > secondId {
		firstId, secondId = secondId, firstId
	}

	firstAccount, err := r.findByIDForUpdate(ctx, tx, firstId)
	if err != nil {
		return nil, nil, err
	}

	secondAccount, err := r.findByIDForUpdate(ctx, tx, secondId)
	if err != nil {
		return nil, nil, err
	}

	if firstAccount.ID == fromAccountID {
		return firstAccount, secondAccount, nil
	}

	return secondAccount, firstAccount, nil
}

func (r *AccountRepository) addBalance(ctx context.Context, tx *sql.Tx, accountId int64, amount float64) (*models.Account, error) {
	query := `
		UPDATE accounts
		SET balance = balance + $1
		WHERE id = $2
		RETURNING id, user_id, balance, currency, created_at
	`

	var account models.Account

	err := tx.QueryRowContext(ctx, query, amount, accountId).Scan(
		&account.ID,
		&account.UserID,
		&account.Balance,
		&account.Currency,
		&account.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to add account balance: %w", err)
	}

	return &account, nil
}

func (r *AccountRepository) substractBalanse(ctx context.Context, tx *sql.Tx, accountId int64, ammount float64) (*models.Account, error) {
	query := `
		UPDATE accounts
		SET balance = balance - $1
		WHERE id = $2
		RETURNING id, user_id, balance, currency, created_at
	`

	var account models.Account

	err := tx.QueryRowContext(ctx, query, ammount, accountId).Scan(
		&account.ID,
		&account.UserID,
		&account.Balance,
		&account.Currency,
		&account.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to substract account balance: %w", err)
	}

	return &account, nil
}

func (r *AccountRepository) insertTransaction(
	ctx context.Context,
	tx *sql.Tx,
	userId int64,
	fromAccountId *int64,
	toAccountId *int64,
	amount float64,
	transactionType string,
	description string) error {

	query := `
			INSERT INTO transactions (
				user_id, from_account_id, to_account_id, amount, type, status, description
			)
			VALUES ($1, $2, $3, $4, $5, 'SUCCESS', $6)
		`

	_, err := tx.ExecContext(
		ctx, query, userId, nullableInt64(fromAccountId), nullableInt64(toAccountId), amount, transactionType, description,
	)

	if err != nil {
		return fmt.Errorf("failed to inser transaction, %w", err)
	}

	return nil
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}
