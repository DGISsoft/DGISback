package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationType string

const (
	NotificationTypeGeneral   NotificationType = "GENERAL"
	NotificationTypePersonal  NotificationType = "PERSONAL"
	NotificationTypeSystem    NotificationType = "SYSTEM"
	NotificationTypeAssignment NotificationType = "ASSIGNMENT"
)

type NotificationStatus string

const (
	NotificationStatusUnread   NotificationStatus = "UNREAD"
	NotificationStatusRead     NotificationStatus = "READ"
	NotificationStatusArchived NotificationStatus = "ARCHIVED"
)

type Notification struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Type         NotificationType     `bson:"type" json:"type"`
	Title        string               `bson:"title" json:"title"`
	Message      string               `bson:"message" json:"message"`
	SenderID     primitive.ObjectID   `bson:"senderId" json:"senderId"`
	RecipientIDs []primitive.ObjectID `bson:"recipientIds,omitempty" json:"recipientIds,omitempty"`
	CreatedAt    time.Time            `bson:"createdAt" json:"createdAt"`
}

type UserNotification struct {
	ID             primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	UserID         primitive.ObjectID    `bson:"userId" json:"userId"`
	NotificationID primitive.ObjectID    `bson:"notificationId" json:"notificationId"`
	Status         NotificationStatus    `bson:"status" json:"status"`
	ReadAt         *time.Time            `bson:"readAt,omitempty" json:"readAt,omitempty"`
	CreatedAt      time.Time             `bson:"createdAt" json:"createdAt"`
}