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
	AssignedUserIds []primitive.ObjectID `bson:"assignedUserIds"`
	UsersRaw        []bson.Raw           `bson:"users"` 
}

func (s *MarkerService) GetAllMarkersWithUsers(ctx context.Context) ([]*models.Marker, error) {
	collection := s.GetCollection("markers")

	var rawMarkers []*rawMarkerWithUsers

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

	log.Println("GetAllMarkersWithUsers: Executing aggregation pipeline...")
	err := query.Aggregate(ctx, collection, pipeline, &rawMarkers)
	if err != nil {
		log.Printf("GetAllMarkersWithUsers: Aggregation failed: %v", err)
		return nil, fmt.Errorf("failed to get markers with users: %w", err)
	}

	log.Printf("GetAllMarkersWithUsers: Successfully retrieved %d raw markers", len(rawMarkers))


	resultMarkers := make([]*models.Marker, len(rawMarkers))
	for i, rawMarker := range rawMarkers {

		marker := &models.Marker{
			ID:       rawMarker.ID,
			MarkerID: rawMarker.MarkerID,
			Position: rawMarker.Position,
			Label:    rawMarker.Label,
		}


		users := make([]*models.User, len(rawMarker.UsersRaw))
		for j, userRaw := range rawMarker.UsersRaw {
			var user models.User

			err := bson.Unmarshal(userRaw, &user)
			if err != nil {
				log.Printf("GetAllMarkersWithUsers: Failed to unmarshal user [%d] for marker [%d] (%s): %v", j, i, rawMarker.ID.Hex(), err)
				continue
			}
			users[j] = &user
		}

		marker.Users = users
		resultMarkers[i] = marker
	}

	log.Printf("GetAllMarkersWithUsers: Successfully converted to %d final markers", len(resultMarkers))
	return resultMarkers, nil
}

func (s *MarkerService) AssignUserToMarker(ctx context.Context, userID, markerID primitive.ObjectID) error {

	markerCollection := s.GetCollection("markers")
	var marker struct {
		ID       primitive.ObjectID `bson:"_id"`
		Label    string             `bson:"label"`
		MarkerID string             `bson:"markerId"`
	}
	
	err := query.FindByID(ctx, markerCollection, markerID, &marker)
	if err != nil {
		return fmt.Errorf("failed to get marker: %w", err)
	}


	_, err = markerCollection.UpdateOne(
		ctx,
		bson.M{"_id": markerID},
		bson.M{"$addToSet": bson.M{"assignedUserIds": userID}},
	)
	if err != nil {
		return fmt.Errorf("failed to assign user to marker: %w", err)
	}


	userCollection := s.GetCollection("users")
	_, err = userCollection.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{
			"$addToSet": bson.M{"assignedMarkers": markerID},
			"$set": bson.M{"building": marker.Label},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to assign marker to user: %w", err)
	}

	return nil
}


func (s *MarkerService) RemoveUserFromMarker(ctx context.Context, userID, markerID primitive.ObjectID) error {

	markerCollection := s.GetCollection("markers")
	_, err := markerCollection.UpdateOne(
		ctx,
		bson.M{"_id": markerID},
		bson.M{"$pull": bson.M{"assignedUserIds": userID}},
	)
	if err != nil {
		return fmt.Errorf("failed to remove user from marker: %w", err)
	}


	var user struct {
		ID      primitive.ObjectID   `bson:"_id"`
		Markers []primitive.ObjectID `bson:"assignedMarkers"`
	}
	
	userCollection := s.GetCollection("users")
	err = query.FindByID(ctx, userCollection, userID, &user)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	_, err = userCollection.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$pull": bson.M{"assignedMarkers": markerID}},
	)
	if err != nil {
		return fmt.Errorf("failed to remove marker from user: %w", err)
	}

	var newBuilding *string = nil
	
	remainingMarkers := make([]primitive.ObjectID, 0)
	for _, mID := range user.Markers {
		if mID != markerID {
			remainingMarkers = append(remainingMarkers, mID)
		}
	}

	if len(remainingMarkers) > 0 {
		var remainingMarker struct {
			Label string `bson:"label"`
		}
		err = query.FindOne(ctx, markerCollection, bson.M{"_id": remainingMarkers[0]}, &remainingMarker)
		if err == nil {
			newBuilding = &remainingMarker.Label
		}
	}

	updateFields := bson.M{"building": newBuilding}
	_, err = userCollection.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$set": updateFields},
	)
	if err != nil {
		return fmt.Errorf("failed to update user building: %w", err)
	}

	return nil
}

func (s *MarkerService) ClearAllUsersFromMarker(ctx context.Context, markerID primitive.ObjectID) error {
	marker, err := s.GetMarkerByID(ctx, markerID)
	if err != nil {
		return err
	}


	if len(marker.Users) > 0 {
		userCollection := s.GetCollection("users")
		

		_, err = userCollection.UpdateMany(
			ctx,
			bson.M{"_id": bson.M{"$in": marker.Users}},
			bson.M{
				"$pull": bson.M{"assignedMarkers": markerID},
				"$set":  bson.M{"building": nil},
			},
		)
		if err != nil {
			return fmt.Errorf("failed to remove marker from users: %w", err)
		}
	}

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


func (s *MarkerService) GetMarkerByLabel(ctx context.Context, label string) (*models.Marker, error) {
	collection := s.GetCollection("markers")
	var marker models.Marker
	filter := bson.M{"label": label}

	err := query.FindOne(ctx, collection, filter, &marker)
	if err != nil {
		return nil, fmt.Errorf("failed to get marker by label '%s': %w", label, err)
	}

	return &marker, nil
}