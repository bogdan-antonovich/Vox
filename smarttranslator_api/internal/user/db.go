package user

import (
	"context"
	"errors"
	mod "smarttranslator/pkg/models"

	"go.uber.org/zap"
)

type UserDB interface {
	GetUserInfo(ctx context.Context, log *zap.Logger, userID string) (u UserInfo, err error)
	// GetHubs(ctx context.Context, log *zap.Logger, userID string) (hubs []string, err error)
}

type PostgresUser struct{ *mod.Pool }

func NewUserDB(pool *mod.Pool) UserDB {
	return &PostgresUser{Pool: pool}
}

func (pu *PostgresUser) GetUserInfo(ctx context.Context, log *zap.Logger, userID string) (u UserInfo, err error) {
	log.Debug("PostgresUser.GetUserInfo", zap.Bool("ctx_is_nil", ctx == nil), zap.Bool("pool_is_nil", pu.Pool == nil), zap.String("user_id", userID))
	if pu.Pool == nil || ctx == nil {
		log.Error("Invalid input")
		return u, errors.New("invalid input")
	}

	u.ID = userID
	err = pu.QueryRow(ctx, "SELECT email, name, picture_url FROM users WHERE id = $1", u.ID).Scan(&u.Email, &u.Name, &u.Picture)
	if err != nil {
		log.Error("Failed to select from users", zap.Error(err))
		return
	}

	log.Debug("User info retrieved", zap.Any("user_info", u))
	return
}

// func (pu *PostgresUser) GetHubs(ctx context.Context, log *zap.Logger, userID string) (hubs []string, err error) {
// 	rows, err := pu.Query(ctx, "SELECT id FROM hubs WHERE user_id = $1 AND is_active = true", userID)
// 	if err != nil {
// 		log.Error("Failed to select from hubs", zap.Error(err))
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var hubID string
// 		if err := rows.Scan(&hubID); err != nil {
// 			log.Error("Failed to scan hub row", zap.Error(err))
// 			return nil, err
// 		}
// 		hubs = append(hubs, hubID)
// 	}

// 	return
// }
