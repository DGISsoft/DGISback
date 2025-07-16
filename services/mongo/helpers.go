package mongo

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ObjectIDFromString(s string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(s)
}

func NewObjectID() primitive.ObjectID {
	return primitive.NewObjectID()
}

func IsValidObjectID(s string) bool {
	_, err := primitive.ObjectIDFromHex(s)
	return err == nil
}