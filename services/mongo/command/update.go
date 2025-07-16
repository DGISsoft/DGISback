package command

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func UpdateOne(ctx context.Context, collection *mongo.Collection, filter bson.M, update bson.M) (*mongo.UpdateResult, error) {
	return collection.UpdateOne(ctx, filter, update)
}

func UpdateByID(ctx context.Context, collection *mongo.Collection, id primitive.ObjectID, update bson.M) (*mongo.UpdateResult, error) {
	filter := bson.M{"_id": id}
	return collection.UpdateOne(ctx, filter, update)
}

func UpdateMany(ctx context.Context, collection *mongo.Collection, filter bson.M, update bson.M) (*mongo.UpdateResult, error) {
	return collection.UpdateMany(ctx, filter, update)
}