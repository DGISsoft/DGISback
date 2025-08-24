package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/DGISsoft/DGISback/api/graph"
	"github.com/DGISsoft/DGISback/env"
	"github.com/DGISsoft/DGISback/middleware"
	"github.com/DGISsoft/DGISback/models"
	serv "github.com/DGISsoft/DGISback/services/mongo"
	"github.com/DGISsoft/DGISback/services/redis"
	"github.com/DGISsoft/DGISback/services/s3"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/ast"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultPort = "8080"

const (
	defaultAdminLogin    = "admin"
	defaultAdminPassword = "admin"
	redisAddr            = "localhost:6379"
	redisPassword        = ""
	redisDB              = 0
)

func createDefaultAdmin(userService *serv.UserService) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	users, err := userService.GetUsers(ctx)
	if err != nil {
		log.Printf("Warning: Failed to check for existing users: %v", err)
		return
	}

	if len(users) > 0 {
		log.Println("Users found in database. Skipping default admin creation.")
		return
	}

	log.Println("No users found. Creating default admin user...")

	adminUser := &models.User{
		Login:        defaultAdminLogin,
		Password:     defaultAdminPassword,
		Role:         models.UserRolePredsedatel,
		FullName:     "Default Administrator",
		PhoneNumber:  "+00000000000",
		TelegramTag:  "@admin",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := userService.CreateUser(ctx, adminUser); err != nil {
		log.Printf("Error: Failed to create default admin user: %v", err)
		return
	}

	log.Printf("Default admin user '%s' created successfully!", defaultAdminLogin)
}


func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] START: %s %s", r.Method, r.URL.Path)
		log.Printf("[HTTP] Headers: Connection=%s, Upgrade=%s, Origin=%s, Sec-WebSocket-Key=%.10s...", 
			r.Header.Get("Connection"), r.Header.Get("Upgrade"), r.Header.Get("Origin"), r.Header.Get("Sec-WebSocket-Key"))


		if r.Header.Get("Connection") == "Upgrade" && r.Header.Get("Upgrade") == "websocket" {
			log.Printf("[HTTP] -> This is a WebSocket Upgrade request")
		}


		next.ServeHTTP(w, r)

		log.Printf("[HTTP] END: %s %s", r.Method, r.URL.Path)
	})
}

func main() {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017")) // Ваш URI
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()

	redis.Init(redisAddr, redisPassword, redisDB)
	if redis.Service == nil {
		log.Fatal("Failed to initialize Redis service")
	}

	database := client.Database("dgis-db")

	mongoService := serv.New(database)
	userService := serv.NewUserService(mongoService)
	markerService := serv.NewMarkerService(mongoService)
	notificationService := serv.NewNotificationService(mongoService, redis.Service)
	reportService := serv.NewReportService(mongoService)

	createDefaultAdmin(userService)

	s3cfg := &s3.S3ClientConfig{
		Bucket:    env.GetEnv("S3_BUCKET", "your-default-bucket"),
		Endpoint:  env.GetEnv("S3_ENDPOINT", ""),
		Region:    env.GetEnv("S3_REGION", ""),
		AccessKey: env.GetEnv("S3_ACCESS_KEY", ""),
		SecretKey: env.GetEnv("S3_SECRET_KEY", ""),
	}


	s3.Init(
		s3cfg.Bucket,
		s3cfg.Endpoint,
		s3cfg.Region,
		s3cfg.AccessKey,
		s3cfg.SecretKey,
	)
	
	// Проверяем, что сервис инициализировался успешно
	if s3.Service == nil {
		log.Fatal("Failed to initialize S3 service")
	}

	resolver := &graph.Resolver{
		UserService:          userService,
		MarkerService:        markerService,
		NotificationService:  notificationService,
		RedisService:         redis.Service,
		ReportService: reportService,
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
	})
	
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	

	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				log.Printf("[WS] Upgrade request from Origin: '%s'", r.Header.Get("Origin"))
				origin := r.Header.Get("Origin")
				if origin == "" {
					log.Println("[WS] Allowing request with empty Origin")
					return true
				}
				allowedOrigins := []string{"http://localhost:5173","file://"}
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						log.Printf("[WS] Allowing request from Origin: %s", origin)
						return true
					}
				}
				log.Printf("[WS] Denying request from Origin: %s", origin)
				return false
			},
		},
	})

	
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})
	
	muxGraphql := http.NewServeMux()
	muxGraphql.Handle("/", playground.Handler("GraphQL playground", "/query"))
	
	muxGraphql.Handle("/query", loggingMiddleware(c.Handler(middleware.AuthMiddleware(srv))))

	log.Printf("Starting GraphQL server on :%s", port)
	if err := http.ListenAndServe(":"+port, muxGraphql); err != nil {
		log.Fatalf("GraphQL server error: %v", err)
	}
}