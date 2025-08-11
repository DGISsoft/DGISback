package mongo

import (
	"context"
	"fmt"

	"github.com/DGISsoft/DGISback/models"
	"github.com/DGISsoft/DGISback/services/mongo/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MarkerService struct {
	*MongoService
}

func NewMarkerService(mongoService *MongoService) *MarkerService {
	return &MarkerService{MongoService: mongoService}
}

func (s *MarkerService) GetMarkerByID(ctx context.Context, id primitive.ObjectID) (*models.Marker, error) {
	collection := s.GetCollection("markers")
	var marker models.Marker

	err := query.FindByID(ctx, collection, id, &marker)
	if err != nil {
		return nil, fmt.Errorf("failed to get marker: %w", err)
	}

	return &marker, nil
}

func (s *MarkerService) GetMarkerByMarkerID(ctx context.Context, markerID string) (*models.Marker, error) {
	collection := s.GetCollection("markers")
	var marker models.Marker
	filter := bson.M{"markerId": markerID}

	err := query.FindOne(ctx, collection, filter, &marker)
	if err != nil {
		return nil, fmt.Errorf("failed to get marker: %w", err)
	}

	return &marker, nil
}

func (s *MarkerService) GetAllMarkers(ctx context.Context) ([]*models.Marker, error) {
	collection := s.GetCollection("markers")
	var markers []*models.Marker

	err := query.FindMany(ctx, collection, bson.M{}, &markers)
	if err != nil {
		return nil, fmt.Errorf("failed to get markers: %w", err)
	}

	return markers, nil
}

func (s *MarkerService) CreateMarker(ctx context.Context, marker *models.Marker) error {
	collection := s.GetCollection("markers")

	_, err := collection.InsertOne(ctx, marker)
	if err != nil {
		return fmt.Errorf("failed to create marker: %w", err)
	}

	return nil
}

func (s *MarkerService) GetAllMarkersWithUsers(ctx context.Context) ([]*models.Marker, error) {
	collection := s.GetCollection("markers")
	var markers []*models.Marker

	// Агрегация для получения маркеров с пользователями
	pipeline := []bson.M{
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "assignedUserIds",
				"foreignField": "_id",
				"as":           "users",
			},
		},
	}

	err := query.Aggregate(ctx, collection, pipeline, &markers)
	if err != nil {
		return nil, fmt.Errorf("failed to get markers with users: %w", err)
	}

	return markers, nil
}

func (s *MarkerService) AssignUserToMarker(ctx context.Context, userID, markerID primitive.ObjectID) error {
	// Обновляем маркер - добавляем пользователя
	markerCollection := s.GetCollection("markers")
	_, err := markerCollection.UpdateOne(
		ctx,
		bson.M{"_id": markerID},
		bson.M{"$addToSet": bson.M{"assignedUserIds": userID}},
	)
	if err != nil {
		return fmt.Errorf("failed to assign user to marker: %w", err)
	}

	// Обновляем пользователя - добавляем маркер
	userCollection := s.GetCollection("users")
	_, err = userCollection.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$addToSet": bson.M{"assignedMarkers": markerID}},
	)
	if err != nil {
		return fmt.Errorf("failed to assign marker to user: %w", err)
	}

	return nil
}

func (s *MarkerService) RemoveUserFromMarker(ctx context.Context, userID, markerID primitive.ObjectID) error {
	// Обновляем маркер - удаляем пользователя
	markerCollection := s.GetCollection("markers")
	_, err := markerCollection.UpdateOne(
		ctx,
		bson.M{"_id": markerID},
		bson.M{"$pull": bson.M{"assignedUserIds": userID}},
	)
	if err != nil {
		return fmt.Errorf("failed to remove user from marker: %w", err)
	}

	// Обновляем пользователя - удаляем маркер
	userCollection := s.GetCollection("users")
	_, err = userCollection.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$pull": bson.M{"assignedMarkers": markerID}},
	)
	if err != nil {
		return fmt.Errorf("failed to remove marker from user: %w", err)
	}

	return nil
}

func (s *MarkerService) ClearAllUsersFromMarker(ctx context.Context, markerID primitive.ObjectID) error {
	// Получаем маркер для получения всех назначенных пользователей
	marker, err := s.GetMarkerByID(ctx, markerID)
	if err != nil {
		return err
	}

	// Удаляем этот маркер из всех пользователей
	if len(marker.Users) > 0 {
		userCollection := s.GetCollection("users")
		_, err = userCollection.UpdateMany(
			ctx,
			bson.M{"_id": bson.M{"$in": marker.Users}},
			bson.M{"$pull": bson.M{"assignedMarkers": markerID}},
		)
		if err != nil {
			return fmt.Errorf("failed to remove marker from users: %w", err)
		}
	}

	// Очищаем список пользователей в маркере
	markerCollection := s.GetCollection("markers")
	_, err = markerCollection.UpdateOne(
		ctx,
		bson.M{"_id": markerID},
		bson.M{"$set": bson.M{"assignedUserIds": []primitive.ObjectID{}}},
	)
	if err != nil {
		return fmt.Errorf("failed to clear marker users: %w", err)
	}

	return nil
}