// middleware/auth.go
package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/DGISsoft/DGISback/api/auth"
)

// contextKey - приватный тип для ключей контекста, чтобы избежать коллизий.
// Изменено на struct{}, как в примере из документации gqlgen.
type contextKey struct {
	name string
}

// UserContextKey - ключ для хранения информации о пользователе в контексте.
// const UserContextKey contextKey = "user" // Убираем это, так как contextKey теперь struct

// AuthCookieName - имя HttpOnly cookie для аутентификации.
const AuthCookieName = "auth-token"

// AuthContext хранит информацию о состоянии аутентификации для текущего запроса.
// Используется для передачи данных между middleware и resolvers.
type AuthContext struct {
	// User содержит информацию о пользователе, если запрос аутентифицирован.
	User *auth.JWTClaims

	// Поля для сигнализации необходимости установки/очистки cookie.
	// Эти поля будут изменяться в resolvers.
	ShouldSetCookie   bool
	TokenToSet        string
	ShouldClearCookie bool
}

// authContextKey - приватный ключ для хранения AuthContext в контексте.
// Используем struct{} как в официальном примере gqlgen.
var authContextKey = &contextKey{"authContext"}

// GetAuthContext извлекает AuthContext из контекста.
// Если контекст не найден, возвращает пустой AuthContext.
func GetAuthContext(ctx context.Context) *AuthContext {
	if ac, ok := ctx.Value(authContextKey).(*AuthContext); ok {
		return ac
	}
	return &AuthContext{} // Возвращаем пустой контекст, если не найден
}

// SignalSetAuthCookie указывает AuthMiddleware установить cookie после выполнения запроса.
// Вызывается из Login resolver.
func SignalSetAuthCookie(ctx context.Context, tokenString string) {
	ac := GetAuthContext(ctx)
	ac.ShouldSetCookie = true
	ac.TokenToSet = tokenString
	// Убедимся, что не пытаемся одновременно установить и удалить
	ac.ShouldClearCookie = false
	// Нет необходимости обновлять контекст через WithValue,
	// так как мы изменяем поля структуры, на которую ссылается указатель.
}

// SignalClearAuthCookie указывает AuthMiddleware удалить cookie после выполнения запроса.
// Вызывается из Logout resolver.
func SignalClearAuthCookie(ctx context.Context) {
	ac := GetAuthContext(ctx)
	ac.ShouldClearCookie = true
	// Убедимся, что не пытаемся одновременно установить и удалить
	ac.ShouldSetCookie = false
	ac.TokenToSet = ""
	// Нет необходимости обновлять контекст через WithValue.
}

