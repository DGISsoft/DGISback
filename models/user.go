package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserRole string

const (
    UserRolePredsedatel UserRole = "PREDSEDATEL"
    UserRoleDgis        UserRole = "DGIS"
    UserRoleStarosta    UserRole = "STAROSTA"
    UserRoleSupervisor  UserRole = "SUPERVISOR"
)

var RoleHierarchy = map[UserRole]int{
    UserRoleSupervisor: 1,
    UserRoleStarosta:   2,
    UserRoleDgis:       3,
    UserRolePredsedatel:   4,
}

type User struct {
    ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"` // Исправлено: bson:"_id,omitempty"
    Login        string             `json:"login" bson:"login"`
    Password     string             `json:"-" bson:"password"`
    Role         UserRole           `json:"role" bson:"role"`
    FullName     string             `json:"full_name" bson:"full_name"`
    Building     *string            `json:"building,omitempty" bson:"building,omitempty"`
    PhoneNumber  string             `json:"phone_number" bson:"phone_number"`
    TelegramTag  string             `json:"telegram_tag" bson:"telegram_tag"`
    CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
    UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}


func (u *User) HasHigherRole(role UserRole) bool {
    userRoleLevel := RoleHierarchy[u.Role]
    targetRoleLevel := RoleHierarchy[role]
    return userRoleLevel > targetRoleLevel
}


func (u *User) HasEqualOrHigherRole(role UserRole) bool {
    userRoleLevel := RoleHierarchy[u.Role]
    targetRoleLevel := RoleHierarchy[role]
    return userRoleLevel >= targetRoleLevel
}

func (r UserRole) IsValid() bool {
    switch r {
    case UserRolePredsedatel, UserRoleDgis, UserRoleStarosta, UserRoleSupervisor:
        return true
    default:
        return false
    }
}

