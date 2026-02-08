package apikey

import (
	"context"
	"fmt"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/internal/utils"

	"github.com/google/uuid"
)

const apiKeyPrefix = "api_"

type ApiKey struct {
	ID        string     `json:"id" bson:"id"`
	TenantID  string     `json:"tenantId" bson:"tenantId"`
	AccountID string     `json:"accountId" bson:"accountId"`
	Name      string     `json:"name" bson:"name"`
	KeyHash   string     `json:"-" bson:"key"`
	KeyPrefix string     `json:"keyPrefix" bson:"keyPrefix"`
	IsEnabled bool       `json:"isEnabled" bson:"isEnabled"`
	Expires   *time.Time `json:"expires" bson:"expires"`
	Created   time.Time  `json:"created" bson:"created"`
	Modified  time.Time  `json:"modified" bson:"modified"`
}

type ApiKeyRepository interface {
	Create(ctx context.Context, tenantID, accountID string, key *ApiKey) error
	Read(ctx context.Context, id, tenantID, accountID string) (*ApiKey, error)
	ReadAll(ctx context.Context, tenantID, accountID string) ([]*ApiKey, error)
	Update(ctx context.Context, tenantID, accountID string, key *ApiKey) error
	Delete(ctx context.Context, id, tenantID, accountID string) error
}

type ApiKeyService interface {
	Generate(ctx context.Context, tenantID, accountID, name string, expire *time.Time) (string, error)
	Suspend(ctx context.Context, ID, tenantID, accountID string) error
	Enable(ctx context.Context, ID, tenantID, accountID string) error
	Revoke(ctx context.Context, ID, tenantID, accountID string) error
	List(ctx context.Context, tenantID, accountID string) ([]*ApiKey, error)
	GetKey(ctx context.Context, id, tenantID, accountID string) (*ApiKey, error)
}

type ApiKeyServiceDefault struct {
	repo       ApiKeyRepository
	encryption utils.Encryption
}

func NewApiKeyServiceDefault(repo ApiKeyRepository, encryption utils.Encryption) ApiKeyService {
	return &ApiKeyServiceDefault{
		repo:       repo,
		encryption: encryption,
	}
}

func (s *ApiKeyServiceDefault) Generate(ctx context.Context, tenantID, accountID, name string, expire *time.Time) (string, error) {
	id := uuid.New().String()
	key := uuid.New().String()
	hashKey, err := s.encryption.Encrypt(key)
	if err != nil {
		return "", err
	}
	apiKey := &ApiKey{
		ID:        id,
		TenantID:  tenantID,
		AccountID: accountID,
		Name:      name,
		KeyHash:   hashKey,
		KeyPrefix: apiKeyPrefix,
		IsEnabled: true,
		Expires:   expire,
		Created:   time.Now(),
		Modified:  time.Now(),
	}

	if err := s.repo.Create(ctx, tenantID, accountID, apiKey); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", apiKey.KeyPrefix, key), nil
}

func (s *ApiKeyServiceDefault) Suspend(ctx context.Context, ID, tenantID, accountID string) error {
	key, err := s.repo.Read(ctx, ID, tenantID, accountID)
	if err != nil {
		return err
	}

	key.IsEnabled = false
	key.Modified = time.Now()

	return s.repo.Update(ctx, tenantID, accountID, key)
}

func (s *ApiKeyServiceDefault) Enable(ctx context.Context, ID, tenantID, accountID string) error {
	key, err := s.repo.Read(ctx, ID, tenantID, accountID)
	if err != nil {
		return err
	}

	key.IsEnabled = true
	key.Modified = time.Now()

	return s.repo.Update(ctx, tenantID, accountID, key)
}

func (s *ApiKeyServiceDefault) Revoke(ctx context.Context, ID, tenantID, accountID string) error {
	return s.repo.Delete(ctx, ID, tenantID, accountID)
}

func (s *ApiKeyServiceDefault) List(ctx context.Context, tenantID, accountID string) ([]*ApiKey, error) {
	return s.repo.ReadAll(ctx, tenantID, accountID)
}

func (s *ApiKeyServiceDefault) GetKey(ctx context.Context, id, tenantID, accountID string) (*ApiKey, error) {
	return s.repo.Read(ctx, id, tenantID, accountID)
}
