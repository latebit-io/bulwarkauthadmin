package accounts

import (
	"context"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
)

type Account struct {
	Email             string           `bson:"email"`
	IsVerified        bool             `bson:"isVerified"`
	VerificationToken string           `bson:"verificationToken"`
	IsEnabled         bool             `bson:"isEnabled"`
	IsDeleted         bool             `bson:"isDeleted"`
	SocialProviders   []SocialProvider `bson:"socialProviders"`
	Roles             []string         `bson:"roles"`
	Created           time.Time        `bson:"created"`
	Modified          time.Time        `bson:"modified"`
}

type AccountDetails struct {
	ID              string            `json:"id"`
	Email           string            `json:"email"`
	IsVerified      bool              `json:"isVerified"`
	IsEnabled       bool              `json:"isEnabled"`
	IsDeleted       bool              `json:"isDeleted"`
	SocialProviders *[]SocialProvider `json:"socialProviders,omitempty"`
	Roles           []string          `json:"roles"`
	Permissions     []string          `json:"permissions"`
	AuthTokens      []AuthTokenModel  `json:"authTokens"`
	MagicCodes      []MagicCodeModel  `json:"magicCodes"`
	Created         time.Time         `json:"created"`
	Modified        time.Time         `json:"modified"`
}

type SocialProvider struct {
	Name     string `bson:"name" json:"name"`
	SocialId string `bson:"socialId" json:"socialId"`
}

type AuthTokenModel struct {
	ID           string    `json:"id" bson:"_id,omitempty"`
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
	ListAccounts(ctx context.Context, filter AccountFilter) ([]Account, error)
	GetAccountDetails(ctx context.Context, email string) (*AccountDetails, error)
	RegisterAccount(ctx context.Context, email string, options AccountOptions) error
	ChangeAccountEmail(ctx context.Context, email string, options AccountOptions) error
	DisableAccount(ctx context.Context, email string) error
	EnableAccount(ctx context.Context, email string) error
	DeactivateAccount(ctx context.Context, email string) error
	PurgeAccount(ctx context.Context, email string) error
}

type AccountRepository interface {
	Create(ctx context.Context, accountModel Account) error
	ReadByEmail(ctx context.Context, email string) (Account, error)
	ReadById(ctx context.Context, id string) (Account, error)
	ReadAll(ctx context.Context, options shared.PageOptions) ([]Account, error)
	Update(ctx context.Context, account Account) error
	Delete(ctx context.Context, email string) error
}

type AccountManagementServiceDefault struct {
	accountRepository AccountRepository
}

// ChangeAccountEmail implements AccountManagementService.
func (a *AccountManagementServiceDefault) ChangeAccountEmail(ctx context.Context, email string, options AccountOptions) error {
	panic("unimplemented")
}

// DeactivateAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) DeactivateAccount(ctx context.Context, email string) error {
	account, err := a.accountRepository.ReadByEmail(ctx, email)
	if err != nil {
		return err
	}
	account.IsDeleted = true
	err = a.accountRepository.Update(ctx, account)

	if err != nil {
		return err
	}

	return nil
}

// DisableAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) DisableAccount(ctx context.Context, email string) error {
	account, err := a.accountRepository.ReadByEmail(ctx, email)
	if err != nil {
		return err
	}
	account.IsEnabled = true
	err = a.accountRepository.Update(ctx, account)

	if err != nil {
		return err
	}

	return nil
}

// EnableAccount implements AccountManagementService.
func (a *AccountManagementServiceDefault) EnableAccount(ctx context.Context, email string) error {
	account, err := a.accountRepository.ReadByEmail(ctx, email)
	if err != nil {
		return err
	}
	account.IsEnabled = false
	err = a.accountRepository.Update(ctx, account)

	if err != nil {
		return err
	}

	return nil
}

// GetAccountDetails implements AccountManagementService.
func (a *AccountManagementServiceDefault) GetAccountDetails(ctx context.Context, email string) (*AccountDetails, error) {
	account, err := a.accountRepository.ReadByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return &AccountDetails{
		Email:      account.Email,
		IsVerified: account.IsVerified,
		IsEnabled:  account.IsEnabled,
		IsDeleted:  account.IsDeleted,
		//TODO: use other repositories to get the following properties
		SocialProviders: nil,
		Roles:           nil,
		Permissions:     nil,
		MagicCodes:      nil,
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
func (a *AccountManagementServiceDefault) PurgeAccount(ctx context.Context, email string) error {
	err := a.accountRepository.Delete(ctx, email)
	if err != nil {
		return err
	}

	return err
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
		return nil
	}
	return nil
}

// NewAccountManagementServiceDefault default
func NewAccountManagementServiceDefault(accountRepository AccountRepository) AccountManagementService {
	return &AccountManagementServiceDefault{
		accountRepository: accountRepository,
	}
}
