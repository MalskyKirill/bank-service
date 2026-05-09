package security

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secret []byte
	ttl    time.Duration
}

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func NewJWTService(secret string, ttlHours int) *JWTService {
	return &JWTService{
		secret: []byte(secret),
		ttl:    time.Duration(ttlHours) * time.Hour,
	}
}

func (s *JWTService) GenerateToken(userID int64) (string, time.Time, error) {
	now := time.Now()

	expiresAt := now.Add(s.ttl)

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}
