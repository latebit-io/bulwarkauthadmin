package accounts

import (
	"context"
	"errors"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
)

type Account struct {
	ID                string           `json:"id" bson:"id"`
	TenantID          string           `json:"tenantId" bson:"tenantId"`
	Email             string           `json:"email" bson:"email"`
	IsVerified        bool             `json:"isVerified" bson:"isVerified"`
	VerificationToken string           `json:"verificationToken" bson:"verificationToken"`
	IsEnabled         bool             `json:"isEnabled" bson:"isEnabled"`
	IsDeleted         bool             `json:"isDeleted" bson:"isDeleted"`
	SocialProviders   []SocialProvider `json:"socialProviders" bson:"socialProviders"`
	Roles             []string         `json:"roles" bson:"roles"`
	Permissions       []string         `json:"permissions" bson:"permissions"`
	Created           time.Time        `json:"created" bson:"created"`
	Modified          time.Time        `json:"modified" bson:"modified"`
}

type AccountDetails struct {
	Account
	AuthTokens []AuthTokenModel `json:"authTokens"`
	MagicCodes []MagicCodeModel `json:"magicCodes"`
}

type SocialProvider struct {
	Name     string `bson:"name" json:"name"`
	SocialId string `bson:"socialId" json:"socialId"`
}

type AuthTokenModel struct {
	ID           string    `json:"id" bson:"id"`
	TenantID     string    `json:"tenantId" bson:"tenantId"`
	UserID       string    `json:"userId" bson:"userId"`
	DeviceID     string    `json:"deviceId" bson:"deviceId"`
	AccessToken  string    `json:"accessToken" bson:"accessToken"`
	RefreshToken string    `json:"refreshToken" bson:"refreshToken"`
	Created      time.Time `json:"created" bson:"created"`
	Modified     time.Time `json:"modified" bson:"modified"`
}

type MagicCodeModel struct {
	ID       string    `json:"id" bson:"_id,omitempty"`
	TenantID string    `json:"tenantId" bson:"tenantId"`
	UserID   string    `json:"userId" bson:"userId"`
	Code     string    `json:"code" bson:"code"`
	Expires  time.Time `json:"expires" bson:"expires"`
	Created  time.Time `json:"created" bson:"created"`
}

type AccountOptions struct {
	IsVerified bool
}

type AccountFilter struct {
	shared.PageOptions
}

type AccountManagementService interface {
	// List accounts with filter
	ListAccounts(ctx context.Context, tenantID string, filter AccountFilter) ([]Account, error)
	// Get account details by ID
	GetAccountDetails(ctx context.Context, tenantID string, id string) (*AccountDetails, error)
	// Register a new account with email and options
	RegisterAccount(ctx context.Context, tenantID string, email string, options AccountOptions) error
	// Unlink SocialProvider
	UnlinkSocialProvider(ctx context.Context, tenantID string, accountID, provider string) error
	// Change account email
	ChangeAccountEmail(ctx context.Context, tenantID string, accountID, newEmail string, options AccountOptions) error
	// Disable account temporarily
	DisableAccount(ctx context.Context, tenantID string, accountId string) error
	// Enable account from being disabled
	EnableAccount(ctx context.Context, tenantID string, accountId string) error
	// Deactivate soft deletes an account
	DeactivateAccount(ctx context.Context, tenantID string, accountId string) error
	// Purge account hard deletes an account
	PurgeAccount(ctx context.Context, tenantID string, accountId string) error
}

type AccountRepository interface {
	Create(ctx context.Context, accountModel Account) error
	ReadByEmail(ctx context.Context, tenantID string, email string) (*Account, error)
	ReadById(ctx context.Context, tenantID string, accountId string) (*Account, error)
	ReadAll(ctx context.Context, tenantID string, options shared.PageOptions) ([]Account, error)
	Update(ctx context.Context, tenantID string, account Account) error
	Delete(ctx context.Context, tenantID string, accountId string) error
}

type AccountManagementServiceDefault struct {
	accountRepository AccountRepository
}

