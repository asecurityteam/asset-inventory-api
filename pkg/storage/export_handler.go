package storage

import (
	"context"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
)

// LocalDBExportHandler implements local handler for individual events that stores them into v2 schema
type LocalDBExportHandler struct {
	db *DB
}

// NewLocalExportHandler returns local handler for individual events that stores them into v2 schema
func NewLocalExportHandler(db *DB) *LocalDBExportHandler {
	return &LocalDBExportHandler{db: db}
}

// Handle writes individual events into v2 schema
func (h *LocalDBExportHandler) Handle(changes domain.CloudAssetChanges) error {
	return h.db.StoreV2(context.Background(), changes)
}
