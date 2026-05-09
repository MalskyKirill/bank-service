package service

import (
	"bank-service/internal/apperror"
	"bank-service/internal/dto"
	"bank-service/internal/models"
	"bank-service/internal/repository"
	"bank-service/internal/security"
	"context"
	"net/http"
	"net/mail"
	"strings"
)

type AuthService struct {
	userRepository *repository.UserRepository
	jwtServicce    *security.JWTService
}

func NewAuthService(userRepository *repository.UserRepository, jwtService *security.JWTService) *AuthService {
	return &AuthService{
		userRepository: userRepository,
		jwtServicce:    jwtService,
	}
}

func (s *AuthService) Registration(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)

	if email == "" {
		return nil, apperror.New(http.StatusBadRequest, "email is required")
	}

	if _, err := mail.ParseAddress(email); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid email")
	}

	if username == "" {
		return nil, apperror.New(http.StatusBadRequest, "username is required")
	}

	if len(username) < 2 {
		return nil, apperror.New(http.StatusBadRequest, "username must be at least 2 characters")
	}

	if password == "" {
		return nil, apperror.New(http.StatusBadRequest, "password is required")
	}

	if len(password) < 8 {
		return nil, apperror.New(http.StatusBadRequest, "password must be at least 8 characters")
	}

	existindgUserByEmail, err := s.userRepository.FindByEmail(ctx, email)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to check email")
	}

	if existindgUserByEmail != nil {
		return nil, apperror.New(http.StatusConflict, "email already exists")
	}

	existingUserByUserName, err := s.userRepository.FindByUsername(ctx, username)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to check email")
	}

	if existingUserByUserName != nil {
		return nil, apperror.New(http.StatusInternalServerError, "username already exists")
	}

	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to hash password")
	}

	user := &models.User{
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
	}

	if err := s.userRepository.Create(ctx, user); err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to create user")
	}

	return &dto.RegisterResponse{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := strings.TrimSpace(req.Password)

	if email == "" {
		return nil, apperror.New(http.StatusBadRequest, "email is required")
	}

	if password == "" {
		return nil, apperror.New(http.StatusBadRequest, "password is required")
	}

	user, err := s.userRepository.FindByUsername(ctx, email)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to find user")
	}

	if user == nil {
		return nil, apperror.New(http.StatusUnauthorized, "invalid email or password")
	}

	if !security.CheckPassword(user.PasswordHash, password) {
		return nil, apperror.New(http.StatusUnauthorized, "invalid email or password")
	}

	token, expiresAt, err := s.jwtServicce.GenerateToken(user.ID)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to generate token")
	}

	return &dto.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
	}, nil
}
