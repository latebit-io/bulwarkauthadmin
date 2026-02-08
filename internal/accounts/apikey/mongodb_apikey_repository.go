package apikey

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	CollectionName = "apiKeys"
)

type MongoDBApiKeyRepository struct {
	db *mongo.Database
}

// Create implements ApiKeyRepository.
func (m *MongoDBApiKeyRepository) Create(ctx context.Context, tenantID, accountID string, key *ApiKey) error {
	key.AccountID = accountID
	key.TenantID = tenantID
	collection := m.db.Collection(CollectionName)
	_, err := collection.InsertOne(ctx, key)
	if err != nil {
		return err
	}

	return nil
}

// Delete implements ApiKeyRepository.
func (m *MongoDBApiKeyRepository) Delete(ctx context.Context, id, tenantID, accountID string) error {
	collection := m.db.Collection(CollectionName)
	_, err := collection.DeleteOne(ctx, bson.M{"id": id, "tenantId": tenantID, "accountId": accountID})
	return err
}

// Ret implements ApiKeyRepository.
func (m *MongoDBApiKeyRepository) Read(ctx context.Context, id, tenantID, accountID string) (*ApiKey, error) {
	collection := m.db.Collection(CollectionName)
	result := collection.FindOne(ctx, bson.M{"id": id, "tenantId": tenantID, "accountId": accountID})
	if err := result.Err(); err != nil {
		return nil, err
	}

	var key ApiKey
	if err := result.Decode(&key); err != nil {
		return nil, err
	}

	return &key, nil
}

// Update implements ApiKeyRepository.
func (m *MongoDBApiKeyRepository) Update(ctx context.Context, tenantID, accountID string, key *ApiKey) error {
	collection := m.db.Collection(CollectionName)
	_, err := collection.UpdateOne(ctx, bson.M{"id": key.ID, "tenantId": tenantID, "accountId": accountID}, bson.M{"$set": key})
	return err
}

func NewMongoDBApiKeyRepository(db *mongo.Database) ApiKeyRepository {
	return &MongoDBApiKeyRepository{
		db: db,
	}
}
