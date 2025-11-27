package accounts

import (
	"context"
	"errors"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
)

type Account struct {
	ID                string           `json:"id" bson:"id"`
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
	UserID       string    `json:"userId" bson:"userId"`
	DeviceID     string    `json:"deviceId" bson:"deviceId"`
	AccessToken  string    `json:"accessToken" bson:"accessToken"`
	RefreshToken string    `json:"refreshToken" bson:"refreshToken"`
	Created      time.Time `json:"created" bson:"created"`
	Modified     time.Time `json:"modified" bson:"modified"`
}

type MagicCodeModel struct {
	ID      string    `json:"id" bson:"_id,omitempty"`
	UserID  string    `json:"userId" bson:"userId"`
	Code    string    `json:"code" bson:"code"`
	Expires time.Time `json:"expires" bson:"expires"`
	Created time.Time `json:"created" bson:"created"`
}

type AccountOptions struct {
	IsVerified bool
}

type AccountFilter struct {
	shared.PageOptions
}

type AccountManagementService interface {
	// List accounts with filter
	ListAccounts(ctx context.Context, filter AccountFilter) ([]Account, error)
	// Get account details by ID
	GetAccountDetails(ctx context.Context, id string) (*AccountDetails, error)
	// Register a new account with email and options
	RegisterAccount(ctx context.Context, email string, options AccountOptions) error
	// Unlink SocialProvider
	UnlinkSocialProvider(ctx context.Context, accountID, provider string) error
	// Change account email
	ChangeAccountEmail(ctx context.Context, accountID, newEmail string, options AccountOptions) error
	// Disable account temporarily
	DisableAccount(ctx context.Context, accountId string) error
	// Enable account from being disabled
	EnableAccount(ctx context.Context, accountId string) error
	// Deactivate soft deletes an account
	DeactivateAccount(ctx context.Context, accountId string) error
	// Purge account hard deletes an account
	PurgeAccount(ctx context.Context, accountId string) error
}

type AccountRepository interface {
	Create(ctx context.Context, accountModel Account) error
	ReadByEmail(ctx context.Context, email string) (*Account, error)
	ReadById(ctx context.Context, accountId string) (*Account, error)
	ReadAll(ctx context.Context, options shared.PageOptions) ([]Account, error)
	Update(ctx context.Context, account Account) error
	Delete(ctx context.Context, accountId string) error
}

type AccountManagementServiceDefault struct {
	accountRepository AccountRepository
}

// UnlinkSocialProvider implements AccountManagementService.
func (a *AccountManagementServiceDefault) UnlinkSocialProvider(ctx context.Context, accountID, provider string) error {
	account, err := a.accountRepository.ReadById(ctx, accountID)
	if err != nil {
		return err
	}
	socials := account.SocialProviders
	for i, social := range socials {
		if social.Name == provider {
			socials = append(socials[:i], socials[i+1:]...)
			account.SocialProviders = socials
			return a.accountRepository.Update(ctx, *account)
		}
	}

	return SocialProviderNotFoundError{
		Value: provider,
	}
}

// ChangeAccountEmail implements AccountManagementService.
func (a *AccountManagementServiceDefault) ChangeAccountEmail(ctx context.Context, accountID, newEmail string, options AccountOptions) error {
	_, err := a.accountRepository.ReadByEmail(ctx, newEmail)
	if err != nil {
		var accountNotFound AccountNotFoundError
		duplicate := errors.As(err, &accountNotFound)
		if duplicate == false {
			return err
		}
	}

	account, err := a.accountRepository.ReadById(ctx, accountID)
	if err != nil {
		return err
	}
	account.IsVerified = options.IsVerified
	account.Email = newEmail

	err = a.accountRepository.Update(ctx, *account)
	if err != nil {
		return err
	}

	return nil
}

// DeactivateAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) DeactivateAccount(ctx context.Context, accountId string) error {
	account, err := a.accountRepository.ReadById(ctx, accountId)
	if err != nil {
		return err
	}
	account.IsDeleted = true
	err = a.accountRepository.Update(ctx, *account)

	if err != nil {
		return err
	}

	return nil
}

// DisableAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) DisableAccount(ctx context.Context, accountId string) error {
	account, err := a.accountRepository.ReadById(ctx, accountId)
	if err != nil {
		return err
	}
	account.IsEnabled = false
	err = a.accountRepository.Update(ctx, *account)

	if err != nil {
		return err
	}

	return nil
}

// EnableAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) EnableAccount(ctx context.Context, accountId string) error {
	account, err := a.accountRepository.ReadById(ctx, accountId)
	if err != nil {
		return err
	}
	account.IsEnabled = true
	err = a.accountRepository.Update(ctx, *account)

	if err != nil {
		return err
	}

	return nil
}

// GetAccountDetails implements AccountManagementService.
func (a *AccountManagementServiceDefault) GetAccountDetails(ctx context.Context, id string) (*AccountDetails, error) {
	account, err := a.accountRepository.ReadById(ctx, id)
	if err != nil {
		return nil, err
	}

	return &AccountDetails{
		Account: *account,

		MagicCodes: nil,
	}, nil

}

// ListAccounts implements AccountManagementService.
func (a *AccountManagementServiceDefault) ListAccounts(ctx context.Context, filter AccountFilter) ([]Account, error) {
	accounts, err := a.accountRepository.ReadAll(ctx, filter.PageOptions)
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

// PurgeAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) PurgeAccount(ctx context.Context, accountId string) error {
	err := a.accountRepository.Delete(ctx, accountId)
	if err != nil {
		return err
	}

	return nil
}

// RegisterAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) RegisterAccount(ctx context.Context, email string, options AccountOptions) error {
	newAccount := Account{
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
