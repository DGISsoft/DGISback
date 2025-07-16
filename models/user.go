package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserRole string

const (
	RoleDGIS       UserRole = "ДГИС"
	RoleChairman   UserRole = "Председатель"
	RoleSupervisor UserRole = "Супервайзер"
	RoleStarosta   UserRole = "Староста"
)

type Campus string

const (
	CampusNorth Campus = "Северные корпуса"
	CampusSouth Campus = "Южные корпуса"
	CampusSmall Campus = "Малый Аякс"
)

type ControlQA struct {
	Question string `json:"question" bson:"question"`
	Answer   string `json:"answer" bson:"answer"`
}

type User struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Login        string             `json:"login" bson:"login"`
	Password     string             `json:"-" bson:"password"`
	Role         UserRole           `json:"role" bson:"role"`
	FullName     string             `json:"full_name" bson:"full_name"`
	Campus       Campus             `json:"campus" bson:"campus"`
	Building     string           `json:"building" bson:"building"`
	ProfileImage string             `json:"profile_image" bson:"profile_image"`
	ControlQA    ControlQA          `json:"control_qa" bson:"control_qa"`
	IsActive     bool               `json:"is_active" bson:"is_active"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}