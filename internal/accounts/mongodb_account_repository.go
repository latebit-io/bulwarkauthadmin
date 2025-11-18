package accounts

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const accountCollection = "accounts"

type MongoDBAccountRepository struct {
	db *mongo.Database
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
	newUuid, err := uuid.NewUUID()
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
			{Key: "email", Value: accountModel.Email},
			{Key: "password", Value: unusablePassword.String()},
			{Key: "isVerified", Value: accountModel.IsVerified},
			{Key: "verificationToken", Value: newUuid.String()},
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
func (m *MongoDBAccountRepository) Delete(ctx context.Context, email string) error {
	collection := m.db.Collection(accountCollection)
	result, err := collection.DeleteOne(ctx, bson.D{{Key: "email", Value: email}})
	if err != nil {
		return nil
	}

	if result.DeletedCount == 0 {
		return AccountNotFoundError{Value: email}
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
	result, err := collection.UpdateOne(ctx, bson.D{{Key: "email", Value: account.Email}}, bson.D{{Key: "$set",
		Value: bson.D{{Key: "email", Value: account.Email}, {Key: "isDeleted", Value: account.IsDeleted},
			{Key: "isEnabled", Value: account.IsEnabled}, {Key: "modified", Value: time.Now()}}}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return AccountNotFoundError{Value: account.Email}
	}

	return nil
}

func NewMongoDBAccountRepository(db *mongo.Database) AccountRepository {
	return &MongoDBAccountRepository{
		db: db,
	}
}
