package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Permission struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Role      UserRole           `json:"role" bson:"role"`
	Resource  string             `json:"resource" bson:"resource"`
	Action    string             `json:"action" bson:"action"`
	CanAccess bool               `json:"can_access" bson:"can_access"`
	Scope     string             `json:"scope" bson:"scope"` // "own", "campus", "building", "all"
}
