package service

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/DGISsoft/DGISback/models"
	"github.com/DGISsoft/DGISback/services/mongo/command"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongo(t *testing.T) {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("dgis-db")
	collection := db.Collection("users")


	building := "6.1"
	user := models.User{
		ID:          primitive.NewObjectID(),
        Login:       "testuser123",
        Password:    "testpassword",
        Role:        models.RoleStarosta,
        FullName:    "Тестовый Пользователь",
        Building:    &building,
        PhoneNumber: "+79991234567",
        TelegramTag: "@testuser",
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
	}

	insertResult, err := command.InsertOne(ctx, collection, user)
    assert.NoError(t, err)
    assert.NotNil(t, insertResult.InsertedID)

	var insertedUser models.User
    err = collection.FindOne(ctx, primitive.M{"_id": user.ID}).Decode(&insertedUser)
    assert.NoError(t, err)
    assert.Equal(t, user.Login, insertedUser.Login)


	deleteResult, err := command.DeleteByID(ctx, collection, user.ID)
    assert.NoError(t, err)
    assert.Equal(t, int64(1), deleteResult.DeletedCount)


	err = collection.FindOne(ctx, primitive.M{"_id": user.ID}).Decode(&insertedUser)
    assert.Error(t, err)
    assert.Equal(t, mongo.ErrNoDocuments, err)
}