package mongo

import (
	"context"
	"fmt"
	"log"

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

type rawMarkerWithUsers struct {
	ID              primitive.ObjectID   `bson:"_id,omitempty"`
	MarkerID        string               `bson:"markerId"`
	Position        []float64            `bson:"position"`
	Label           string               `bson:"label"`
	AssignedUserIds []primitive.ObjectID `bson:"assignedUserIds"`        // Оригинальное поле
	UsersRaw        []bson.Raw           `bson:"users"`                  // Результат $lookup как сырые BSON документы
	// Или, если вы уверены, что models.User совместим, можно использовать
	// UsersRaw        []models.User       `bson:"users"` 
	// Но bson.Raw более гибкий и безопасный для промежуточного шага.
}

func (s *MarkerService) GetAllMarkersWithUsers(ctx context.Context) ([]*models.Marker, error) {
	collection := s.GetCollection("markers")
	// Используем промежуточный тип для декодирования
	var rawMarkers []*rawMarkerWithUsers

	pipeline := []bson.M{
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "assignedUserIds", // Источник связей
				"foreignField": "_id",             // Поле в коллекции users
				"as":           "users",           // Имя поля для результата
			},
		},
	}

	log.Println("GetAllMarkersWithUsers: Executing aggregation pipeline...")
	err := query.Aggregate(ctx, collection, pipeline, &rawMarkers)
	if err != nil {
		log.Printf("GetAllMarkersWithUsers: Aggregation failed: %v", err)
		return nil, fmt.Errorf("failed to get markers with users: %w", err)
	}

	log.Printf("GetAllMarkersWithUsers: Successfully retrieved %d raw markers", len(rawMarkers))

	// Преобразуем сырые данные в модели
	resultMarkers := make([]*models.Marker, len(rawMarkers))
	for i, rawMarker := range rawMarkers {
		// 1. Создаем экземпляр models.Marker
		marker := &models.Marker{
			ID:       rawMarker.ID,
			MarkerID: rawMarker.MarkerID,
			Position: rawMarker.Position,
			Label:    rawMarker.Label,
			// AssignedUserIds: rawMarker.AssignedUserIds, // Если нужно
		}

		// 2. Преобразуем []bson.Raw в []*models.User
		users := make([]*models.User, len(rawMarker.UsersRaw))
		for j, userRaw := range rawMarker.UsersRaw {
			var user models.User
			// Декодируем bson.Raw в models.User
			err := bson.Unmarshal(userRaw, &user)
			if err != nil {
				log.Printf("GetAllMarkersWithUsers: Failed to unmarshal user [%d] for marker [%d] (%s): %v", j, i, rawMarker.ID.Hex(), err)
				// Можно либо пропустить пользователя, либо вернуть ошибку
				// Пока пропустим
				continue
			}
			users[j] = &user
		}
		// Убираем nil значения, если были ошибки декодирования
		// (Это не обязательно, если мы уверены в данных)
		// filteredUsers := []*models.User{}
		// for _, u := range users { if u != nil { filteredUsers = append(filteredUsers, u) } }
		// marker.Users = filteredUsers
		
		marker.Users = users

		resultMarkers[i] = marker

		// --- Логирование для проверки ---
		// if i < 3 {
		// 	log.Printf("Marker [%d] Converted - ID: %s, Label: %s, Users (len): %d", 
		// 		i, marker.ID.Hex(), marker.Label, len(marker.Users))
		// 	if len(marker.Users) > 0 && marker.Users[0] != nil {
		// 		log.Printf("Marker [%d] First User - ID: %s, Name: %s", 
		// 			i, marker.Users[0].ID.Hex(), marker.Users[0].FullName)
		// 	}
		// }
		// --- Конец логирования ---
	}

	log.Printf("GetAllMarkersWithUsers: Successfully converted to %d final markers", len(resultMarkers))
	return resultMarkers, nil
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