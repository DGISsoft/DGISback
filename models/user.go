package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserRole string

const (
    RoleChairman   UserRole = "Председатель"
    RoleDGIS       UserRole = "ДГИС"
    RoleStarosta   UserRole = "Староста"
    RoleSupervisor UserRole = "Супервайзер"
)


var RoleHierarchy = map[UserRole]int{
    RoleSupervisor: 1, // наименьшие права
    RoleStarosta:   2,
    RoleDGIS:       3,
    RoleChairman:   4, // наибольшие права
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
    case RoleChairman, RoleDGIS, RoleStarosta, RoleSupervisor:
        return true
    default:
        return false
    }
}

