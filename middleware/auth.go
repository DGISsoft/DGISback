// middleware/auth.go
package middleware

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/DGISsoft/DGISback/api/auth"
)


type contextKey struct {
	name string
}

const AuthCookieName = "auth-token"

type AuthContext struct {
	User *auth.JWTClaims 
}

var (
	authContextKey     = &contextKey{"authContext"}
	responseWriterKey  = &contextKey{"responseWriter"}
)

type AuthResponseWriterWrapper struct {
	http.ResponseWriter
}

func GetUserFromContext(ctx context.Context) (*auth.JWTClaims, bool) {
	if ac, ok := ctx.Value(authContextKey).(*AuthContext); ok && ac.User != nil {
		return ac.User, true
	}
	return nil, false
}

func GetResponseWriterFromContext(ctx context.Context) (http.ResponseWriter, error) {
	if w, ok := ctx.Value(responseWriterKey).(*AuthResponseWriterWrapper); ok {
		return w.ResponseWriter, nil
	}
	if w, ok := ctx.Value(responseWriterKey).(http.ResponseWriter); ok {
		return w, nil
	}
	return nil, fmt.Errorf("http.ResponseWriter not found in context")
}

// GetUploadedFileFromContext извлекает данные загруженного файла из контекста
func GetUploadedFileFromContext(ctx context.Context) ([]byte, bool) {
	if fileData, ok := ctx.Value("uploadedFile").([]byte); ok && fileData != nil {
		return fileData, true
	}
	return nil, false
}

// GetUploadedFilenameFromContext извлекает имя загруженного файла из контекста
func GetUploadedFilenameFromContext(ctx context.Context) (string, bool) {
	if filename, ok := ctx.Value("filename").(string); ok && filename != "" {
		return filename, true
	}
	return "", false
}

// GetUploadedContentTypeFromContext извлекает тип содержимого загруженного файла из контекста
func GetUploadedContentTypeFromContext(ctx context.Context) (string, bool) {
	if contentType, ok := ctx.Value("contentType").(string); ok && contentType != "" {
		return contentType, true
	}
	return "", false
}

func SignalSetAuthCookieDirect(w http.ResponseWriter, tokenString string) {
	tokenDuration := auth.GetTokenDuration()
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   int(tokenDuration.Seconds()),
	})
	log.Printf("Auth: Set cookie, token length: %d", len(tokenString))
}

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

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Auth: Processing %s %s", r.Method, r.URL.Path)

		authCtx := &AuthContext{}

		var tokenString string
		if cookie, err := r.Cookie(AuthCookieName); err == nil {
			tokenString = cookie.Value
			log.Printf("Auth: Found cookie, length: %d", len(tokenString))
		} else {
			log.Printf("Auth: No cookie found: %v", err)
		}

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

		ctxWithAuth := context.WithValue(r.Context(), authContextKey, authCtx)
		wrappedW := &AuthResponseWriterWrapper{ResponseWriter: w}
		ctxWithEverything := context.WithValue(ctxWithAuth, responseWriterKey, wrappedW)
		rWithAuth := r.WithContext(ctxWithEverything)

		log.Println("Auth: Calling next handler")
		next.ServeHTTP(w, rWithAuth)
		log.Println("Auth: Handler finished")
	})
}

func UploadMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, является ли запрос multipart/form-data
		contentType := r.Header.Get("Content-Type")
		if r.Method == "POST" && contentType != "" && len(contentType) >= 9 && 
		   contentType[:9] == "multipart" {
			
			// Увеличиваем максимальный размер загружаемых данных
			err := r.ParseMultipartForm(32 << 20) // 32MB
			if err != nil {
				http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
				return
			}

			// Ищем файл в форме
			var file multipart.File
			var fileHeader *multipart.FileHeader
			
			// Пробуем найти файл по разным ключам
			for _, key := range []string{"file", "upload", "image"} {
				file, fileHeader, err = r.FormFile(key)
				if err == nil && file != nil {
					defer file.Close()
					
					// Читаем содержимое файла
					fileData, err := io.ReadAll(file)
					if err != nil {
						http.Error(w, "Failed to read file", http.StatusBadRequest)
						return
					}
					
					// Добавляем данные файла в контекст
					ctx := context.WithValue(r.Context(), "uploadedFile", fileData)
					ctx = context.WithValue(ctx, "filename", fileHeader.Filename)
					ctx = context.WithValue(ctx, "contentType", fileHeader.Header.Get("Content-Type"))
					
					r = r.WithContext(ctx)
					break
				}
			}
		}
		
		next.ServeHTTP(w, r)
	})
}