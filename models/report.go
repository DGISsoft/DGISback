package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReportType string

const (
	ReportTypeDaily   ReportType = "daily"
	ReportTypeWeekly  ReportType = "weekly"
	ReportTypeMonthly ReportType = "monthly"
)

type ReportStatus string

const (
	StatusDraft     ReportStatus = "draft"
	StatusSubmitted ReportStatus = "submitted"
	StatusMerged  ReportStatus = "merged"
)

type MediaFile struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Filename string             `json:"filename" bson:"filename"`
	FilePath string             `json:"file_path" bson:"file_path"`
	FileType string             `json:"file_type" bson:"file_type"`
	FileSize int64              `json:"file_size" bson:"file_size"`
}

type ReportComment struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Content   string             `json:"content" bson:"content"`
	AuthorID  primitive.ObjectID `json:"author_id" bson:"author_id"`
	Author    *User              `json:"author,omitempty" bson:"author,omitempty"`
	IsLike    bool               `json:"is_like" bson:"is_like"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

type Report struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Title       string             `json:"title" bson:"title"`
	Content     string             `json:"content" bson:"content"`
	Type        ReportType         `json:"type" bson:"type"`
	Status      ReportStatus       `json:"status" bson:"status"`
	AuthorID    primitive.ObjectID `json:"author_id" bson:"author_id"`
	Author      *User              `json:"author,omitempty" bson:"author,omitempty"`
	Campus      Campus             `json:"campus" bson:"campus"`
	Building    string           `json:"building" bson:"building"`
	MediaFiles  []MediaFile        `json:"media_files" bson:"media_files"`
	Comments    []ReportComment    `json:"comments" bson:"comments"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
	SubmittedAt *time.Time         `json:"submitted_at,omitempty" bson:"submitted_at,omitempty"`
}