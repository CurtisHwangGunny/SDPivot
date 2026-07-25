package interfaces

import (
	"context"
	"io"

	"github.com/Tencent/WeKnora/internal/types"
)

type BackupService interface {
	Start(ctx context.Context) error
	Stop()
	Create(ctx context.Context, actorID, trigger string) (*types.BackupRecord, error)
	List(ctx context.Context, limit, offset int) ([]*types.BackupRecord, int64, error)
	Get(ctx context.Context, id string) (*types.BackupRecord, error)
	Open(ctx context.Context, id string) (io.ReadCloser, *types.BackupRecord, error)
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	GetSchedule(ctx context.Context) (*types.BackupScheduleConfig, error)
	UpdateSchedule(ctx context.Context, config *types.BackupScheduleConfig) error
}
