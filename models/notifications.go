package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotificationType определяет тип уведомления
type NotificationType string

const (
	NotificationTypeGeneral   NotificationType = "GENERAL"
	NotificationTypePersonal  NotificationType = "PERSONAL"
	NotificationTypeSystem    NotificationType = "SYSTEM"
	NotificationTypeAssignment NotificationType = "ASSIGNMENT"
)

// NotificationStatus определяет статус уведомления для пользователя
type NotificationStatus string

const (
	NotificationStatusUnread   NotificationStatus = "UNREAD"
	NotificationStatusRead     NotificationStatus = "READ"
	NotificationStatusArchived NotificationStatus = "ARCHIVED"
)

// Notification представляет собой шаблон уведомления
type Notification struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Type         NotificationType     `bson:"type" json:"type"`
	Title        string               `bson:"title" json:"title"`
	Message      string               `bson:"message" json:"message"`
	SenderID     primitive.ObjectID   `bson:"senderId" json:"senderId"`         // ID отправителя
	RecipientIDs []primitive.ObjectID `bson:"recipientIds,omitempty" json:"recipientIds,omitempty"` // ID получателей (для массовых)
	CreatedAt    time.Time            `bson:"createdAt" json:"createdAt"`
}

// UserNotification связывает уведомление с конкретным пользователем
type UserNotification struct {
	ID             primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	UserID         primitive.ObjectID    `bson:"userId" json:"userId"`             // ID получателя
	NotificationID primitive.ObjectID    `bson:"notificationId" json:"notificationId"` // Ссылка на уведомление
	Status         NotificationStatus    `bson:"status" json:"status"`             // Статус для этого пользователя
	ReadAt         *time.Time            `bson:"readAt,omitempty" json:"readAt,omitempty"` // Когда прочитано
	CreatedAt      time.Time             `bson:"createdAt" json:"createdAt"`
}