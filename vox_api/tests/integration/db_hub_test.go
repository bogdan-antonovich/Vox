//go:build integration

package integration

import (
	"context"
	"testing"
	"vox/internal/hub"
	mod "vox/pkg/models"
	"vox/tests/utils/db"
	"vox/tests/utils/helpers"
	"vox/tests/utils/vars"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestGetReference(t *testing.T) {
	cases := []struct {
		name         string
		u            vars.UserForTests
		userID       string
		fileID       string
		nilPool      bool
		nilCtx       bool
		seed         bool
		wantErr      bool
		wantNotOwner bool
		wantPath     string
		wantText     string
		wantFileType string
	}{
		{
			name:         "valid reference retrieved",
			u:            vars.User,
			userID:       vars.User.ID,
			fileID:       vars.File.ID,
			seed:         true,
			wantErr:      false,
			wantPath:     vars.File.Path,
			wantText:     vars.File.Text,
			wantFileType: vars.File.Type,
		},
		{
			name:    "user not found",
			u:       vars.User,
			userID:  "nonexistent-user",
			fileID:  vars.File.ID,
			seed:    false,
			wantErr: true,
		},
		{
			name:         "user is not owner of file",
			u:            vars.User,
			userID:       "nonexistent-user",
			fileID:       vars.File.ID,
			seed:         true,
			wantErr:      true,
			wantNotOwner: true,
		},
		{
			name:    "nil pool",
			u:       vars.User,
			userID:  vars.User.ID,
			fileID:  vars.File.ID,
			nilPool: true,
			wantErr: true,
		},
		{
			name:    "nil ctx",
			u:       vars.User,
			userID:  vars.User.ID,
			fileID:  vars.File.ID,
			nilCtx:  true,
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dbtest := db.NewTestDB(t)
			log := zaptest.NewLogger(t)
			ctx := context.Background()
			if tc.nilCtx {
				ctx = nil
			}
			var dbHub hub.PostgresHub
			if !tc.nilPool {
				dbHub = hub.PostgresHub{Pool: &mod.Pool{Pool: dbtest}}
			}
			if tc.seed {
				helpers.InsertAdditionalUserInfo(t, vars.User, dbtest)
				helpers.InsertFileMetadata(t, vars.File.ID, vars.File.Path, vars.File.Type, vars.File.Text, dbtest)
				helpers.InsertFileAndUser(t, vars.User.ID, vars.File.ID, dbtest)
			}
			path, filetype, text, err := dbHub.GetReference(ctx, log, tc.userID, tc.fileID)
			if tc.wantErr {
				assert.Error(t, err)
				if tc.wantNotOwner {
					assert.ErrorIs(t, err, hub.ErrNotOwner)
				}
				assert.Empty(t, path)
				assert.Empty(t, filetype)
				assert.Empty(t, text)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantPath, path)
				assert.Equal(t, tc.wantFileType, filetype)
				assert.Equal(t, tc.wantText, text)
			}
		})
	}
}
