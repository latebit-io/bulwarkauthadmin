package apikey

import (
	"context"
	"log"

	"github.com/latebit-io/bulwarkauthadmin/internal/accounts"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionName = "apiKeys"
)

type MongoDBApiKeyRepository struct {
	db *mongo.Database
}

// Create implements ApiKeyRepository.
func (m *MongoDBApiKeyRepository) Create(ctx context.Context, tenantID, accountID string, key *ApiKey) error {
	key.AccountID = accountID
	key.TenantID = tenantID
	collection := m.db.Collection(collectionName)
	_, err := collection.InsertOne(ctx, key)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return accounts.ApiKeyDuplicateError{
				Value: key.Name,
			}
		}
		return err
	}

	return nil
}

// Delete implements ApiKeyRepository.
func (m *MongoDBApiKeyRepository) Delete(ctx context.Context, id, tenantID, accountID string) error {
	collection := m.db.Collection(collectionName)
	_, err := collection.DeleteOne(ctx, bson.M{"id": id, "tenantId": tenantID, "accountId": accountID})
	return err
}

// Ret implements ApiKeyRepository.
func (m *MongoDBApiKeyRepository) Read(ctx context.Context, id, tenantID, accountID string) (*ApiKey, error) {
	collection := m.db.Collection(collectionName)
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
	collection := m.db.Collection(collectionName)
	_, err := collection.UpdateOne(ctx, bson.M{"id": key.ID, "tenantId": tenantID, "accountId": accountID}, bson.M{"$set": key})
	return err
}

func NewMongoDBApiKeyRepository(db *mongo.Database) ApiKeyRepository {
	collection := db.Collection(collectionName)
	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "accountId", Value: 1}, {Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatal(err)
	}

	return &MongoDBApiKeyRepository{
		db: db,
	}
}
