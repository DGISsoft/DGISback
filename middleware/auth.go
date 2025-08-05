// middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/DGISsoft/DGISback/api/auth"
)

type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware проверяет JWT токен и добавляет информацию о пользователе в контекст
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            // Если нет заголовка авторизации, продолжаем без аутентификации
            next.ServeHTTP(w, r)
            return
        }
        
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            // Если заголовок не в правильном формате, продолжаем без аутентификации
            next.ServeHTTP(w, r)
            return
        }
        
        // Создаем JWT менеджер
        jwtManager := auth.NewJWTManager(auth.GetSecretKey(), auth.GetTokenDuration())
        
        // Проверяем токен
        claims, err := jwtManager.VerifyToken(tokenString)
        if err != nil {
            // Если токен недействителен, продолжаем без аутентификации
            next.ServeHTTP(w, r)
            return
        }
        
        // Добавляем информацию о пользователе в контекст
        ctx := context.WithValue(r.Context(), UserContextKey, claims)
        r = r.WithContext(ctx)
        
        next.ServeHTTP(w, r)
    })
}

// GetUserFromContext извлекает информацию о пользователе из контекста
func GetUserFromContext(ctx context.Context) (*auth.JWTClaims, bool) {
    claims, ok := ctx.Value(UserContextKey).(*auth.JWTClaims)
    return claims, ok
}