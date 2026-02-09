package apikey

import (
	"context"
	"errors"
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
	result, err := collection.DeleteOne(ctx, bson.M{"id": id, "tenantId": tenantID, "accountId": accountID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return accounts.ApiKeyNotFoundError{
			Value: id,
		}
	}
	return nil
}

// Read implements ApiKeyRepository.
func (m *MongoDBApiKeyRepository) Read(ctx context.Context, id, tenantID, accountID string) (*ApiKey, error) {
	collection := m.db.Collection(collectionName)
	result := collection.FindOne(ctx, bson.M{"id": id, "tenantId": tenantID, "accountId": accountID})
	var key ApiKey
	err := result.Decode(&key)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, accounts.ApiKeyNotFoundError{Value: id}
		}
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

func (m *MongoDBApiKeyRepository) ReadAll(ctx context.Context, tenantID, accountID string) ([]*ApiKey, error) {
	collection := m.db.Collection(collectionName)
	cursor, err := collection.Find(ctx, bson.M{"tenantId": tenantID, "accountId": accountID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var keys []*ApiKey
	for cursor.Next(ctx) {
		var key ApiKey
		if err := cursor.Decode(&key); err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return keys, nil
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
