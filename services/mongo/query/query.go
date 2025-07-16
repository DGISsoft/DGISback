package query

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func FindOne[T any](ctx context.Context, collection *mongo.Collection, filter bson.M, result *T) error {
	return collection.FindOne(ctx, filter).Decode(result)
}

func FindByID[T any](ctx context.Context, collection *mongo.Collection, id primitive.ObjectID, result *T) error {
	filter := bson.M{"_id": id}
	return collection.FindOne(ctx, filter).Decode(result)
}

func FindMany[T any](ctx context.Context, collection *mongo.Collection, filter bson.M, results *[]T, opts ...*options.FindOptions) error {
	var findOptions *options.FindOptions
	if len(opts) > 0 {
		findOptions = opts[0]
	}
	
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	
	return cursor.All(ctx, results)
}

func FindWithPagination[T any](ctx context.Context, collection *mongo.Collection, filter bson.M, results *[]T, limit, skip int64) error {
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
	}
	if skip > 0 {
		opts.SetSkip(skip)
	}
	opts.SetSort(bson.D{{Key: "_id", Value: -1}})
	
	return FindMany(ctx, collection, filter, results, opts)
}

func Count(ctx context.Context, collection *mongo.Collection, filter bson.M) (int64, error) {
	return collection.CountDocuments(ctx, filter)
}

func Exists(ctx context.Context, collection *mongo.Collection, filter bson.M) (bool, error) {
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func Aggregate[T any](ctx context.Context, collection *mongo.Collection, pipeline []bson.M, results *[]T) error {
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	
	return cursor.All(ctx, results)
}