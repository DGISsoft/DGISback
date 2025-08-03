package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/ast"
)


const defaultPort = "8080"

func main() {
	//redisHost := env.GetEnv("REDIS", "")
	//redisDBStr := env.GetEnv("REDIS_DB", "")
	//redisDB, err := strconv.Atoi(redisDBStr)
	// if err != nil {
	// 	log.Fatalf("Invalid REDIS_DB value: %v", err)
	// }
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

	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))

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