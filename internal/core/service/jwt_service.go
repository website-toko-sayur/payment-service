package service

import (
	"payment-service/config"

	"github.com/golang-jwt/jwt/v5"
)

type jwtService struct {
	secretKey string
}

type JwtServiceInterface interface {
	ValidateToken(token string) (*jwt.Token, error)
}

func NewJwtService(cfg *config.Config) JwtServiceInterface {
	return &jwtService{
		secretKey: cfg.App.JwtSecretKey,
	}
}

func (j *jwtService) ValidateToken(encodetoken string) (*jwt.Token, error) {
	return jwt.Parse(encodetoken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}

		return []byte(j.secretKey), nil
	})
}
