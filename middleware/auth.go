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
const AuthCookieName = "auth-token" // Имя вашей аутентификационной куки

// AuthMiddleware проверяет JWT токен из заголовка Authorization или из Cookie и добавляет информацию о пользователе в контекст
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var tokenString string

        // 1. Сначала пробуем получить токен из заголовка Authorization (для обратной совместимости или API вызовов)
        authHeader := r.Header.Get("Authorization")
        if authHeader != "" {
            tokenString = strings.TrimPrefix(authHeader, "Bearer ")
            if tokenString == authHeader {
                tokenString = "" // Не в правильном формате
            }
        }

        // 2. Если в заголовке нет токена, пробуем получить его из Cookie
        if tokenString == "" {
            if cookie, err := r.Cookie(AuthCookieName); err == nil {
                tokenString = cookie.Value
            }
        }

        // 3. Если токен так и не найден, продолжаем без аутентификации
        if tokenString == "" {
            next.ServeHTTP(w, r)
            return
        }

        // 4. Создаем JWT менеджер
        jwtManager := auth.NewJWTManager(auth.GetSecretKey(), auth.GetTokenDuration())

        // 5. Проверяем токен
        claims, err := jwtManager.VerifyToken(tokenString)
        if err != nil {
            // Если токен недействителен, продолжаем без аутентификации
            // Можно добавить логирование ошибки для отладки
            // log.Printf("Invalid token: %v", err)
            next.ServeHTTP(w, r)
            return
        }

        // 6. Добавляем информацию о пользователе в контекст
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