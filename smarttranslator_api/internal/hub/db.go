package hub

import (
	"context"
	"errors"
	"smarttranslator/pkg/helpers"
	mod "smarttranslator/pkg/models"

	"go.uber.org/zap"
)

type HubDB interface {
	GetReference(ctx context.Context, log *zap.Logger, userID, fileID string) (path, filetype, text string, err error)
	// NewHub(ctx context.Context, log *zap.Logger, userID, hubID string) (err error)
	// DeleteHub(ctx context.Context, log *zap.Logger, hubID string) (err error)
	// IsOwner(ctx context.Context, log *zap.Logger, hubID, userID string) (ok bool, err error)
}

type PostgresHub struct{ *mod.Pool }

func NewHubDB(pool *mod.Pool) HubDB {
	return &PostgresHub{Pool: pool}
}

var ErrNotOwner = errors.New("not the owner")

func (ph *PostgresHub) GetReference(ctx context.Context, log *zap.Logger, userID, fileID string) (path, filetype, text string, err error) {
	log.Debug("PostgresHub.GetReference", zap.Bool("ctx_is_nil", ctx == nil), zap.Bool("pool_is_nil", ph.Pool == nil), zap.String("userID", userID))
	if ph.Pool == nil || ctx == nil {
		log.Error("Invalid input")
		return path, filetype, text, errors.New("invalid input")
	}

	tx, err := ph.Begin(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", zap.Error(err))
		return
	}

	defer helpers.CommitOrRollback(ctx, tx, err, log)

	var count int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM files_and_users WHERE user_id = $1 AND file_id = $2 and is_active = TRUE", userID, fileID).Scan(&count)
	if err != nil {
		log.Error("Failed to select from files_and_users", zap.Error(err))
		return
	}

	if count == 0 {
		log.Error("File not found or not active", zap.String("userID", userID), zap.String("fileID", fileID))
		return path, filetype, text, ErrNotOwner
	}

	err = tx.QueryRow(ctx, "SELECT full_path, type, text FROM files WHERE id = $1", fileID).Scan(&path, &filetype, &text)
	if err != nil {
		log.Error("Failed to select from files", zap.Error(err))
		return
	}

	log.Debug("Voice reference retrieved", zap.String("userID", userID), zap.String("path", path), zap.String("filetype", filetype), zap.String("text", text))
	return
}

// func (ph *PostgresHub) NewHub(ctx context.Context, log *zap.Logger, userID, hubID string) (err error) {
// 	tx, err := ph.Begin(ctx)
// 	if err != nil {
// 		log.Error("Failed to begin transaction", zap.Error(err))
// 		return
// 	}

// 	defer helpers.CommitOrRollback(ctx, tx, err, log)

// 	_, err = tx.Exec(ctx, "INSERT INTO hubs (id, user_id, is_active) VALUES ($1, $2, true)", hubID, userID)
// 	if err != nil {
// 		log.Error("Failed to insert into hubs", zap.Error(err))
// 		return
// 	}

// 	return
// }

// func (ph *PostgresHub) DeleteHub(ctx context.Context, log *zap.Logger, hubID string) (err error) {
// 	tx, err := ph.Begin(ctx)
// 	if err != nil {
// 		log.Error("Failed to begin transaction", zap.Error(err))
// 		return
// 	}

// 	defer helpers.CommitOrRollback(ctx, tx, err, log)

// 	_, err = tx.Exec(ctx, "UPDATE hubs SET is_active = false WHERE id = $1", hubID)
// 	if err != nil {
// 		log.Error("Failed to delete from hubs", zap.Error(err))
// 		return
// 	}

// 	return
// }

// func (ph *PostgresHub) IsOwner(ctx context.Context, log *zap.Logger, hubID, userID string) (ok bool, err error) {
// 	var count int
// 	err = ph.QueryRow(ctx, "SELECT COUNT(*) FROM users_and_hubs WHERE user_id = $1 AND hub_id = $2 AND is_active = true", userID, hubID).Scan(&count)
// 	if err != nil {
// 		log.Error("Failed to select from hubs", zap.Error(err))
// 		return
// 	}

// 	return count > 0, nil
// }
