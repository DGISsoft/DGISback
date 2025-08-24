package service

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/DGISsoft/DGISback/env"
	"github.com/DGISsoft/DGISback/models"
	"github.com/DGISsoft/DGISback/services/mongo/command"
	red "github.com/DGISsoft/DGISback/services/redis"
	"github.com/DGISsoft/DGISback/services/s3"
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
        Role:        models.UserRoleStarosta,
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


func TestRedis(t *testing.T) {

	redisHost := env.GetEnv("REDIS_HOST", "")
	redisClient := red.NewRedisClient(redisHost, "", 0)
	if redisClient == nil {
		t.Fatal("Redis client is nil")
	}
	redisService := red.NewRedisService(redisClient)
	redisService.SetValue("test_key", "test_value")
	value, err := redisService.GetValue("test_key")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(value)
	redisService.DeleteValue("test_key")
}


func TestS3UploadAndDeleteFile(t *testing.T) {
	s3cfg := &s3.S3ClientConfig{
		Bucket:    "json-usa-vacancies",
		Endpoint:  env.GetEnv("AWS_ENDPOINT", ""),
		Region:    env.GetEnv("AWS_REGION", ""),
		AccessKey: env.GetEnv("AWS_ACCESS_KEY_ID", ""),
		SecretKey: env.GetEnv("AWS_SECRET_ACCESS_KEY", ""),
	}

	key := "resume/test_file.json"
	content := []byte("hello from test")
	contentType := "text/plain"

	service, err := s3.NewS3Service(s3cfg)
	assert.NoError(t, err, "create S3 service")

	assert.NoError(t, err, "config load error")

	err = service.UploadFile(context.TODO(), s3cfg.Bucket, key, content, contentType)
	assert.NoError(t, err, "upload file error")

	exists, err := service.FileExists(context.TODO(), s3cfg.Bucket, key)
	assert.NoError(t, err, "file exists check error")
	assert.True(t, exists, "file should exist")

	err = service.DeleteFile(context.TODO(), s3cfg.Bucket, key)
	assert.NoError(t, err, "delete file error")

	exists, err = service.FileExists(context.TODO(), s3cfg.Bucket, key)
	assert.NoError(t, err, "file exists after delete check error")
	assert.False(t, exists, "file should not exist after deletion")
}