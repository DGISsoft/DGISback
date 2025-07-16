package command

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func DeleteOne(ctx context.Context, collection *mongo.Collection, filter bson.M) (*mongo.DeleteResult, error) {
	return collection.DeleteOne(ctx, filter)
}

func DeleteByID(ctx context.Context, collection *mongo.Collection, id primitive.ObjectID) (*mongo.DeleteResult, error) {
	filter := bson.M{"_id": id}
	return collection.DeleteOne(ctx, filter)
}

func DeleteMany(ctx context.Context, collection *mongo.Collection, filter bson.M) (*mongo.DeleteResult, error) {
	return collection.DeleteMany(ctx, filter)
}