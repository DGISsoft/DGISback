// middleware/auth.go
package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/DGISsoft/DGISback/api/auth"
)

// contextKey - приватный тип для ключей контекста.
type contextKey struct {
	name string
}

// AuthCookieName - имя HttpOnly cookie для аутентификации.
const AuthCookieName = "auth-token"

// AuthContext хранит информацию о состоянии аутентификации.
type AuthContext struct {
	User *auth.JWTClaims // Информация о пользователе, если аутентифицирован.
}

// Ключи для контекста
var (
	authContextKey     = &contextKey{"authContext"}
	responseWriterKey  = &contextKey{"responseWriter"}
)

// AuthResponseWriterWrapper оборачивает http.ResponseWriter.
type AuthResponseWriterWrapper struct {
	http.ResponseWriter
}

// GetUserFromContext извлекает информацию о пользователе из контекста.
func GetUserFromContext(ctx context.Context) (*auth.JWTClaims, bool) {
	if ac, ok := ctx.Value(authContextKey).(*AuthContext); ok && ac.User != nil {
		return ac.User, true
	}
	return nil, false
}

// GetResponseWriterFromContext извлекает http.ResponseWriter из контекста.
func GetResponseWriterFromContext(ctx context.Context) (http.ResponseWriter, error) {
	if w, ok := ctx.Value(responseWriterKey).(*AuthResponseWriterWrapper); ok {
		return w.ResponseWriter, nil
	}
	if w, ok := ctx.Value(responseWriterKey).(http.ResponseWriter); ok {
		return w, nil
	}
	return nil, fmt.Errorf("http.ResponseWriter not found in context")
}

// SignalSetAuthCookieDirect немедленно устанавливает cookie.
func SignalSetAuthCookieDirect(w http.ResponseWriter, tokenString string) {
	tokenDuration := auth.GetTokenDuration()
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(tokenDuration.Seconds()),
	})
	log.Printf("Auth: Set cookie, token length: %d", len(tokenString))
}

// SignalClearAuthCookieDirect немедленно очищает cookie.
func SignalClearAuthCookieDirect(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	log.Println("Auth: Cleared cookie")
}

// AuthMiddleware проверяет JWT токен из Cookie.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Auth: Processing %s %s", r.Method, r.URL.Path)

		authCtx := &AuthContext{}

		// Получаем токен из Cookie
		var tokenString string
		if cookie, err := r.Cookie(AuthCookieName); err == nil {
			tokenString = cookie.Value
			log.Printf("Auth: Found cookie, length: %d", len(tokenString))
		} else {
			log.Printf("Auth: No cookie found: %v", err)
		}

		// Проверяем токен
		if tokenString != "" {
			jwtManager := auth.NewJWTManager(auth.GetSecretKey(), auth.GetTokenDuration())
			if claims, err := jwtManager.VerifyToken(tokenString); err == nil {
				log.Println("Auth: Token verified")
				authCtx.User = claims
			} else {
				log.Printf("Auth: Invalid token: %v", err)
			}
		} else {
			log.Println("Auth: No token to verify")
		}

		// Кладем AuthContext и ResponseWriter в контекст
		ctxWithAuth := context.WithValue(r.Context(), authContextKey, authCtx)
		wrappedW := &AuthResponseWriterWrapper{ResponseWriter: w}
		ctxWithEverything := context.WithValue(ctxWithAuth, responseWriterKey, wrappedW)
		rWithAuth := r.WithContext(ctxWithEverything)

		// Передаем управление следующему обработчику
		log.Println("Auth: Calling next handler")
		next.ServeHTTP(w, rWithAuth) // Передаем оригинальный ResponseWriter
		log.Println("Auth: Handler finished")
	})
}