// UnlinkSocialProvider implements AccountManagementService.
func (a *AccountManagementServiceDefault) UnlinkSocialProvider(ctx context.Context, tenantID, accountID, provider string) error {
	account, err := a.accountRepository.ReadById(ctx, tenantID, accountID)
	if err != nil {
		return err
	}
	socials := account.SocialProviders
	for i, social := range socials {
		if social.Name == provider {
			socials = append(socials[:i], socials[i+1:]...)
			account.SocialProviders = socials
			return a.accountRepository.Update(ctx, tenantID, *account)
		}
	}

	return SocialProviderNotFoundError{
		Value: provider,
	}
}

// ChangeAccountEmail implements AccountManagementService.
func (a *AccountManagementServiceDefault) ChangeAccountEmail(ctx context.Context, tenantID, accountID, newEmail string, options AccountOptions) error {
	_, err := a.accountRepository.ReadByEmail(ctx, tenantID, newEmail)
	if err != nil {
		var accountNotFound AccountNotFoundError
		notFound := errors.As(err, &accountNotFound)
		if !notFound {
			return AccountDuplicateError{
				Value: newEmail,
			}
		}
	}

	account, err := a.accountRepository.ReadById(ctx, tenantID, accountID)
	if err != nil {
		return err
	}
	account.IsVerified = options.IsVerified
	account.Email = newEmail

	err = a.accountRepository.Update(ctx, tenantID, *account)
	if err != nil {
		return err
	}

	return nil
}

// DeactivateAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) DeactivateAccount(ctx context.Context, tenantID, accountId string) error {
	account, err := a.accountRepository.ReadById(ctx, tenantID, accountId)
	if err != nil {
		return err
	}
	account.IsDeleted = true
	err = a.accountRepository.Update(ctx, tenantID, *account)

	if err != nil {
		return err
	}

	return nil
}

// DisableAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) DisableAccount(ctx context.Context, tenantID, accountId string) error {
	account, err := a.accountRepository.ReadById(ctx, tenantID, accountId)
	if err != nil {
		return err
	}
	account.IsEnabled = false
	err = a.accountRepository.Update(ctx, tenantID, *account)

	if err != nil {
		return err
	}

	return nil
}

// EnableAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) EnableAccount(ctx context.Context, tenantID, accountId string) error {
	account, err := a.accountRepository.ReadById(ctx, tenantID, accountId)
	if err != nil {
		return err
	}
	account.IsEnabled = true
	err = a.accountRepository.Update(ctx, tenantID, *account)

	if err != nil {
		return err
	}

	return nil
}

// GetAccountDetails implements AccountManagementService.
func (a *AccountManagementServiceDefault) GetAccountDetails(ctx context.Context, tenantID, id string) (*AccountDetails, error) {
	account, err := a.accountRepository.ReadById(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	return &AccountDetails{
		Account:    *account,
		MagicCodes: nil,
	}, nil

}

// ListAccounts implements AccountManagementService.
func (a *AccountManagementServiceDefault) ListAccounts(ctx context.Context, tenantID string, filter AccountFilter) ([]Account, error) {
	accounts, err := a.accountRepository.ReadAll(ctx, tenantID, filter.PageOptions)
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

// PurgeAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) PurgeAccount(ctx context.Context, tenantID, accountId string) error {
	err := a.accountRepository.Delete(ctx, tenantID, accountId)
	if err != nil {
		return err
	}

	return nil
}

// RegisterAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) RegisterAccount(ctx context.Context, tenantID, email string, options AccountOptions) error {
	newAccount := Account{
		TenantID:   tenantID,
		Email:      email,
		IsVerified: true,
		IsEnabled:  true,
		IsDeleted:  false,
	}
	err := a.accountRepository.Create(ctx, newAccount)
	if err != nil {
		return err
	}
	return nil
}

// NewAccountManagementServiceDefault default
func NewAccountManagementServiceDefault(accountRepository AccountRepository) AccountManagementService {
	return &AccountManagementServiceDefault{
		accountRepository: accountRepository,
	}
}
