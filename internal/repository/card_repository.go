package repository

import (
	"bank-service/internal/models"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrCardNotFound  = errors.New("card not fount")
	ErrCardForbidden = errors.New("ard does not belong to user")
)

type CardRepository struct {
	db        *sql.DB
	pgpSecret string
}

func NewCardRepository(db *sql.DB, pgpSecret string) *CardRepository {
	return &CardRepository{
		db:        db,
		pgpSecret: pgpSecret,
	}
}

func (r *CardRepository) Create(ctx context.Context, card *models.Card) error {
	query := `
		INSERT INTO cards (
			user_id,
			account_id,
			encrypted_number,
			encrypted_expiry,
			cvv_hash,
			number_hmac
		)
		VALUES (
			$1,
			$2,
			encode(pgp_sym_encrypt($3, $4), 'base64'),
			encode(pgp_sym_encrypt($5, $4), 'base64'),
			$6,
			$7
		)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		card.UserID,
		card.AccountID,
		card.Number,
		r.pgpSecret,
		card.Expiry,
		card.CVVHash,
		card.NumberHMAC,
	).Scan(&card.ID, &card.CreatedAt)

	if err != nil {
		return fmt.Errorf("filed to create card, %w", err)
	}

	return nil
}

func (r *CardRepository) FindAllByUserId(ctx context.Context, userId int64) ([]models.Card, error) {
	query := `
		SELECT 
			id,
			user_id,
			account_id,
			pgp_sym_decrypt(decode(encrypted_number, 'base64'), $2) AS number,
			pgp_sym_decrypt(decode(encrypted_expiry, 'base64'), $2) AS expiry,
			number_hmac,
			created_at
		FROM cards
		WHERE user_id = $1
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, userId, r.pgpSecret)
	if err != nil {
		return nil, fmt.Errorf("filed to find cards, %w", err)
	}

	defer rows.Close()

	cards := make([]models.Card, 0)

	for rows.Next() {
		var card models.Card

		err := rows.Scan(
			&card.ID,
			&card.UserID,
			&card.AccountID,
			&card.Number,
			&card.Expiry,
			&card.NumberHMAC,
			&card.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("filed to scan card, %w", err)
		}

		cards = append(cards, card)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cards rows error, %w", err)
	}

	return cards, nil
}

func (r *CardRepository) FindById(ctx context.Context, id int64) (*models.Card, error) {
	query := `
		SELECT
			id,
			user_id,
			account_id,
			pgp_sym_decrypt(decode(encrypted_number, 'base64'), $2) AS number,
			pgp_sym_decrypt(decode(encrypted_expiry, 'base64'), $2) AS expiry,
			number_hmac,
			created_at
		FROM cards
		WHERE id = $1
	`

	var card models.Card

	err := r.db.QueryRowContext(ctx, query, id, r.pgpSecret).Scan(
		&card.ID,
		&card.UserID,
		&card.AccountID,
		&card.Number,
		&card.Expiry,
		&card.NumberHMAC,
		&card.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrCardNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find card: %w", err)
	}

	return &card, nil
}

func (r *CardRepository) Pay(ctx context.Context, userId int64, cardId int64, amount float64, description string) (*models.Account, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("filed begin card transaction, %w", err)
	}
	defer tx.Rollback()

	card, err := r.findCardForPayment(ctx, tx, cardId)
	if err != nil {
		return nil, err
	}

	if card.UserID != userId {
		return nil, ErrCardForbidden
	}

	account, err := r.findAccountForUpdate(ctx, tx, card.AccountID)
	if err != nil {
		return nil, err
	}

	if account.Balance < amount {
		return nil, ErrInsufficientFunds
	}

	updateAccount, err := r.subtractAccountBalance(ctx, tx, account.ID, amount)
	if err != nil {
		return nil, err
	}

	err = r.insertCardPaymentTransaction(
		ctx,
		tx,
		userId,
		account.ID,
		amount,
		description,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("filed to commit card payment transaction, %w", err)
	}

	return updateAccount, nil
}

func (r *CardRepository) EnsureAccountBelongsToUser(ctx context.Context, userId int64, accountId int64) error {
	query := `
		SELECT user_id
		FROM accounts 
		WHERE id = $1
	`

	var ownerId int64

	err := r.db.QueryRowContext(ctx, query, accountId).Scan(&ownerId)
	if err == sql.ErrNoRows {
		return ErrAcconutNotFound
	}

	if err != nil {
		return fmt.Errorf("filed to check accaunt owner, %w", err)
	}

	if ownerId != userId {
		return ErrAccountForbidden
	}

	return nil
}

func (r *CardRepository) findCardForPayment(ctx context.Context, tx *sql.Tx, cardId int64) (*models.Card, error) {
	query := `
		SELECT id, user_id, account_id
		from cards
		WHERE id = $1
		FOR UPDATE
	`

	var card models.Card

	err := tx.QueryRowContext(ctx, query, cardId).Scan(
		&card.ID,
		&card.UserID,
		&card.AccountID,
	)

	if err == sql.ErrNoRows {
		return nil, ErrCardNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("filed to find card for payment, %w", err)
	}

	return &card, nil
}

func (r *CardRepository) findAccountForUpdate(ctx context.Context, tx *sql.Tx, accountId int64) (*models.Account, error) {
	query := `
		SELECT id, user_id, balance, currency, created_at
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`

	var account models.Account

	err := tx.QueryRowContext(ctx, query, accountId).Scan(
		&account.ID,
		&account.UserID,
		&account.Balance,
		&account.Currency,
		&account.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrAcconutNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("filed to find account for card payment, %w", err)
	}

	return &account, nil
}

func (r *CardRepository) subtractAccountBalance(ctx context.Context, tx *sql.Tx, accountId int64, amount float64) (*models.Account, error) {
	query := `
		UPDATE accounts
		SET balance = balance - $1
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
		return nil, fmt.Errorf("failed to subtract account balanse for card payment, %w", err)
	}

	return &account, nil
}

func (r *CardRepository) insertCardPaymentTransaction(ctx context.Context, tx *sql.Tx, userId int64, accountId int64, amount float64, description string) error {
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
		VALUES ($1, $2, NULL, $3, 'CARD_PAYMENT', 'SUCCESS', $4)
	`

	_, err := tx.ExecContext(ctx, query, userId, accountId, amount, description)

	if err != nil {
		return fmt.Errorf("failed to insert card payment transaction: %w", err)
	}

	return nil
}
