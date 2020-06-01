package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newInsertAccountOwnerHandler(storer domain.AccountOwnerStorer) *AccountOwnerInsertHandler {
	return &AccountOwnerInsertHandler{
		LogFn:              testLogFn,
		StatFn:             testStatFn,
		AccountOwnerStorer: storer,
	}
}

func validInput() AccountOwner {
	return AccountOwner{
		AccountID: "awsaccountid123",
		Owner: Person{
			Name:  "john dane",
			Login: "jdane",
			Email: "jdane@atlassian.com",
			Valid: true,
		},
		Champions: []Person{
			{
				Name:  "john dane",
				Login: "jdane",
				Email: "jdane@atlassian.com",
				Valid: true,
			},
		},
	}
}

func TestInsertAccountOwnerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := NewMockAccountOwnerStorer(ctrl)
	storage.EXPECT().StoreAccountOwner(gomock.Any(), gomock.Any()).Return(errors.New("error"))

	e := newInsertAccountOwnerHandler(storage).Handle(context.Background(), validInput())
	assert.NotNil(t, e)
}
