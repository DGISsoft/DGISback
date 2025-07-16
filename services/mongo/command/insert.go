package command

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

func InsertOne[T any](ctx context.Context, collection *mongo.Collection, document T) (*mongo.InsertOneResult, error) {
	return collection.InsertOne(ctx, document)
}

func InsertMany[T any](ctx context.Context, collection *mongo.Collection, documents []T) (*mongo.InsertManyResult, error) {
	docs := make([]interface{}, len(documents))
	for i, doc := range documents {
		docs[i] = doc
	}
	return collection.InsertMany(ctx, docs)
}