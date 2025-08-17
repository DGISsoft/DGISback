package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DGISsoft/DGISback/models"
	"github.com/DGISsoft/DGISback/services/mongo/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NotificationService предоставляет методы для работы с уведомлениями
type NotificationService struct {
	*MongoService
}

// NewNotificationService создает новый сервис для работы с уведомлениями
func NewNotificationService(mongoService *MongoService) *NotificationService {
	return &NotificationService{MongoService: mongoService}
}

// CreateNotification создает новое уведомление (шаблон)
func (s *NotificationService) CreateNotification(ctx context.Context, notif *models.Notification) error {
	collection := s.GetCollection("notifications")
	notif.CreatedAt = time.Now()

	_, err := collection.InsertOne(ctx, notif)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	
	log.Printf("NotificationService: Created notification %s (Title: '%s')", notif.ID.Hex(), notif.Title)
	return nil
}

// CreateUserNotifications создает записи UserNotification для списка пользователей
func (s *NotificationService) CreateUserNotifications(ctx context.Context, notificationID primitive.ObjectID, recipientIDs []primitive.ObjectID, senderID primitive.ObjectID) error {
	if len(recipientIDs) == 0 {
		log.Printf("NotificationService: No recipients for notification %s, skipping user notification creation", notificationID.Hex())
		return nil // Нет получателей
	}

	// Исключаем отправителя из списка получателей, если он там есть
	// (опционально, зависит от бизнес-логики)
	filteredRecipients := make([]primitive.ObjectID, 0, len(recipientIDs))
	for _, id := range recipientIDs {
		if id != senderID {
			filteredRecipients = append(filteredRecipients, id)
		}
	}
	
	if len(filteredRecipients) == 0 {
		log.Printf("NotificationService: No valid recipients for notification %s after filtering sender", notificationID.Hex())
		return nil
	}

	collection := s.GetCollection("user_notifications")
	createdAt := time.Now()

	// Создаем срез документов для вставки
	var docs []interface{}
	for _, userID := range filteredRecipients {
		docs = append(docs, models.UserNotification{
			UserID:         userID,
			NotificationID: notificationID,
			Status:         models.NotificationStatusUnread,
			CreatedAt:      createdAt,
		})
	}

	_, err := collection.InsertMany(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to create user notifications: %w", err)
	}
	
	log.Printf("NotificationService: Created %d user notifications for notification %s", len(docs), notificationID.Hex())
	return nil
}

// GetUserNotifications получает список уведомлений для конкретного пользователя
func (s *NotificationService) GetUserNotifications(ctx context.Context, userID primitive.ObjectID, statuses []models.NotificationStatus, limit, offset int) ([]*models.UserNotification, error) {
	collection := s.GetCollection("user_notifications")
	
	// Фильтр по пользователю
	filter := bson.M{"userId": userID}
	
	// Если переданы статусы, добавляем фильтр по ним
	if len(statuses) > 0 {
		filter["status"] = bson.M{"$in": statuses}
	}

	// Опции сортировки (новые сверху) и пагинации
	opts := options.Find()
	opts.SetSort(bson.D{{"createdAt", -1}}) // Сортировка по убыванию даты создания
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}

	var userNotifs []*models.UserNotification
	err := query.FindManyWithOptions(ctx, collection, filter, &userNotifs, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get user notifications: %w", err)
	}

	return userNotifs, nil
}

// MarkAsRead помечает уведомление как прочитанное
func (s *NotificationService) MarkAsRead(ctx context.Context, userNotifID primitive.ObjectID) error {
	collection := s.GetCollection("user_notifications")
	now := time.Now()

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": userNotifID},
		bson.M{
			"$set": bson.M{
				"status": models.NotificationStatusRead,
				"readAt": now,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("user notification %s not found", userNotifID.Hex())
	}
	
	log.Printf("NotificationService: Marked user notification %s as read", userNotifID.Hex())
	return nil
}

// DeleteUserNotification удаляет запись UserNotification (например, при архивации)
func (s *NotificationService) DeleteUserNotification(ctx context.Context, userNotifID primitive.ObjectID) error {
	collection := s.GetCollection("user_notifications")

	result, err := collection.DeleteOne(ctx, bson.M{"_id": userNotifID})
	if err != nil {
		return fmt.Errorf("failed to delete user notification: %w", err)
	}
	
	if result.DeletedCount == 0 {
		return fmt.Errorf("user notification %s not found", userNotifID.Hex())
	}
	
	log.Printf("NotificationService: Deleted user notification %s", userNotifID.Hex())
	return nil
}

// GetNotificationByID получает шаблон уведомления по ID
func (s *NotificationService) GetNotificationByID(ctx context.Context, id primitive.ObjectID) (*models.Notification, error) {
	collection := s.GetCollection("notifications")
	var notif models.Notification

	err := query.FindByID(ctx, collection, id, &notif)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notif, nil
}

// GetUserNotificationWithDetails получает UserNotification с полной информацией о самом уведомлении
func (s *NotificationService) GetUserNotificationWithDetails(ctx context.Context, userNotifID primitive.ObjectID) (*models.UserNotification, *models.Notification, error) {
	// 1. Получить UserNotification
	userNotifColl := s.GetCollection("user_notifications")
	var userNotif models.UserNotification
	
	err := query.FindByID(ctx, userNotifColl, userNotifID, &userNotif)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user notification: %w", err)
	}

	// 2. Получить связанный Notification
	notif, err := s.GetNotificationByID(ctx, userNotif.NotificationID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get notification details: %w", err)
	}

	return &userNotif, notif, nil
}

// GetUnreadCount получает количество непрочитанных уведомлений для пользователя
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID primitive.ObjectID) (int, error) {
	collection := s.GetCollection("user_notifications")
	
	count, err := collection.CountDocuments(ctx, bson.M{
		"userId": userID,
		"status": models.NotificationStatusUnread,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}

	return int(count), nil
}

// GetUserNotificationByID получает UserNotification по ID
func (s *NotificationService) GetUserNotificationByID(ctx context.Context, id primitive.ObjectID) (*models.UserNotification, error) {
	collection := s.GetCollection("user_notifications")
	var userNotif models.UserNotification

	err := query.FindByID(ctx, collection, id, &userNotif)
	if err != nil {
		return nil, fmt.Errorf("failed to get user notification: %w", err)
	}

	return &userNotif, nil
}