// AuthMiddleware проверяет JWT токен из Cookie или заголовка Authorization
// и управляет HttpOnly cookie для аутентификации.
// AuthMiddleware проверяет JWT токен из Cookie или заголовка Authorization
// и управляет HttpOnly cookie для аутентификации.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("AuthMiddleware: [1] Incoming request for %s %s", r.Method, r.URL.Path) // Лог
		// 1. Инициализируем AuthContext для этого запроса
		authCtx := &AuthContext{}
		log.Printf("AuthMiddleware: [2] Created new AuthContext at %p", authCtx) // Лог

		// 2. Пробуем получить токен из Cookie
		var tokenString string
		if cookie, err := r.Cookie(AuthCookieName); err == nil {
			tokenString = cookie.Value
			log.Printf("AuthMiddleware: [3] Found auth cookie, value length: %d", len(tokenString)) // Лог
		} else {
			 log.Println("AuthMiddleware: [3] No auth cookie found in request") // Лог
		}

		// 3. Fallback: пробуем получить токен из заголовка Authorization (Bearer)
		// (для обратной совместимости или API вызовов)
		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
				if tokenString == authHeader {
					tokenString = "" // Не в правильном формате
				}
				if tokenString != "" {
					log.Printf("AuthMiddleware: [3b] Found auth token in Authorization header, length: %d", len(tokenString)) // Лог
				}
			}
		}

		// 4. Если токен найден, проверяем его
		if tokenString != "" {
			jwtManager := auth.NewJWTManager(auth.GetSecretKey(), auth.GetTokenDuration())
			if claims, err := jwtManager.VerifyToken(tokenString); err == nil {
				log.Println("AuthMiddleware: [4] Token verified successfully") // Лог
				authCtx.User = claims
			} else {
				log.Printf("AuthMiddleware: [4] Invalid or expired token: %v", err)
			}
		} else {
			 log.Println("AuthMiddleware: [4] No token found to verify") // Лог
		}

		// 5. Кладем AuthContext в контекст запроса
		ctxWithAuth := context.WithValue(r.Context(), authContextKey, authCtx)
		log.Printf("AuthMiddleware: [5] Put AuthContext (%p) into context with key %v", authCtx, authContextKey) // Лог
		rWithAuth := r.WithContext(ctxWithAuth)
		log.Printf("AuthMiddleware: [6] Request context replaced. New context contains AuthContext: %t", GetAuthContext(rWithAuth.Context()) == authCtx) // Лог

		// 6. Передаем управление следующему обработчику (srv)
		// Это блокирующий вызов, который выполняет весь GraphQL-запрос,
		// включая все resolvers. Именно здесь resolvers могут вызвать
		// SignalSetAuthCookie или SignalClearAuthCookie.
		log.Println("AuthMiddleware: [7] Calling next.ServeHTTP (GraphQL handler)") // Лог
		next.ServeHTTP(w, rWithAuth)
		log.Println("AuthMiddleware: [8] Returned from next.ServeHTTP (GraphQL handler)") // Лог

		// --- ВАЖНО: Код ниже выполняется ПОСЛЕ завершения next.ServeHTTP ---
		// Здесь мы можем получить обновленный AuthContext из контекста,
		// если resolver'ы вызвали Signal функции.

		// 7. Получаем потенциально обновленный AuthContext
		// Используем контекст из rWithAuth, так как именно он мог быть изменен
		// (хотя контекст http.Request не изменяется, структура AuthContext по указателю - изменяется).
		finalAuthCtx := GetAuthContext(rWithAuth.Context())
		log.Printf("AuthMiddleware: [9] Retrieved final AuthContext (%p) from request context", finalAuthCtx) // Лог
		log.Printf("AuthMiddleware: [9b] Final AuthContext details - User: %v, ShouldSet: %v, TokenToSet: '%.10s...', ShouldClear: %v", finalAuthCtx.User != nil, finalAuthCtx.ShouldSetCookie, finalAuthCtx.TokenToSet, finalAuthCtx.ShouldClearCookie) // Лог

		// 8. Выполняем действия с cookie на основе сигналов
		if finalAuthCtx.ShouldClearCookie {
			log.Println("AuthMiddleware: [10] Clearing auth cookie") // Лог
			http.SetCookie(w, &http.Cookie{
				Name:     AuthCookieName,
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   -1,
				Domain:   "localhost",
			})
		} else if finalAuthCtx.ShouldSetCookie && finalAuthCtx.TokenToSet != "" {
			log.Printf("AuthMiddleware: [10] Setting auth cookie, token length: %d", len(finalAuthCtx.TokenToSet)) // Лог
			http.SetCookie(w, &http.Cookie{
				Name:     AuthCookieName,
				Value:    finalAuthCtx.TokenToSet,
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteLaxMode,
				Domain:   "localhost",
				MaxAge:   int(auth.GetTokenDuration().Seconds()),
				Expires:  time.Now().Add(auth.GetTokenDuration()),
			})
		} else {
			log.Println("AuthMiddleware: [10] No cookie action needed after request") // Лог
		}
		log.Printf("AuthMiddleware: [11] Finished processing request for %s %s", r.Method, r.URL.Path) // Лог
	})
}

// GetUserFromContext извлекает информацию о пользователе из контекста.
// Возвращает информацию о пользователе и флаг, указывающий, был ли пользователь аутентифицирован.
func GetUserFromContext(ctx context.Context) (*auth.JWTClaims, bool) {
	// Используем GetAuthContext для получения пользователя
	ac := GetAuthContext(ctx)
	if ac.User != nil {
		return ac.User, true
	}
	return nil, false
}