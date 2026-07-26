package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRecordLoginFailureLocksAccountAfterFiveAttempts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.User{}))

	user := &types.User{
		ID:           "u1",
		Username:     "lockout-user",
		Email:        "lockout@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}
	require.NoError(t, db.Create(user).Error)
	repo := NewUserRepository(db)
	ctx := context.Background()

	for attempt := 1; attempt <= 5; attempt++ {
		lockedUntil, recordErr := repo.RecordLoginFailure(ctx, user.ID, 5, 30*time.Minute)
		require.NoError(t, recordErr)
		if attempt < 5 {
			assert.Nil(t, lockedUntil)
		} else {
			require.NotNil(t, lockedUntil)
			assert.WithinDuration(t, time.Now().Add(30*time.Minute), *lockedUntil, 5*time.Second)
		}
	}

	stored, err := repo.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, stored.FailedLoginAttempts)
	assert.NotNil(t, stored.LockedUntil)
}
