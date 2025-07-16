package command

import (
	"go.mongodb.org/mongo-driver/bson"
)

type UpdateBuilder struct {
	update bson.M
}

func NewUpdateBuilder() *UpdateBuilder {
	return &UpdateBuilder{update: bson.M{}}
}

func (u *UpdateBuilder) Set(key string, value interface{}) *UpdateBuilder {
	if u.update["$set"] == nil {
		u.update["$set"] = bson.M{}
	}
	u.update["$set"].(bson.M)[key] = value
	return u
}

func (u *UpdateBuilder) Inc(key string, value interface{}) *UpdateBuilder {
	if u.update["$inc"] == nil {
		u.update["$inc"] = bson.M{}
	}
	u.update["$inc"].(bson.M)[key] = value
	return u
}

func (u *UpdateBuilder) Push(key string, value interface{}) *UpdateBuilder {
	if u.update["$push"] == nil {
		u.update["$push"] = bson.M{}
	}
	u.update["$push"].(bson.M)[key] = value
	return u
}

func (u *UpdateBuilder) Pull(key string, value interface{}) *UpdateBuilder {
	if u.update["$pull"] == nil {
		u.update["$pull"] = bson.M{}
	}
	u.update["$pull"].(bson.M)[key] = value
	return u
}

func (u *UpdateBuilder) Unset(key string) *UpdateBuilder {
	if u.update["$unset"] == nil {
		u.update["$unset"] = bson.M{}
	}
	u.update["$unset"].(bson.M)[key] = ""
	return u
}

func (u *UpdateBuilder) Build() bson.M {
	return u.update
}