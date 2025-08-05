package graph

import "github.com/DGISsoft/DGISback/services/mongo"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
type Resolver struct{
	UserService *mongo.MongoService
}
