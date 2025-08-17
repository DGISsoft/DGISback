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
	"github.com/DGISsoft/DGISback/middleware"
	"github.com/DGISsoft/DGISback/models"
	serv "github.com/DGISsoft/DGISback/services/mongo"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/ast"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultPort = "8080"


const (
    defaultAdminLogin    = "admin"
    defaultAdminPassword = "admin"
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
        Login:       defaultAdminLogin,
        Password:    defaultAdminPassword,
        Role:        models.UserRolePredsedatel,
        FullName:    "Default Administrator",
        PhoneNumber: "+00000000000",
        TelegramTag: "@admin",
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }

    if err := userService.CreateUser(ctx, adminUser); err != nil {
        log.Printf("Error: Failed to create default admin user: %v", err)
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


    database := client.Database("dgis-db")


    mongoService := serv.New(database)


    userService := serv.NewUserService(mongoService)
    markerService := serv.NewMarkerService(mongoService)
    notificationService := serv.NewNotificationService(mongoService)
    createDefaultAdmin(userService)


    resolver := &graph.Resolver{
        UserService: userService,
        MarkerService: markerService,
        NotificationService: notificationService,
    }
    port := os.Getenv("PORT")
    if port == "" {
        port = defaultPort
    }

    c := cors.New(cors.Options{
        AllowedOrigins: []string{"http://localhost:5173"},
        AllowCredentials: true,
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"*"},
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
	muxGraphql.Handle("/", playground.Handler("GraphQL playground", "/query"))
	muxGraphql.Handle("/query", c.Handler(middleware.AuthMiddleware(srv)))

    log.Printf("Starting GraphQL server on :%s", port)
    if err := http.ListenAndServe(":"+port, muxGraphql); err != nil {
        log.Fatalf("GraphQL server error: %v", err)
    }
}
