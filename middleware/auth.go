// middleware/auth.go
package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time" // Добавлено для работы с Duration

	"github.com/DGISsoft/DGISback/api/auth"
)

// contextKey - приватный тип для ключей контекста, чтобы избежать коллизий.
type contextKey struct {
	name string
}

// AuthCookieName - имя HttpOnly cookie для аутентификации.
const AuthCookieName = "auth-token"

// AuthContext хранит информацию о состоянии аутентификации для текущего запроса.
type AuthContext struct {
	// User содержит информацию о пользователе, если запрос аутентифицирован.
	User *auth.JWTClaims

	// Поля для сигнализации необходимости установки/очистки cookie.
	ShouldSetCookie   bool
	TokenToSet        string
	ShouldClearCookie bool
}

// authContextKey - приватный ключ для хранения AuthContext в контексте.
var authContextKey = &contextKey{"authContext"}

// GetAuthContextForSignal извлекает AuthContext из контекста.
// Используется внутри Signal функций.
// Возвращает ошибку, если AuthContext не найден в контексте.
func GetAuthContextForSignal(ctx context.Context) (*AuthContext, error) {
	if ac, ok := ctx.Value(authContextKey).(*AuthContext); ok {
		return ac, nil
	}
	// Это критическая ошибка - AuthContext должен быть в контексте,
	// если middleware была применена правильно.
	return nil, fmt.Errorf("auth.AuthContext not found in context. Ensure AuthMiddleware is applied")
}

// SignalSetAuthCookie указывает AuthMiddleware установить cookie после выполнения запроса.
// Вызывается из Login resolver.
// ВАЖНО: Требует, чтобы AuthMiddleware была применена и AuthContext был в контексте.
func SignalSetAuthCookie(ctx context.Context, tokenString string) error {
	ac, err := GetAuthContextForSignal(ctx)
	if err != nil {
		log.Printf("SignalSetAuthCookie: Error getting AuthContext: %v", err)
		return err // Возвращаем ошибку, чтобы resolver мог её обработать
	}
	log.Printf("SignalSetAuthCookie: Found AuthContext at %p", ac) // Для отладки
	ac.ShouldSetCookie = true
	ac.TokenToSet = tokenString
	// Убедимся, что не пытаемся одновременно установить и удалить
	ac.ShouldClearCookie = false
	log.Printf("SignalSetAuthCookie: Set flags on AuthContext %p - ShouldSet: %v", ac, ac.ShouldSetCookie) // Для отладки
	return nil
}

// SignalClearAuthCookie указывает AuthMiddleware удалить cookie после выполнения запроса.
// Вызывается из Logout resolver.
// ВАЖНО: Требует, чтобы AuthMiddleware была применена и AuthContext был в контексте.
func SignalClearAuthCookie(ctx context.Context) error {
	ac, err := GetAuthContextForSignal(ctx)
	if err != nil {
		log.Printf("SignalClearAuthCookie: Error getting AuthContext: %v", err)
		return err
	}
	log.Printf("SignalClearAuthCookie: Found AuthContext at %p", ac) // Для отладки
	ac.ShouldClearCookie = true
	// Убедимся, что не пытаемся одновременно установить и удалить
	ac.ShouldSetCookie = false
	ac.TokenToSet = ""
	log.Printf("SignalClearAuthCookie: Set flags on AuthContext %p - ShouldClear: %v", ac, ac.ShouldClearCookie) // Для отладки
	return nil
}

// AuthMiddleware проверяет JWT токен ТОЛЬКО из Cookie
// и управляет HttpOnly cookie для аутентификации.
// middleware/auth.go

// ... (импорты и определения типов остаются без изменений) ...

