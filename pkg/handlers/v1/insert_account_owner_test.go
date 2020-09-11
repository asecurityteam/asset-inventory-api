package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
)

func newInsertAccountOwnerHandler(storer domain.AccountOwnerStorer) *AccountOwnerInsertHandler {
	return &AccountOwnerInsertHandler{
		LogFn:              testLogFn,
		StatFn:             testStatFn,
		AccountOwnerStorer: storer,
	}
}

func testInputWithChampion() AccountOwner {
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

func testInputWithoutChampion() AccountOwner {
	return AccountOwner{
		AccountID: "awsaccountid123",
		Owner: Person{
			Name:  "john dane",
			Login: "jdane",
			Email: "jdane@atlassian.com",
			Valid: true,
		},
	}
}

func TestInsertAccountOwnerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := NewMockAccountOwnerStorer(ctrl)
	storage.EXPECT().StoreAccountOwner(gomock.Any(), gomock.Any()).Return(errors.New("error"))

	e := newInsertAccountOwnerHandler(storage).Handle(context.Background(), testInputWithChampion())
	assert.NotNil(t, e)
}

func TestInsertAccountOwnerWithChampion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := NewMockAccountOwnerStorer(ctrl)
	storage.EXPECT().StoreAccountOwner(gomock.Any(), gomock.Any()).Return(nil)

	e := newInsertAccountOwnerHandler(storage).Handle(context.Background(), testInputWithChampion())
	assert.Nil(t, e)
}

func TestInsertAccountOwnerWithoutChampion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := NewMockAccountOwnerStorer(ctrl)
	storage.EXPECT().StoreAccountOwner(gomock.Any(), gomock.Any()).Return(nil)

	e := newInsertAccountOwnerHandler(storage).Handle(context.Background(), testInputWithoutChampion())
	assert.Nil(t, e)
}
