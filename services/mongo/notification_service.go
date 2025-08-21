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

// CreateUserNotifications создает пользовательские уведомления и оповещает получателей.
func (s *NotificationService) CreateUserNotifications(ctx context.Context, notificationID primitive.ObjectID, recipientIDs []primitive.ObjectID, senderID primitive.ObjectID) error {
	if len(recipientIDs) == 0 {
		log.Printf("NotificationService: No recipients for notification %s, skipping user notification creation", notificationID.Hex())
		return nil
	}

	// Фильтруем получателей, исключая отправителя
	filteredRecipients := make([]primitive.ObjectID, 0, len(recipientIDs))
	for _, id := range recipientIDs {
		// Убедитесь, что сравнение ObjectID корректное.
		// Возможно, нужно использовать id.Equal(senderID) если это возможно,
		// или преобразовать в Hex и сравнивать строки, как ниже.
		// Прямое сравнение == для primitive.ObjectID также должно работать.
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
			Status:         models.NotificationStatusUnread, // Новые уведомления по умолчанию непрочитаны
			CreatedAt:      createdAt,
		})
	}

	_, err := collection.InsertMany(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to create user notifications: %w", err)
	}
	
	// --- КЛЮЧЕВОЕ ИЗМЕНЕНИЕ ---
	// Оповещаем ВСЕХ отфильтрованных получателей о новых уведомлениях.
	// Это обеспечит обновление счетчиков непрочитанных уведомлений
	// в реальном времени у получателей (например, в Navbar).
	for _, userID := range filteredRecipients {
		// Передаем контекст ctx в NotifyUserNotificationChanged
		s.NotifyUserNotificationChanged(ctx, userID)
	}
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---
	
	log.Printf("NotificationService: Created %d user notifications for notification %s and notified recipients", len(docs), notificationID.Hex())
	return nil
}

