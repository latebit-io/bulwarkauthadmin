package accounts

import "fmt"

type AccountDuplicateError struct {
	Value string `json:"value"`
}

func (e AccountDuplicateError) Error() string {
	return fmt.Sprintf("duplicate account: '%s' already exists", e.Value)
}
