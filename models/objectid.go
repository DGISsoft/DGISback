package models

import (
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ObjectID = primitive.ObjectID

func MarshalObjectID(id ObjectID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		fmt.Fprintf(w, "\"%s\"", id.Hex())
	})
}

func UnmarshalObjectID(v interface{}) (ObjectID, error) {
	str, ok := v.(string)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("ObjectID must be a string")
	}
	return primitive.ObjectIDFromHex(str)
}
func MarshalObjectIDScalar(id primitive.ObjectID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		fmt.Fprintf(w, "\"%s\"", id.Hex())
	})
}