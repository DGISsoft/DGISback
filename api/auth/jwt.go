package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/DGISsoft/DGISback/env"
	"github.com/DGISsoft/DGISback/models"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type JWTClaims struct {
	UserID   string          `json:"user_id"`
	Login    string          `json:"login"`
	Role     models.UserRole `json:"role"`
	FullName string          `json:"full_name"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
}

func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{secretKey: secretKey, tokenDuration: tokenDuration}
}

func (manager *JWTManager) GenerateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(manager.tokenDuration)

	claims := JWTClaims{
		UserID:   user.ID.Hex(),
		Login:    user.Login,
		Role:     user.Role,
		FullName: user.FullName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(manager.secretKey))
}

func (manager *JWTManager) VerifyToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		
		return []byte(manager.secretKey), nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpiredToken
	}
	
	return claims, nil
}

func GetSecretKey() string {
	secret := env.GetEnv("JWT_SECRET","default-secret-key-change-in-production")
	return secret
}


func GetTokenDuration() time.Duration {
	duration := 24 * time.Hour
	
	if durationStr := env.GetEnv("JWT_DURATION",""); durationStr != "" {
		if d, err := time.ParseDuration(durationStr); err == nil {
			duration = d
		}
	}	
	return duration
}