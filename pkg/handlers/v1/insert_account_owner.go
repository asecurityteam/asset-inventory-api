package v1

import (
	"context"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/asecurityteam/asset-inventory-api/pkg/logs"
)

// AccountOwner represents an incoming AWS account with its owner and account champions to be inserted or updated
type AccountOwner struct {
	AccountID string   `json:"accountId"`
	Owner     Person   `json:"owner"`
	Champions []Person `json:"champions"`
}

// Person represents an incoming details about an AWS account owner and/or champion to be inserted or updated
type Person struct {
	Name  string `json:"name"`
	Login string `json:"login"`
	Email string `json:"email"`
	Valid bool   `json:"valid"`
}

// AccountOwnerInsertHandler defines a lambda handler for updating or inserting account owner and account ID
type AccountOwnerInsertHandler struct {
	LogFn              domain.LogFn
	StatFn             domain.StatFn
	AccountOwnerStorer domain.AccountOwnerStorer
}

// Handle handles the insert or update operation for account owner
func (h *AccountOwnerInsertHandler) Handle(ctx context.Context, input AccountOwner) error {
	logger := h.LogFn(ctx)

	accountOwner := domain.AccountOwner{
		AccountID: input.AccountID,
		Owner: domain.Person{
			Name:  input.Owner.Name,
			Login: input.Owner.Login,
			Email: input.Owner.Email,
			Valid: input.Owner.Valid,
		},
		Champions: make([]domain.Person, 0, len(input.Champions)),
	}
	for _, val := range input.Champions {
		accountOwner.Champions = append(accountOwner.Champions, domain.Person{
			Name:  val.Name,
			Login: val.Login,
			Email: val.Email,
			Valid: val.Valid,
		})
	}

	if e := h.AccountOwnerStorer.Store(ctx, accountOwner); e != nil {
		logger.Error(logs.StorageError{Reason: e.Error()})
		return e
	}
	return nil

}
