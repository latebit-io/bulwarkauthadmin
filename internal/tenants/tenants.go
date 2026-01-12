package tenants

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	tenantCollection = "tenants"
)

type Tenant struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Domain      string    `json:"domain"`
	Created     time.Time `json:"created_at"`
	Modified    time.Time `json:"modified_at"`
}

type TenantRepository interface {
	ReadAll(ctx context.Context) ([]Tenant, error)
	Read(ctx context.Context, tenantID string) (*Tenant, error)
	Create(ctx context.Context, tenant Tenant) error
	Update(ctx context.Context, tenant Tenant) error
	Delete(ctx context.Context, tenantID string) error
}

type TenantService interface {
	ListTenants(ctx context.Context) ([]Tenant, error)
	GetTenant(ctx context.Context, tenantID string) (*Tenant, error)
	AddTenant(ctx context.Context, name, description, domain string) error
	UpdateTenant(ctx context.Context, tenantID, name, description, domain string) error
	RemoveTenant(ctx context.Context, tenantID string) error
}

type MongoDbTenantRepository struct {
	db *mongo.Database
}

func NewMongoDbTenantRepository(db *mongo.Database) TenantRepository {
	return &MongoDbTenantRepository{
		db: db,
	}
}

func (t *MongoDbTenantRepository) ReadAll(ctx context.Context) ([]Tenant, error) {
	collection := t.db.Collection(tenantCollection)
	var tenants []Tenant

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var tenant Tenant
		err := cursor.Decode(&tenant)
		if err != nil {
			return tenants, err
		}
		tenants = append(tenants, tenant)
	}
	if err := cursor.Err(); err != nil {
		return tenants, err
	}
	return tenants, nil
}

func (t *MongoDbTenantRepository) Read(ctx context.Context, tenantID string) (*Tenant, error) {
	collection := t.db.Collection(tenantCollection)
	filter := bson.M{"id": tenantID}
	var tenant Tenant
	err := collection.FindOne(ctx, filter).Decode(&tenant)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (t *MongoDbTenantRepository) Create(ctx context.Context, tenant Tenant) error {
	collection := t.db.Collection(tenantCollection)
	tenant.ID = uuid.New().String()
	tenant.Created = time.Now()
	tenant.Modified = time.Now()
	_, err := collection.InsertOne(ctx, tenant)
	return err
}

func (t *MongoDbTenantRepository) Update(ctx context.Context, tenant Tenant) error {
	collection := t.db.Collection(tenantCollection)
	_, err := collection.UpdateOne(ctx, bson.M{"id": tenant.ID}, bson.M{"$set": tenant})
	return err
}

func (t *MongoDbTenantRepository) Delete(ctx context.Context, tenantID string) error {
	collection := t.db.Collection(tenantCollection)
	_, err := collection.DeleteOne(ctx, bson.M{"id": tenantID})
	return err
}

type DefaultTenantService struct {
	repo TenantRepository
}

func NewDefaultTenantService(repo TenantRepository) TenantService {
	return &DefaultTenantService{
		repo: repo,
	}
}

func (s *DefaultTenantService) ListTenants(ctx context.Context) ([]Tenant, error) {
	return s.repo.ReadAll(ctx)
}

func (s *DefaultTenantService) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
	return s.repo.Read(ctx, tenantID)
}

func (s *DefaultTenantService) AddTenant(ctx context.Context, name, description, domain string) error {
	newTenant := Tenant{
		Name:        name,
		Description: description,
		Domain:      domain,
	}
	return s.repo.Create(ctx, newTenant)
}

func (s *DefaultTenantService) UpdateTenant(ctx context.Context, tenantID, name, description, domain string) error {
	tenant := Tenant{
		ID:          tenantID,
		Name:        name,
		Description: description,
		Domain:      domain,
	}
	return s.repo.Update(ctx, tenant)
}

func (s *DefaultTenantService) RemoveTenant(ctx context.Context, tenantID string) error {
	return s.repo.Delete(ctx, tenantID)
}
