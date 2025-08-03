// middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "user"

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            next.ServeHTTP(w, r)
            return
        }
        
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            next.ServeHTTP(w, r)
            return
        }
        

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            // Верните ваш секретный ключ
            return []byte("your-secret-key"), nil
        })
        
        if err != nil || !token.Valid {
            next.ServeHTTP(w, r)
            return
        }
        

        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            ctx := context.WithValue(r.Context(), UserContextKey, claims)
            r = r.WithContext(ctx)
        }
        
        next.ServeHTTP(w, r)
    })
}