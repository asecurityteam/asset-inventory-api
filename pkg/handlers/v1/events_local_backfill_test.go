package v1

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)


func TestBackFillEventsFromTimeErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := NewMockBackFillSchemaRunner(ctrl)

	handler := BackFillEventsLocalHandler{
		LogFn:  testLogFn,
		Runner: mockRunner,
	}

	to := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC)
	err := handler.Handle(context.Background(), BackFillEventsInput{
		From: "not valid time",
		To:   to.Format(time.RFC3339Nano),
	})
	assert.Error(t, err)
}
func TestBackFillEventsToTimeErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := NewMockBackFillSchemaRunner(ctrl)

	handler := BackFillEventsLocalHandler{
		LogFn:  testLogFn,
		Runner: mockRunner,
	}

	from := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC)
	err := handler.Handle(context.Background(), BackFillEventsInput{
		From:   from.Format(time.RFC3339Nano),
		To: "not valid time",
	})
	assert.Error(t, err)
}


func TestBackFillEventsErrDatesNotInSequence(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := NewMockBackFillSchemaRunner(ctrl)

	handler := BackFillEventsLocalHandler{
		LogFn:  testLogFn,
		Runner: mockRunner,
	}

	from := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC)
	to := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC)
	err := handler.Handle(context.Background(), BackFillEventsInput{
		From: to.Format(time.RFC3339Nano), //NB inverted dates, error expected
		To:   from.Format(time.RFC3339Nano),
	})
	assert.Error(t, err)
}

func TestBackFillEventsErrInStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := NewMockBackFillSchemaRunner(ctrl)
	mockRunner.EXPECT().BackFillEventsLocally(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New(""))

	handler := BackFillEventsLocalHandler{
		LogFn:  testLogFn,
		Runner: mockRunner,
	}

	from := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC)
	to := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC)
	err := handler.Handle(context.Background(), BackFillEventsInput{
		From: from.Format(time.RFC3339Nano),
		To:   to.Format(time.RFC3339Nano),
	})
	assert.Error(t, err)
}

func TestBackFillEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRunner := NewMockBackFillSchemaRunner(ctrl)
	mockRunner.EXPECT().BackFillEventsLocally(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	handler := BackFillEventsLocalHandler{
		LogFn:  testLogFn,
		Runner: mockRunner,
	}

	from := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC)
	to := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC)
	err := handler.Handle(context.Background(), BackFillEventsInput{
		From: from.Format(time.RFC3339Nano),
		To:   to.Format(time.RFC3339Nano),
	})
	assert.NoError(t, err)
}
