package accounts

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const accountCollection = "accounts"

type MongoDBAccountRepository struct {
	db *mongo.Database
}

func NewMongoDBAccountRepository(db *mongo.Database) AccountRepository {
	collection := db.Collection(accountCollection)
	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatal(err)
	}
	return &MongoDBAccountRepository{
		db: db,
	}
}

// ReadById implements AccountRepository.
func (m *MongoDBAccountRepository) ReadById(ctx context.Context, id string) (*Account, error) {
	collection := m.db.Collection(accountCollection)
	result := collection.FindOne(ctx, bson.D{{Key: "id", Value: id}})
	var account Account
	err := result.Decode(&account)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, AccountNotFoundError{Value: id}
		}
		return nil, err
	}
	return &account, nil
}

// Create implements AccountRepository.
func (m *MongoDBAccountRepository) Create(ctx context.Context, accountModel Account) error {
	var errorMessages []string
	if accountModel.Email == "" {
		errorMessages = append(errorMessages, "email is required")
	}

	if len(errorMessages) > 0 {
		return errors.New(strings.Join(errorMessages, ","))
	}

	collection := m.db.Collection(accountCollection)
	verificationToken, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	// Only users should be able to manage their own passwords, this keeps their privacy intact
	unusablePassword, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	_, err = collection.InsertOne(ctx,
		bson.D{
			{Key: "id", Value: uuid.New().String()},
			{Key: "email", Value: accountModel.Email},
			{Key: "password", Value: unusablePassword.String()},
			{Key: "isVerified", Value: accountModel.IsVerified},
			{Key: "verificationToken", Value: verificationToken.String()},
			{Key: "isEnabled", Value: false},
			{Key: "isDeleted", Value: false},
			{Key: "created", Value: time.Now()},
			{Key: "modified", Value: time.Now()},
		})

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return AccountDuplicateError{
				Value: accountModel.Email,
			}
		}
		return err
	}

	return nil
}

// Delete implements AccountRepository.
func (m *MongoDBAccountRepository) Delete(ctx context.Context, accountId string) error {
	collection := m.db.Collection(accountCollection)
	result, err := collection.DeleteOne(ctx, bson.D{{Key: "id", Value: accountId}})
	if err != nil {
		return nil
	}

	if result.DeletedCount == 0 {
		return AccountNotFoundError{Value: accountId}
	}

	return nil
}

// ReadAll implements AccountRepository.
func (m *MongoDBAccountRepository) ReadAll(ctx context.Context, options shared.PageOptions) ([]Account, error) {
	collection := m.db.Collection(accountCollection)
	var accounts []Account
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return accounts, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var account Account
		err := cursor.Decode(&account)
		if err != nil {
			return accounts, err
		}
		accounts = append(accounts, account)
	}
	if err := cursor.Err(); err != nil {
		return accounts, err
	}
	return accounts, nil
}

// ReadByEmail implements AccountRepository.
func (m *MongoDBAccountRepository) ReadByEmail(ctx context.Context, email string) (*Account, error) {
	collection := m.db.Collection(accountCollection)
	result := collection.FindOne(ctx, bson.D{{Key: "email", Value: email}})
	var account Account
	err := result.Decode(&account)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, AccountNotFoundError{Value: email}
		}
		return nil, err
	}
	return &account, nil
}

// Update implements AccountRepository.
func (m *MongoDBAccountRepository) Update(ctx context.Context, account Account) error {
	collection := m.db.Collection(accountCollection)
	result, err := collection.UpdateOne(ctx, bson.D{{Key: "id", Value: account.ID}}, bson.D{{Key: "$set",
		Value: bson.D{{Key: "email", Value: account.Email}, {Key: "isDeleted", Value: account.IsDeleted},
			{Key: "isEnabled", Value: account.IsEnabled}, {Key: "socialProvider", Value: account.SocialProviders}, {Key: "roles", Value: account.Roles},
			{Key: "modified", Value: time.Now()}}}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return AccountNotFoundError{Value: account.Email}
	}

	return nil
}