func (s *NotificationService) GetUserNotifications(ctx context.Context, userID primitive.ObjectID, statuses []models.NotificationStatus, limit, offset int) ([]*models.UserNotification, error) {
	collection := s.GetCollection("user_notifications")

	filter := bson.M{"userId": userID} // Убедитесь, что имя поля в БД "userId"

	if len(statuses) > 0 {
		filter["status"] = bson.M{"$in": statuses}
	}

	opts := options.Find()
	opts.SetSort(bson.D{{Key: "createdAt", Value: -1}}) // Сортировка по убыванию даты создания
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

// GetUserNotificationByID получает пользовательское уведомление по его ID.
func (s *NotificationService) GetUserNotificationByID(ctx context.Context, id primitive.ObjectID) (*models.UserNotification, error) {
	collection := s.GetCollection("user_notifications")
	var userNotif models.UserNotification

	err := query.FindByID(ctx, collection, id, &userNotif)
	if err != nil {
		return nil, fmt.Errorf("failed to get user notification: %w", err)
	}

	return &userNotif, nil
}

// MarkAsRead отмечает пользовательское уведомление как прочитанное и оповещает подписчиков.
func (s *NotificationService) MarkAsRead(ctx context.Context, userNotifID primitive.ObjectID) error {
	// Получаем текущее состояние уведомления перед изменением
	oldNotif, err := s.GetUserNotificationByID(ctx, userNotifID)
	if err != nil {
		return fmt.Errorf("failed to get user notification before marking as read: %w", err)
	}
	
	// Проверка: если уведомление уже прочитано, ничего не делаем
	if oldNotif.Status == models.NotificationStatusRead {
		 log.Printf("NotificationService: User notification %s is already marked as read", userNotifID.Hex())
		 return nil // Или вернуть ошибку, если это неожиданное состояние
	}

	collection := s.GetCollection("user_notifications")
	now := time.Now()

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": userNotifID}, // Фильтр по ID уведомления
		bson.M{ // Обновление статуса и времени прочтения
			"$set": bson.M{
				"status": models.NotificationStatusRead,
				"readAt": now,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}
	
	// Проверка, было ли найдено и обновлено уведомление
	if result.MatchedCount == 0 {
		return fmt.Errorf("user notification %s not found", userNotifID.Hex())
	}
	// Ожидается, что ModifiedCount > 0, если статус изменился, но проверка MatchedCount достаточна.
	
	// --- КЛЮЧЕВОЕ ИЗМЕНЕНИЕ ---
	// Оповещаем подписчиков (в данном случае, самого пользователя)
	// только если статус действительно изменился с Unread на Read.
	// oldNotif.Status проверяется выше.
	// Передаем контекст ctx в NotifyUserNotificationChanged
	s.NotifyUserNotificationChanged(ctx, oldNotif.UserID)
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---
	
	log.Printf("NotificationService: Marked user notification %s as read for user %s", userNotifID.Hex(), oldNotif.UserID.Hex())
	return nil
}

// DeleteUserNotification удаляет пользовательское уведомление и оповещает подписчиков.
func (s *NotificationService) DeleteUserNotification(ctx context.Context, userNotifID primitive.ObjectID) error {
	// Получаем информацию об уведомлении перед удалением для логгирования и оповещения
	userNotif, err := s.GetUserNotificationByID(ctx, userNotifID)
	if err != nil {
		return fmt.Errorf("failed to get user notification before deletion: %w", err)
	}
	
	collection := s.GetCollection("user_notifications")

	result, err := collection.DeleteOne(ctx, bson.M{"_id": userNotifID}) // Фильтр по ID
	if err != nil {
		return fmt.Errorf("failed to delete user notification: %w", err)
	}
	
	// Проверка, было ли удалено уведомление
	if result.DeletedCount == 0 {
		return fmt.Errorf("user notification %s not found", userNotifID.Hex())
	}
	
	// --- КЛЮЧЕВОЕ ИЗМЕНЕНИЕ ---
	// Оповещаем подписчиков (пользователя) об удалении уведомления.
	// Передаем контекст ctx в NotifyUserNotificationChanged
	s.NotifyUserNotificationChanged(ctx, userNotif.UserID)
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---
	
	log.Printf("NotificationService: Deleted user notification %s for user %s", userNotifID.Hex(), userNotif.UserID.Hex())
	return nil
}

// GetNotificationByID получает общее уведомление по его ID.
func (s *NotificationService) GetNotificationByID(ctx context.Context, id primitive.ObjectID) (*models.Notification, error) {
	collection := s.GetCollection("notifications")
	var notif models.Notification

	err := query.FindByID(ctx, collection, id, &notif)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notif, nil
}

// GetUserNotificationWithDetails получает пользовательское уведомление с деталями общего уведомления.
func (s *NotificationService) GetUserNotificationWithDetails(ctx context.Context, userNotifID primitive.ObjectID) (*models.UserNotification, *models.Notification, error) {
	userNotifColl := s.GetCollection("user_notifications")
	var userNotif models.UserNotification
	
	// Получаем пользовательское уведомление
	err := query.FindByID(ctx, userNotifColl, userNotifID, &userNotif)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user notification: %w", err)
	}

	// Получаем связанное общее уведомление
	notif, err := s.GetNotificationByID(ctx, userNotif.NotificationID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get notification details: %w", err)
	}

	return &userNotif, notif, nil
}

// GetUnreadCount возвращает количество непрочитанных уведомлений для пользователя.
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID primitive.ObjectID) (int, error) {
	collection := s.GetCollection("user_notifications")
	
	// Подсчитываем документы, соответствующие фильтру
	count, err := collection.CountDocuments(ctx, bson.M{
		"userId": userID,                    // Фильтр по ID пользователя
		"status": models.NotificationStatusUnread, // Фильтр по статусу "непрочитано"
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}

	return int(count), nil
}

// NotifyUserNotificationChanged оповещает подписчиков об изменении уведомлений пользователя.
// Это делается путем публикации сообщения в Redis.
// Теперь принимает context.Context.
func (s *NotificationService) NotifyUserNotificationChanged(ctx context.Context, userID primitive.ObjectID) {
	// Формируем имя канала Redis, уникальное для пользователя
	// ВАЖНО: Имя канала ДОЛЖНО точно совпадать с именем, на которое подписывается клиент.
	channel := fmt.Sprintf("unread_notifications_changed:%s", userID.Hex())
	
	// Отправляем сообщение в канал Redis.
	// Передаем контекст ctx в RedisService.Publish.
	// Сообщение может быть любым, например, "updated" или "changed".
	if err := s.RedisService.Publish(ctx, channel, "updated"); err != nil {
		// Логируем ошибку публикации, но не возвращаем её,
		// чтобы не прерывать основную бизнес-логику (создание/чтение/удаление уведомления).
		log.Printf("NotificationService: Failed to publish notification change to Redis for user %s on channel %s: %v", userID.Hex(), channel, err)
		// return // Не возвращаем ошибку здесь
	} else {
		 log.Printf("NotificationService: Successfully published notification change to Redis for user %s on channel %s", userID.Hex(), channel)
	}
}