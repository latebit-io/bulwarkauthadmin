package tenants

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/latebit-io/bulwarkauthadmin/internal/email"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	tenantCollection = "tenants"
)

type Tenant struct {
	ID          string    `bson:"id" json:"id"`
	Name        string    `bson:"name" json:"name"`
	Description string    `bson:"description" json:"description"`
	Domain      string    `bson:"domain" json:"domain"`
	Created     time.Time `bson:"created" json:"created_at"`
	Modified    time.Time `bson:"modified" json:"modified_at"`
}

type TenantRepository interface {
	ReadAll(ctx context.Context) ([]Tenant, error)
	Read(ctx context.Context, tenantID string) (*Tenant, error)
	Create(ctx context.Context, tenant Tenant) (string, error)
	CreateSystem(ctx context.Context) error
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
	repo := &MongoDbTenantRepository{
		db: db,
	}

	// Create unique index on name field
	collection := db.Collection(tenantCollection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		fmt.Printf("Warning: failed to create unique index on tenant name: %v\n", err)
	}

	return repo
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
		if err == mongo.ErrNoDocuments {
			return nil, TenantNotFoundError{Value: tenantID}
		}
		return nil, err
	}
	return &tenant, nil
}

func (t *MongoDbTenantRepository) Create(ctx context.Context, tenant Tenant) (string, error) {
	collection := t.db.Collection(tenantCollection)
	tenant.ID = uuid.New().String()
	tenant.Created = time.Now()
	tenant.Modified = time.Now()
	_, err := collection.InsertOne(ctx, tenant)

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", TenantDuplicateError{
				Value: tenant.Name,
			}
		}
		return "", err
	}

	return tenant.ID, nil
}

func (t *MongoDbTenantRepository) CreateSystem(ctx context.Context) error {
	collection := t.db.Collection(tenantCollection)
	systemTenant := Tenant{
		ID:          uuid.Nil.String(),
		Name:        "System",
		Description: "System tenant",
		Created:     time.Now(),
		Modified:    time.Now(),
	}
	_, err := collection.InsertOne(ctx, systemTenant)
	if err != nil {
		// If system tenant already exists, that's fine - this is idempotent
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return err
	}
	return nil
}

func (t *MongoDbTenantRepository) Update(ctx context.Context, tenant Tenant) error {
	collection := t.db.Collection(tenantCollection)
	_, err := collection.UpdateOne(ctx, bson.M{"id": tenant.ID}, bson.M{"$set": tenant})
	return err
}

func (t *MongoDbTenantRepository) Delete(ctx context.Context, tenantID string) error {
	collection := t.db.Collection(tenantCollection)
	result, err := collection.DeleteOne(ctx, bson.M{"id": tenantID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return TenantNotFoundError{Value: tenantID}
	}
	return err
}

type DefaultTenantService struct {
	repo         TenantRepository
	emailService email.EmailService
}

func NewDefaultTenantService(repo TenantRepository, emailService email.EmailService) TenantService {
	return &DefaultTenantService{
		repo:         repo,
		emailService: emailService,
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
	tenantID, err := s.repo.Create(ctx, newTenant)
	if err != nil {
		return err
	}
	err = s.emailService.CreateDefaultTemplates(ctx, tenantID)
	if err != nil {
		return err
	}
	return nil
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
