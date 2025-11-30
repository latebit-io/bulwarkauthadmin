package accounts

import (
	"context"
	"errors"
	"testing"

	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
	"github.com/latebit-io/bulwarkauthadmin/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupMongoServer(t *testing.T) (*mongo.Client, *mongo.Database) {
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}

	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		t.Fatal(err)
	}

	return client, client.Database("bulwark")
}

func cleanupMongoServer(t *testing.T, client *mongo.Client) {
	err := client.Disconnect(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
}

func TestMongoDBAccountRepository_Create(t *testing.T) {
	tests := []struct {
		name        string
		account     Account
		expectedErr error
	}{
		{
			"Valid Account",
			Account{Email: "test@latebit.io", IsVerified: false},
			nil,
		},
		{
			"Empty Email",
			Account{Email: "", IsVerified: false},
			errors.New("email is required"),
		},
		{
			"Duplicate Email",
			Account{Email: "test@latebit.io", IsVerified: false},
			AccountDuplicateError{Value: "test@latebit.io"},
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBAccountRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(context.TODO(), tt.account)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMongoDBAccountRepository_ReadById(t *testing.T) {
	tests := []struct {
		name          string
		accountEmail  string
		searchId      string
		expectedEmail string
		expectedErr   bool
	}{
		{
			"Valid Account",
			"test@latebit.io",
			"", // Will be set after creation
			"test@latebit.io",
			false,
		},
		{
			"Account Not Found",
			"",
			"nonexistent-id",
			"",
			true,
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBAccountRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.accountEmail != "" {
				account := Account{Email: tt.accountEmail, IsVerified: false}
				err := repo.Create(context.TODO(), account)
				assert.NoError(t, err)

				createdAccount, err := repo.ReadByEmail(context.TODO(), tt.accountEmail)
				assert.NoError(t, err)
				tt.searchId = createdAccount.ID
			}

			account, err := repo.ReadById(context.TODO(), tt.searchId)

			if tt.expectedErr {
				assert.Error(t, err)
				var notFoundErr AccountNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedEmail, account.Email)
			}
		})
	}
}

func TestMongoDBAccountRepository_ReadByEmail(t *testing.T) {
	tests := []struct {
		name          string
		accountEmail  string
		searchEmail   string
		expectedEmail string
		expectedErr   bool
	}{
		{
			"Valid Account",
			"test@latebit.io",
			"test@latebit.io",
			"test@latebit.io",
			false,
		},
		{
			"Account Not Found",
			"",
			"notfound@latebit.io",
			"",
			true,
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBAccountRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.accountEmail != "" {
				account := Account{Email: tt.accountEmail, IsVerified: false}
				err := repo.Create(context.TODO(), account)
				assert.NoError(t, err)
			}

			account, err := repo.ReadByEmail(context.TODO(), tt.searchEmail)

			if tt.expectedErr {
				assert.Error(t, err)
				var notFoundErr AccountNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedEmail, account.Email)
			}
		})
	}
}

func TestMongoDBAccountRepository_ReadAll(t *testing.T) {
	tests := []struct {
		name          string
		accountEmails []string
		expectedCount int
	}{
		{
			"Multiple Accounts",
			[]string{"test1@latebit.io", "test2@latebit.io", "test3@latebit.io"},
			3,
		},
		{
			"Empty Database",
			[]string{},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, db := setupMongoServer(t)
			defer cleanupMongoServer(t, client)

			repo := NewMongoDBAccountRepository(db)

			for _, email := range tt.accountEmails {
				account := Account{Email: email, IsVerified: false}
				err := repo.Create(context.TODO(), account)
				assert.NoError(t, err)
			}

			accounts, err := repo.ReadAll(context.TODO(), shared.NewPageOptions(0, 0, ""))

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(accounts))
		})
	}
}

func TestMongoDBAccountRepository_Update(t *testing.T) {
	tests := []struct {
		name         string
		accountEmail string
		newEmail     string
		isDeleted    bool
		isEnabled    bool
		expectedErr  bool
	}{
		{
			"Update Email",
			"test@latebit.io",
			"newemail@latebit.io",
			false,
			true,
			false,
		},
		{
			"Update IsDeleted Flag",
			"test2@latebit.io",
			"test2@latebit.io",
			true,
			true,
			false,
		},
		{
			"Update IsEnabled Flag",
			"test3@latebit.io",
			"test3@latebit.io",
			false,
			false,
			false,
		},
		{
			"Account Not Found",
			"",
			"newemail@latebit.io",
			false,
			true,
			true,
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBAccountRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var accountId string
			if tt.accountEmail != "" {
				account := Account{Email: tt.accountEmail, IsVerified: false}
				err := repo.Create(context.TODO(), account)
				assert.NoError(t, err)

				createdAccount, err := repo.ReadByEmail(context.TODO(), tt.accountEmail)
				assert.NoError(t, err)
				accountId = createdAccount.ID
			}

			accountToUpdate := Account{
				ID:        accountId,
				Email:     tt.newEmail,
				IsDeleted: tt.isDeleted,
				IsEnabled: tt.isEnabled,
			}

			err := repo.Update(context.TODO(), accountToUpdate)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				updatedAccount, err := repo.ReadById(context.TODO(), accountId)
				assert.NoError(t, err)
				assert.Equal(t, tt.newEmail, updatedAccount.Email)
				assert.Equal(t, tt.isDeleted, updatedAccount.IsDeleted)
				assert.Equal(t, tt.isEnabled, updatedAccount.IsEnabled)
			}
		})
	}
}

func TestMongoDBAccountRepository_Delete(t *testing.T) {
	tests := []struct {
		name         string
		accountEmail string
		deleteId     string
		expectedErr  bool
	}{
		{
			"Valid Delete",
			"test@latebit.io",
			"", // Will be set after creation
			false,
		},
		{
			"Account Not Found",
			"",
			"nonexistent-id",
			true,
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBAccountRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.accountEmail != "" {
				account := Account{Email: tt.accountEmail, IsVerified: false}
				err := repo.Create(context.TODO(), account)
				assert.NoError(t, err)

				createdAccount, err := repo.ReadByEmail(context.TODO(), tt.accountEmail)
				assert.NoError(t, err)
				tt.deleteId = createdAccount.ID
			}

			err := repo.Delete(context.TODO(), tt.deleteId)

			if tt.expectedErr {
				assert.Error(t, err)
				var notFoundErr AccountNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			} else {
				assert.NoError(t, err)

				_, err := repo.ReadById(context.TODO(), tt.deleteId)
				assert.Error(t, err)
				var notFoundErr AccountNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			}
		})
	}
}
