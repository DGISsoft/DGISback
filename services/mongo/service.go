package mongo

import (
	"go.mongodb.org/mongo-driver/mongo"
)
type UserService struct {
    *MongoService
}

type MongoService struct {
	db *mongo.Database
}

func New(db *mongo.Database) *MongoService {
	return &MongoService{db: db}
}

func (s *MongoService) GetDatabase() *mongo.Database {
	return s.db
}

func (s *MongoService) GetCollection(name string) *mongo.Collection {
	return s.db.Collection(name)
}