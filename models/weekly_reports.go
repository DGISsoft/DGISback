package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReportStatus string

const (
	StatusReviewed   ReportStatus = "reviewed"
	StatusNotReviewed ReportStatus = "not_reviewed"
)

type Rating string

const (
	RatingGood Rating = "good"
	RatingBad  Rating = "bad"
)

type WeeklyReport struct {
	ID                    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CreatedAt             time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt             time.Time          `bson:"updated_at" json:"updated_at"`
	Applications          string             `bson:"applications" json:"applications"`
	Inspection            string             `bson:"inspection" json:"inspection"`
	Additional            string             `bson:"additional" json:"additional"`
	Status                ReportStatus       `bson:"status" json:"status"`
	SupervisorRate        *Rating            `bson:"supervisor_rate,omitempty" json:"supervisor_rate,omitempty"`
	PredsedatelRate       *Rating            `bson:"predsedatel_rate,omitempty" json:"predsedatel_rate,omitempty"`
	
	ApplicationsImageKeys []string           `bson:"applications_image_keys" json:"applications_image_keys"`
	InspectionImageKeys   []string           `bson:"inspection_image_keys" json:"inspection_image_keys"`
	AdditionalImageKeys   []string           `bson:"additional_image_keys" json:"additional_image_keys"`
	
	UserID                primitive.ObjectID `bson:"user_id" json:"user_id"`
}