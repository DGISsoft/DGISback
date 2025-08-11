package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/DGISsoft/DGISback/models"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// JWTClaims кастомные claims для JWT токена
type JWTClaims struct {
	UserID   string          `json:"user_id"`
	Login    string          `json:"login"`
	Role     models.UserRole `json:"role"`
	FullName string          `json:"full_name"`
	jwt.RegisteredClaims
}

// JWTManager менеджер для работы с JWT токенами
type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
}

// NewJWTManager создает новый JWT менеджер
func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{secretKey: secretKey, tokenDuration: tokenDuration}
}

// GenerateToken генерирует новый JWT токен для пользователя
func (manager *JWTManager) GenerateToken(user *models.User) (string, error) {
	// Устанавливаем время истечения токена
	expirationTime := time.Now().Add(manager.tokenDuration)

	// Создаем claims
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

	// Создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен
	return token.SignedString([]byte(manager.secretKey))
}

// VerifyToken проверяет JWT токен и возвращает claims
func (manager *JWTManager) VerifyToken(tokenString string) (*JWTClaims, error) {
	// Парсим токен
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		
		return []byte(manager.secretKey), nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	// Проверяем валидность токена
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	
	// Проверяем время истечения
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpiredToken
	}
	
	return claims, nil
}

// GetSecretKey возвращает секретный ключ из переменных окружения
func GetSecretKey() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// В production всегда должен использоваться надежный секретный ключ
		secret = "default-secret-key-change-in-production"
	}
	return secret
}

// GetTokenDuration возвращает продолжительность жизни токена из переменных окружения
func GetTokenDuration() time.Duration {
	// По умолчанию токен действует 24 часа
	duration := 24 * time.Hour
	
	// Можно переопределить через переменную окружения
	if durationStr := os.Getenv("JWT_DURATION"); durationStr != "" {
		if d, err := time.ParseDuration(durationStr); err == nil {
			duration = d
		}
	}	
	return duration
}