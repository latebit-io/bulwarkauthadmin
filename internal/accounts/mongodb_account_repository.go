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
	collection := a.db.Collection(accountCollection)
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

// ReadAll implements AccountRepository.
func (m *MongoDBAccountRepository) ReadAll(ctx context.Context, options shared.PageOptions) ([]Account, error) {
	panic("unimplemented")
}

// ReadByEmail implements AccountRepository.
func (m *MongoDBAccountRepository) ReadByEmail(ctx context.Context, email string) (Account, error) {
	panic("unimplemented")
}

// ReadById implements AccountRepository.
func (m *MongoDBAccountRepository) ReadById(ctx context.Context, id string) (Account, error) {
	panic("unimplemented")
}

// Update implements AccountRepository.
func (m *MongoDBAccountRepository) Update(ctx context.Context, account Account) error {
	panic("unimplemented")
}

func NewMongoDBAccountRepository(db *mongo.Database) AccountRepository {
	return &MongoDBAccountRepository{
		db: db,
	}
}
