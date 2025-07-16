// services/mongo/test/service_test.go
package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mongo"
	"mongo/command"
	"mongo/query"
)

type TestDoc struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Name string             `bson:"name"`
	Age  int                `bson:"age"`
}

func setupTest(t *testing.T) (*mongo.MongoService, *mongodriver.Collection, func()) {
	ctx := context.Background()
	
	client, err := mongodriver.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	
	testDB := client.Database("test_db")
	service := mongo.New(testDB)
	collection := service.GetCollection("test_collection")
	
	// Очистка
	collection.Drop(ctx)
	
	cleanup := func() {
		collection.Drop(ctx)
		client.Disconnect(ctx)
	}
	
	return service, collection, cleanup
}

func TestInsertAndFind(t *testing.T) {
	_, collection, cleanup := setupTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Вставка
	doc := TestDoc{Name: "John", Age: 30}
	result, err := command.InsertOne(ctx, collection, doc)
	require.NoError(t, err)
	assert.NotNil(t, result.InsertedID)
	
	// Поиск
	var found TestDoc
	err = query.FindByID(ctx, collection, result.InsertedID.(primitive.ObjectID), &found)
	require.NoError(t, err)
	assert.Equal(t, "John", found.Name)
	assert.Equal(t, 30, found.Age)
}

func TestUpdateAndDelete(t *testing.T) {
	_, collection, cleanup := setupTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Вставка
	doc := TestDoc{Name: "Jane", Age: 25}
	result, err := command.InsertOne(ctx, collection, doc)
	require.NoError(t, err)
	
	id := result.InsertedID.(primitive.ObjectID)
	
	// Обновление
	update := command.NewUpdateBuilder().Set("age", 26).Build()
	updateResult, err := command.UpdateByID(ctx, collection, id, update)
	require.NoError(t, err)
	assert.Equal(t, int64(1), updateResult.ModifiedCount)
	
	// Удаление
	deleteResult, err := command.DeleteByID(ctx, collection, id)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleteResult.DeletedCount)
}

func TestQueryBuilder(t *testing.T) {
	_, collection, cleanup := setupTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Вставка тестовых данных
	docs := []TestDoc{
		{Name: "Alice", Age: 20},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 40},
	}
	
	_, err := command.InsertMany(ctx, collection, docs)
	require.NoError(t, err)
	
	// Поиск с билдером
	filter := query.NewBuilder().WhereGT("age", 25).Build()
	
	var results []TestDoc
	err = query.FindMany(ctx, collection, filter, &results)
	require.NoError(t, err)
	
	assert.Len(t, results, 2) // Bob и Charlie
}

func TestGetDocumentID(t *testing.T) {
	doc := TestDoc{
		ID:   primitive.NewObjectID(),
		Name: "Test",
		Age:  25,
	}
	
	id, err := command.GetDocumentID(doc)
	require.NoError(t, err)
	assert.Equal(t, doc.ID, id)
	
	// Тест с указателем
	id2, err := command.GetDocumentID(&doc)
	require.NoError(t, err)
	assert.Equal(t, doc.ID, id2)
}

func TestCount(t *testing.T) {
	_, collection, cleanup := setupTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Вставка тестовых данных
	docs := []TestDoc{
		{Name: "User1", Age: 20},
		{Name: "User2", Age: 30},
		{Name: "User3", Age: 40},
	}
	
	_, err := command.InsertMany(ctx, collection, docs)
	require.NoError(t, err)
	
	// Подсчет всех документов
	count, err := query.Count(ctx, collection, bson.M{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
	
	// Подсчет с фильтром
	count, err = query.Count(ctx, collection, bson.M{"age": bson.M{"$gt": 25}})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestExists(t *testing.T) {
	_, collection, cleanup := setupTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Проверка несуществующего документа
	exists, err := query.Exists(ctx, collection, bson.M{"name": "NonExistent"})
	require.NoError(t, err)
	assert.False(t, exists)
	
	// Вставка документа
	doc := TestDoc{Name: "TestExists", Age: 25}
	_, err = command.InsertOne(ctx, collection, doc)
	require.NoError(t, err)
	
	// Проверка существующего документа
	exists, err = query.Exists(ctx, collection, bson.M{"name": "TestExists"})
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestPagination(t *testing.T) {
	_, collection, cleanup := setupTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Вставка большого количества документов
	docs := make([]TestDoc, 10)
	for i := 0; i < 10; i++ {
		docs[i] = TestDoc{
			Name: fmt.Sprintf("User%d", i),
			Age:  20 + i,
		}
	}
	
	_, err := command.InsertMany(ctx, collection, docs)
	require.NoError(t, err)
	
	// Первая страница (5 элементов)
	var page1 []TestDoc
	err = query.FindWithPagination(ctx, collection, bson.M{}, &page1, 5, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 5)
	
	// Вторая страница (следующие 5 элементов)
	var page2 []TestDoc
	err = query.FindWithPagination(ctx, collection, bson.M{}, &page2, 5, 5)
	require.NoError(t, err)
	assert.Len(t, page2, 5)
	
	// Убеждаемся что страницы разные
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}