// AuthMiddleware проверяет JWT токен ТОЛЬКО из Cookie
// и управляет HttpOnly cookie для аутентификации.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("====> AUTH MIDDLEWARE CALLED FOR: %s %s", r.Method, r.URL.Path) // Для отладки

		// 1. Инициализируем AuthContext для этого запроса
		// СОХРАНЯЕМ ССЫЛКУ на authCtx. Это ключевое изменение.
		authCtx := &AuthContext{}
		log.Printf("AuthMiddleware [BEFORE]: Created AuthContext at %p", authCtx) // Для отладки

		// 2. Пробуем получить токен ТОЛЬКО из Cookie (без fallback)
		var tokenString string
		if cookie, err := r.Cookie(AuthCookieName); err == nil {
			tokenString = cookie.Value
			log.Printf("AuthMiddleware: Found auth cookie, value length: %d", len(tokenString)) // Для отладки
		} else {
			log.Printf("AuthMiddleware: No auth cookie found in request: %v", err) // Для отладки
		}

		// 3. Если токен найден в cookie, проверяем его
		if tokenString != "" {
			jwtManager := auth.NewJWTManager(auth.GetSecretKey(), auth.GetTokenDuration())
			if claims, err := jwtManager.VerifyToken(tokenString); err == nil {
				log.Println("AuthMiddleware: Token verified successfully") // Для отладки
				authCtx.User = claims
			} else {
				log.Printf("AuthMiddleware: Invalid or expired token: %v", err) // Для отладки
				// Опционально: сигнализировать об очистке cookie, если токен невалиден
				// authCtx.ShouldClearCookie = true 
			}
		} else {
			log.Println("AuthMiddleware: No token found in cookie to verify") // Для отладки
		}

		// 4. Кладем AuthContext в контекст запроса
		// ВАЖНО: Мы используем authCtx - ту же самую ссылку, которую сохранили.
		ctxWithAuth := context.WithValue(r.Context(), authContextKey, authCtx)
		log.Printf("AuthMiddleware [BEFORE]: Put AuthContext %p into context with key %v", authCtx, authContextKey) // Для отладки
		// Создаем новый запрос с обновленным контекстом
		rWithAuth := r.WithContext(ctxWithAuth)

		// 5. Передаем управление следующему обработчику (srv)
		// ВАЖНО: Передаем rWithAuth, у которого контекст содержит authCtx
		log.Println("AuthMiddleware: Calling next (GraphQL handler)") // Для отладки
		next.ServeHTTP(w, rWithAuth)
		log.Println("AuthMiddleware: Returned from next (GraphQL handler)") // Для отладки

		// --- ВАЖНО: Код ниже выполняется ПОСЛЕ завершения next.ServeHTTP ---
		// Здесь мы используем СОХРАНЕННУЮ ССЫЛКУ authCtx, а не пытаемся извлечь её снова.
		// Так как authCtx это указатель, любые изменения, сделанные в Signal функциях,
		// будут видны здесь через ту же ссылку.

		log.Printf("AuthMiddleware [AFTER]: Using original AuthContext reference at %p", authCtx) // Для отладки
		log.Printf("AuthMiddleware [AFTER]: State - User: %v, ShouldSet: %v, Token Length: %d, ShouldClear: %v", authCtx.User != nil, authCtx.ShouldSetCookie, len(authCtx.TokenToSet), authCtx.ShouldClearCookie) // Для отладки

		// 6. Выполняем действия с cookie на основе сигналов из authCtx (по ссылке)
		if authCtx.ShouldClearCookie {
			log.Println("AuthMiddleware [AFTER]: CLEARING cookie") // Для отладки
			http.SetCookie(w, &http.Cookie{
				Name:     AuthCookieName,
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // Установите true в production при использовании HTTPS
				SameSite: http.SameSiteStrictMode,
				MaxAge:   -1, // Удаление cookie
				Expires:  time.Unix(0, 0), // Альтернативный способ удаления
			})
		} else if authCtx.ShouldSetCookie && authCtx.TokenToSet != "" {
			log.Printf("AuthMiddleware [AFTER]: SETTING cookie, token length: %d", len(authCtx.TokenToSet)) // Для отладки
			// Получаем длительность токена для установки MaxAge
			tokenDuration := auth.GetTokenDuration()
			http.SetCookie(w, &http.Cookie{
				Name:     AuthCookieName,
				Value:    authCtx.TokenToSet,
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // Установите true в production при использовании HTTPS
				SameSite: http.SameSiteStrictMode,
				MaxAge:   int(tokenDuration.Seconds()), // Устанавливаем срок действия cookie
				// Expires:  time.Now().Add(tokenDuration), // Альтернативный способ установки срока
			})
		} else {
			log.Println("AuthMiddleware [AFTER]: No cookie action taken") // Для отладки
		}
		log.Printf("AuthMiddleware: Finished processing request for %s %s", r.Method, r.URL.Path) // Для отладки
	})
}

// GetUserFromContext извлекает информацию о пользователе из контекста.
// Эта функция по-прежнему извлекает AuthContext из контекста, что корректно.
func GetUserFromContext(ctx context.Context) (*auth.JWTClaims, bool) {
	if ac, ok := ctx.Value(authContextKey).(*AuthContext); ok && ac.User != nil {
		return ac.User, true
	}
	return nil, false
}

// SignalSetAuthCookie и SignalClearAuthCookie остаются БЕЗ ИЗМЕНЕНИЙ
// Они извлекают AuthContext из контекста и модифицируют его поля.
// Так как это указатель, изменения будут видны в AuthMiddleware по сохраненной ссылке.