package accounts

import "fmt"

type AccountDuplicateError struct {
	Value string `json:"value"`
}

func (e AccountDuplicateError) Error() string {
	return fmt.Sprintf("duplicate account: '%s' already exists", e.Value)
}

type AccountNotFoundError struct {
	Value string `json:"value"`
}

func (e AccountNotFoundError) Error() string {
	return fmt.Sprintf("account not found: '%s'", e.Value)
}

type SocialProviderNotFoundError struct {
	Value string `json:"value"`
}

func (e SocialProviderNotFoundError) Error() string {
	return fmt.Sprintf("social provider not found: '%s'", e.Value)
}

type ApiKeyDuplicateError struct {
	Value string `json:"value"`
}

func (e ApiKeyDuplicateError) Error() string {
	return fmt.Sprintf("duplicate apiKey: '%s' already exists", e.Value)
}
