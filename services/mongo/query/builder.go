package query

import (
	"go.mongodb.org/mongo-driver/bson"
)

type Builder struct {
	filter bson.M
}

func NewBuilder() *Builder {
	return &Builder{filter: bson.M{}}
}

func (b *Builder) Where(key string, value interface{}) *Builder {
	b.filter[key] = value
	return b
}

func (b *Builder) WhereIn(key string, values []interface{}) *Builder {
	b.filter[key] = bson.M{"$in": values}
	return b
}

func (b *Builder) WhereRegex(key, pattern string) *Builder {
	b.filter[key] = bson.M{"$regex": pattern, "$options": "i"}
	return b
}

func (b *Builder) WhereGT(key string, value interface{}) *Builder {
	b.filter[key] = bson.M{"$gt": value}
	return b
}

func (b *Builder) WhereLT(key string, value interface{}) *Builder {
	b.filter[key] = bson.M{"$lt": value}
	return b
}

func (b *Builder) WhereExists(key string) *Builder {
	b.filter[key] = bson.M{"$exists": true}
	return b
}

func (b *Builder) Build() bson.M {
	return b.filter
}