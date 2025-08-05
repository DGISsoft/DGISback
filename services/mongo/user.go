package mongo

import (
	"context"
	"fmt"

	"github.com/DGISsoft/DGISback/models"
	"github.com/DGISsoft/DGISback/services/mongo/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// UserService сервис для работы с пользователями
type UserService struct {
	*MongoService
}

// NewUserService создает новый сервис для работы с пользователями
func NewUserService(mongoService *MongoService) *UserService {
	return &UserService{MongoService: mongoService}
}

// GetUserByLogin получает пользователя по логину
func (s *UserService) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	collection := s.GetCollection("users")
	
	var user models.User
	filter := bson.M{"login": login}
	
	err := query.FindOne(ctx, collection, filter, &user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return &user, nil
}

// GetUserByID получает пользователя по ID
func (s *UserService) GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	collection := s.GetCollection("users")
	
	var user models.User
	
	err := query.FindByID(ctx, collection, id, &user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return &user, nil
}

// GetUsers получает список всех пользователей
func (s *UserService) GetUsers(ctx context.Context) ([]*models.User, error) {
	collection := s.GetCollection("users")
	
	var users []*models.User
	
	err := query.FindMany(ctx, collection, bson.M{}, &users)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	
	return users, nil
}

// CreateUser создает нового пользователя
func (s *UserService) CreateUser(ctx context.Context, user *models.User) error {
	collection := s.GetCollection("users")
	
	// Хешируем пароль перед сохранением
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)
	
	// Устанавливаем временные метки
	user.CreatedAt = user.UpdatedAt
	
	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	
	return nil
}

func (s *UserService) UpdateUser(ctx context.Context, id primitive.ObjectID, updateData bson.M) error {
    collection := s.GetCollection("users")

    updateQuery := bson.M{
        "$set":         updateData,
        "$currentDate": bson.M{"updated_at": true}, // Устанавливаем updated_at в текущую дату/время
    }


    _, err := collection.UpdateOne(ctx, bson.M{"_id": id}, updateQuery) // Используем updateQuery
    if err != nil {
        return fmt.Errorf("failed to update user: %w", err)
    }

    return nil
}

// DeleteUser удаляет пользователя
func (s *UserService) DeleteUser(ctx context.Context, id primitive.ObjectID) error {
	collection := s.GetCollection("users")
	
	_, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	
	return nil
}

// CheckPassword проверяет пароль пользователя
func (s *UserService) CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// ChangePassword изменяет пароль пользователя
func (s *UserService) ChangePassword(ctx context.Context, id primitive.ObjectID, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	
	return s.UpdateUser(ctx, id, bson.M{"password": string(hashedPassword)})
}