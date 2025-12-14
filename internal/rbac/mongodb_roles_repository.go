package rbac

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const rolesCollection = "roles"

type MongoDBRolesRepository struct {
	db             *mongo.Database
	collectionName string
}

// Create implements RolesRepository.
func (m *MongoDBRolesRepository) Create(ctx context.Context, role Role) error {
	collection := m.db.Collection(m.collectionName)
	_, err := collection.InsertOne(ctx, role)

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return RoleDuplicateError{
				Value: role.Name,
			}
		}
		return err
	}

	return nil
}

// Delete implements RolesRepository.
func (m *MongoDBRolesRepository) Delete(ctx context.Context, roleName string) error {
	collection := m.db.Collection(m.collectionName)
	filter := bson.M{"name": roleName}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return RoleNotFoundError{Value: roleName}
	}
	return nil
}

// Read implements RolesRepository.
func (m *MongoDBRolesRepository) Read(ctx context.Context, roleName string) (*Role, error) {
	collection := m.db.Collection(m.collectionName)
	filter := bson.M{"name": roleName}
	var role Role
	err := collection.FindOne(ctx, filter).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, RoleNotFoundError{Value: roleName}
		}
		return nil, err
	}
	return &role, nil
}

// Update implements RolesRepository.
func (m *MongoDBRolesRepository) Update(ctx context.Context, role Role) error {
	collection := m.db.Collection(m.collectionName)
	filter := bson.M{"name": role.Name}
	update := bson.M{"$set": role}
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func NewMongoDBRolesRepository(db *mongo.Database) RolesRepository {
	repo := &MongoDBRolesRepository{
		db:             db,
		collectionName: rolesCollection,
	}

	// Create unique index on name field
	collection := db.Collection(rolesCollection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		fmt.Printf("Warning: failed to create unique index on role name: %v\n", err)
	}

	return repo
}
