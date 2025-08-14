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
	"github.com/DGISsoft/DGISback/api/graph"
	"github.com/DGISsoft/DGISback/middleware"
	"github.com/DGISsoft/DGISback/models" // Импортируем models
	serv "github.com/DGISsoft/DGISback/services/mongo"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/ast"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultPort = "8080"

// Константы для первого администратора
const (
    defaultAdminLogin    = "admin"
    defaultAdminPassword = "admin" // ВАЖНО: Менять на сложный пароль в production!
)

// createDefaultAdmin проверяет, существуют ли пользователи, и создает админа, если их нет.
func createDefaultAdmin(userService *serv.UserService) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Попытка получить список пользователей
    users, err := userService.GetUsers(ctx)
    if err != nil {
        log.Printf("Warning: Failed to check for existing users: %v", err)
        // Не останавливаем сервер из-за этой ошибки, но логируем
        return
    }

    // Если пользователи есть, ничего не делаем
    if len(users) > 0 {
        log.Println("Users found in database. Skipping default admin creation.")
        return
    }

    log.Println("No users found. Creating default admin user...")

    // Создаем админа
    adminUser := &models.User{
        Login:       defaultAdminLogin,
        Password:    defaultAdminPassword, // UserService.CreateUser хэширует пароль
        Role:        models.UserRolePredsedatel, // Или любая подходящая роль по умолчанию
        FullName:    "Default Administrator",
        PhoneNumber: "+00000000000",
        TelegramTag: "@admin",
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
        // Building можно оставить пустым или задать значение по умолчанию
    }

    if err := userService.CreateUser(ctx, adminUser); err != nil {
        log.Printf("Error: Failed to create default admin user: %v", err)
        // Опять же, не останавливаем сервер, но логируем критическую ошибку
        // В production, возможно, нужно завершить работу
        return
    }

    log.Printf("Default admin user '%s' created successfully!", defaultAdminLogin)
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

    // 2. Получение базы данных
    database := client.Database("dgis-db") // Ваше имя БД

    // 3. Создание MongoService
    mongoService := serv.New(database) // <-- ВАЖНО: Передаем *mongo.Database

    // 4. Создание UserService, передавая ему MongoService
    userService := serv.NewUserService(mongoService) // <-- ВАЖНО: Передаем *MongoService

    // 5. Создание администратора при необходимости (новая логика)
    createDefaultAdmin(userService)

    // 6. Создание Resolver с UserService
    resolver := &graph.Resolver{
        UserService: userService, // <-- ВАЖНО: Передаем *UserService
    }
    port := os.Getenv("PORT")
    if port == "" {
        port = defaultPort
    }

    c := cors.New(cors.Options{
        // ВАЖНО: Замените "*" на конкретный origin вашего фронтенда в production
        AllowedOrigins: []string{"http://localhost:5173"},
        AllowCredentials: true,
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"Authorization", "Content-Type"},
        // Использование "*" с AllowCredentials = true запрещено спецификацией CORS


    })
    srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

    srv.AddTransport(transport.Options{})
    srv.AddTransport(transport.GET{})
    srv.AddTransport(transport.POST{})

    srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

    srv.Use(extension.Introspection{})
    srv.Use(extension.AutomaticPersistedQuery{
        Cache: lru.New[string](100),
    })
    muxGraphql := http.NewServeMux()

	muxGraphql.Handle("/query", c.Handler(middleware.AuthMiddleware(srv)))

    log.Printf("Starting GraphQL server on :%s", port)
    if err := http.ListenAndServe(":"+port, muxGraphql); err != nil {
        log.Fatalf("GraphQL server error: %v", err)
    }
    // Убираем select {} и запускаем сервер синхронно
}
