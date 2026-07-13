package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/tscommunication/ts-cloud/internal/config"
	"github.com/tscommunication/ts-cloud/internal/models"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(user *models.User) (string, error) {

	cfg := config.Load()

	if cfg.JWTSecret == "" {
		return "", errors.New("JWT_SECRET is not configured")
	}

	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(cfg.JWTSecret))
}

func ValidateToken(tokenString string) (*Claims, error) {

	cfg := config.Load()

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
