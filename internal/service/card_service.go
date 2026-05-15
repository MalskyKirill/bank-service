package service

import (
	"bank-service/internal/apperror"
	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repository"
	"bank-service/internal/security"
	"context"
	"errors"
	"net/http"
	"strings"
)

type CardService struct {
	cardRepository *repository.CardRepository
	hmacSecret     []byte
}

func NewCardService(cardRepository *repository.CardRepository, hmacSecret string) *CardService {
	return &CardService{
		cardRepository: cardRepository,
		hmacSecret:     []byte(hmacSecret),
	}
}

func (s *CardService) CreateCard(ctx context.Context, userId int64, req dto.CreateCardRequest) (*dto.CreateCardResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	if req.AccountID <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "account id is requared")
	}

	if err := s.cardRepository.EnsureAccountBelongsToUser(ctx, userId, req.AccountID); err != nil {
		return nil, mapCardRepositoryError(err, "failed to check account")
	}

	number, err := security.GenerateCardNumber()
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to generate card number")
	}

	expiry := security.GenerateCardExpiry()

	cvv, err := security.GenerateCVV()
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to generate cvv")
	}

	hashCvv, err := security.HashPassword(cvv)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to hash cvv")
	}

	numberHMAC := security.ComputeHMAC(number, s.hmacSecret)

	card := &models.Card{
		UserID:     userId,
		AccountID:  req.AccountID,
		Number:     number,
		Expiry:     expiry,
		CVVHash:    hashCvv,
		NumberHMAC: numberHMAC,
	}

	if err := s.cardRepository.Create(ctx, card); err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to create card")
	}

	return &dto.CreateCardResponse{
		ID:        card.ID,
		AccountId: card.AccountID,
		Number:    card.Number,
		Expiry:    card.Expiry,
		CVV:       cvv,
		CreatedAt: card.CreatedAt,
	}, nil
}

func (s *CardService) GetUserCards(ctx context.Context, userId int64) (*[]dto.CardResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	cards, err := s.cardRepository.FindAllByUserId(ctx, userId)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to get cards")
	}

	response := make([]dto.CardResponse, 0, len(cards))

	for _, card := range cards {
		if !security.VerifyHMAC(card.Number, card.NumberHMAC, s.hmacSecret) {
			return nil, apperror.New(http.StatusInternalServerError, "card data integrity check failed")
		}

		response = append(response, dto.CardResponse{
			ID:        card.ID,
			AccountId: card.AccountID,
			Number:    security.MaskCardNumber(card.Number),
			Expiry:    card.Expiry,
			CreatedAt: card.CreatedAt,
		})
	}

	return &response, nil
}

func (s *CardService) GetCard(ctx context.Context, userId int64, cardId int64) (*dto.CardResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	if cardId <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "invalid card id")
	}

	card, err := s.cardRepository.FindById(ctx, cardId)
	if err != nil {
		return nil, mapCardRepositoryError(err, "failed to get card")
	}

	if card.UserID != userId {
		return nil, apperror.New(http.StatusForbidden, "access denied to card")
	}

	if !security.VerifyHMAC(card.Number, card.NumberHMAC, s.hmacSecret) {
		return nil, apperror.New(http.StatusInternalServerError, "card data integrity check filed")
	}

	return &dto.CardResponse{
		ID:        card.ID,
		AccountId: card.AccountID,
		Number:    card.Number,
		Expiry:    card.Expiry,
		CreatedAt: card.CreatedAt,
	}, nil
}

func (s *CardService) Pay(ctx context.Context, userId int64, cardId int64, req dto.CardPaymentRequest) (*dto.CardPaymentResponse, error) {
	if userId <= 0 {
		return nil, apperror.New(http.StatusUnauthorized, "unauthorized")
	}

	if cardId <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "invalid card id")
	}

	if req.Amount <= 0 {
		return nil, apperror.New(http.StatusBadRequest, "invalid amount")
	}

	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = "Card payment"
	}

	account, err := s.cardRepository.Pay(ctx, userId, cardId, req.Amount, description)
	if err != nil {
		return nil, mapCardRepositoryError(err, "failed to process card payment")
	}

	return &dto.CardPaymentResponse{
		Message: "card payment completed successfully",
		CardId:  cardId,
		Amount:  req.Amount,
		Account: *toAccountResponse(account),
	}, nil
}

func mapCardRepositoryError(err error, defoltMessage string) error {
	if errors.Is(err, repository.ErrCardNotFound) {
		return apperror.New(http.StatusNotFound, "card not found")
	}

	if errors.Is(err, repository.ErrCardForbidden) {
		return apperror.New(http.StatusForbidden, "access denied to card")
	}

	if errors.Is(err, repository.ErrAcconutNotFound) {
		return apperror.New(http.StatusNotFound, "accaunt not found")
	}

	if errors.Is(err, repository.ErrAccountForbidden) {
		return apperror.New(http.StatusForbidden, "access denied to account")
	}

	if errors.Is(err, repository.ErrInsufficientFunds) {
		return apperror.New(http.StatusBadRequest, "insuffficient funds")
	}

	return apperror.New(http.StatusInternalServerError, defoltMessage)
}
