package rbac

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const permissionCollection = "permissions"

type MongoDBPermissionsRepository struct {
	db             *mongo.Database
	collectionName string
}

// Read implements PermissionsRepository.
func (m *MongoDBPermissionsRepository) Read(ctx context.Context, key string) (*Permission, error) {
	key = strings.TrimSpace(key)
	collection := m.db.Collection(m.collectionName)
	filter := bson.M{"key": key}
	var permission Permission
	err := collection.FindOne(ctx, filter).Decode(&permission)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, PermissionNotFoundError{Value: key}
		}
		return nil, err
	}
	return &permission, nil
}

// Create implements PermissionsRepository.
func (m *MongoDBPermissionsRepository) Create(ctx context.Context, permission Permission) error {
	collection := m.db.Collection(m.collectionName)
	_, err := collection.InsertOne(ctx, permission)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return PermissionDuplicateError{
				Value: permission.Key,
			}
		}
		return err
	}
	return nil
}

// Delete implements PermissionsRepository.
func (m *MongoDBPermissionsRepository) Delete(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	collection := m.db.Collection(m.collectionName)
	filter := bson.M{"key": key}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return PermissionNotFoundError{Value: key}
	}
	return nil
}

// ReadAll implements PermissionsRepository.
func (m *MongoDBPermissionsRepository) ReadAll(ctx context.Context, paging shared.PageOptions) ([]Permission, error) {
	collection := m.db.Collection(m.collectionName)
	filter := bson.M{}
	opts := options.Find().SetSkip(int64(paging.Page())).SetLimit(int64(paging.Size()))
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var permissions []Permission
	for cursor.Next(ctx) {
		var permission Permission
		if err := cursor.Decode(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return permissions, nil
}

func NewMongoDBPermissionsRepository(db *mongo.Database) PermissionsRepository {
	repo := &MongoDBPermissionsRepository{
		db:             db,
		collectionName: permissionCollection,
	}

	// Create unique index on key field
	collection := db.Collection(permissionCollection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		fmt.Printf("Warning: failed to create unique index on permission key: %v\n", err)
	}

	return repo
}
