// services/notification_service.go
package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DGISsoft/DGISback/models"
	"github.com/DGISsoft/DGISback/services/mongo/query"
	"github.com/DGISsoft/DGISback/services/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NotificationService struct {
	*MongoService
	RedisService *redis.RedisService
}

func NewNotificationService(mongoService *MongoService, redisService *redis.RedisService) *NotificationService {
	return &NotificationService{
		MongoService: mongoService,
		RedisService: redisService,
	}
}

func (s *NotificationService) CreateNotification(ctx context.Context, notif *models.Notification) error {
	collection := s.GetCollection("notifications")
	notif.CreatedAt = time.Now()

	res, err := collection.InsertOne(ctx, notif)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		notif.ID = oid
		log.Printf("NotificationService: Created notification with ID %s", notif.ID.Hex())
	} else {
		return fmt.Errorf("failed to get inserted notification ID, expected ObjectID, got %T", res.InsertedID)
	}
	
	log.Printf("NotificationService: Created notification %s (Title: '%s')", notif.ID.Hex(), notif.Title)
	return nil
}

func (s *NotificationService) CreateUserNotifications(ctx context.Context, notificationID primitive.ObjectID, recipientIDs []primitive.ObjectID, senderID primitive.ObjectID) error {
	if len(recipientIDs) == 0 {
		log.Printf("NotificationService: No recipients for notification %s, skipping user notification creation", notificationID.Hex())
		return nil
	}

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
	
	// Оповещаем всех получателей о новых уведомлениях
	for _, userID := range filteredRecipients {
		// Передаем ctx в NotifyUserNotificationChanged
		s.NotifyUserNotificationChanged(ctx, userID)
	}
	
	log.Printf("NotificationService: Created %d user notifications for notification %s", len(docs), notificationID.Hex())
	return nil
}

func (s *NotificationService) GetUserNotifications(ctx context.Context, userID primitive.ObjectID, statuses []models.NotificationStatus, limit, offset int) ([]*models.UserNotification, error) {
	collection := s.GetCollection("user_notifications")

	filter := bson.M{"userId": userID}

	if len(statuses) > 0 {
		filter["status"] = bson.M{"$in": statuses}
	}

	opts := options.Find()
	opts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}

	var userNotifs []*models.UserNotification

	err := query.FindMany(ctx, collection, filter, &userNotifs, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get user notifications: %w", err)
	}

	return userNotifs, nil
}

// Добавляем вспомогательный метод для получения уведомления по ID
func (s *NotificationService) GetUserNotificationByID(ctx context.Context, id primitive.ObjectID) (*models.UserNotification, error) {
	collection := s.GetCollection("user_notifications")
	var userNotif models.UserNotification

	err := query.FindByID(ctx, collection, id, &userNotif)
	if err != nil {
		return nil, fmt.Errorf("failed to get user notification: %w", err)
	}

	return &userNotif, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, userNotifID primitive.ObjectID) error {
	// Получаем текущее состояние уведомления перед изменением
	oldNotif, err := s.GetUserNotificationByID(ctx, userNotifID)
	if err != nil {
		return fmt.Errorf("failed to get user notification before marking as read: %w", err)
	}
	
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
	
	// Оповещаем подписчиков только если статус действительно изменился
	if oldNotif.Status != models.NotificationStatusRead {
		// Передаем ctx в NotifyUserNotificationChanged
		s.NotifyUserNotificationChanged(ctx, oldNotif.UserID)
	}
	
	log.Printf("NotificationService: Marked user notification %s as read", userNotifID.Hex())
	return nil
}

func (s *NotificationService) DeleteUserNotification(ctx context.Context, userNotifID primitive.ObjectID) error {
	// Получаем информацию об уведомлении перед удалением
	userNotif, err := s.GetUserNotificationByID(ctx, userNotifID)
	if err != nil {
		return fmt.Errorf("failed to get user notification before deletion: %w", err)
	}
	
	collection := s.GetCollection("user_notifications")

	result, err := collection.DeleteOne(ctx, bson.M{"_id": userNotifID})
	if err != nil {
		return fmt.Errorf("failed to delete user notification: %w", err)
	}
	
	if result.DeletedCount == 0 {
		return fmt.Errorf("user notification %s not found", userNotifID.Hex())
	}
	
	// Оповещаем подписчиков об удалении
	// Передаем ctx в NotifyUserNotificationChanged
	s.NotifyUserNotificationChanged(ctx, userNotif.UserID)
	
	log.Printf("NotificationService: Deleted user notification %s", userNotifID.Hex())
	return nil
}

func (s *NotificationService) GetNotificationByID(ctx context.Context, id primitive.ObjectID) (*models.Notification, error) {
	collection := s.GetCollection("notifications")
	var notif models.Notification

	err := query.FindByID(ctx, collection, id, &notif)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notif, nil
}

func (s *NotificationService) GetUserNotificationWithDetails(ctx context.Context, userNotifID primitive.ObjectID) (*models.UserNotification, *models.Notification, error) {
	userNotifColl := s.GetCollection("user_notifications")
	var userNotif models.UserNotification
	
	err := query.FindByID(ctx, userNotifColl, userNotifID, &userNotif)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user notification: %w", err)
	}

	notif, err := s.GetNotificationByID(ctx, userNotif.NotificationID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get notification details: %w", err)
	}

	return &userNotif, notif, nil
}

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

// Метод для оповещения подписчиков об изменении уведомлений
// Теперь принимает context.Context
func (s *NotificationService) NotifyUserNotificationChanged(ctx context.Context, userID primitive.ObjectID) {
	// Публикуем сообщение в Redis канал
	channel := fmt.Sprintf("unread_notifications_changed:%s", userID.Hex())
	
	// Отправляем сообщение через Redis, передавая ctx
	if err := s.RedisService.Publish(ctx, channel, "changed"); err != nil {
		log.Printf("Failed to publish notification change for user %s: %v", userID.Hex(), err)
	}
}