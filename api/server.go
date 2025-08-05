package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/DGISsoft/DGISback/api/graph"
	serv "github.com/DGISsoft/DGISback/services/mongo"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/ast"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)


const defaultPort = "8080"

func main() {
	//redisHost := env.GetEnv("REDIS", "")
	//redisDBStr := env.GetEnv("REDIS_DB", "")
	//redisDB, err := strconv.Atoi(redisDBStr)
	// if err != nil {
	// 	log.Fatalf("Invalid REDIS_DB value: %v", err)
	// }
    client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017")) // Ваш URI
    if err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(context.TODO()) // Не забудьте defer

    // 2. Получение базы данных
    database := client.Database("dgis-db") // Ваше имя БД

    // 3. Создание MongoService
    mongoService := serv.New(database) // <-- ВАЖНО: Передаем *mongo.Database

    // 4. Создание UserService, передавая ему MongoService
    userService := serv.NewUserService(mongoService) // <-- ВАЖНО: Передаем *MongoService

    // 5. Создание Resolver с UserService
    resolver := &graph.Resolver{
        UserService: userService, // <-- ВАЖНО: Передаем *UserService
    }
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: false,
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
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
	muxGraphql.Handle("/query", c.Handler(srv))

	go func() {
		log.Printf("Starting GraphQL server on :%s", port)
		if err := http.ListenAndServe(":"+port, muxGraphql); err != nil {
			log.Fatalf("GraphQL server error: %v", err)
		}
	}()
	
	select {}
}