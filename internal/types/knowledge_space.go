package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// KnowledgeSpace is an OP space inside an official tenant/workspace.
type KnowledgeSpace struct {
	ID          string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64         `json:"tenant_id" gorm:"not null;index"`
	OrgID       *string        `json:"org_id,omitempty" gorm:"type:varchar(36);index"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	Description string         `json:"description" gorm:"type:text"`
	Visibility  string         `json:"visibility" gorm:"type:varchar(32);not null;default:private"`
	OwnerID     *string        `json:"owner_id,omitempty" gorm:"type:varchar(36)"`
	Icon        string         `json:"icon,omitempty" gorm:"type:varchar(50)"`
	CreatorID   *string        `json:"creator_id,omitempty" gorm:"type:varchar(36)"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (KnowledgeSpace) TableName() string { return "knowledge_spaces" }
func (s *KnowledgeSpace) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}

type SpaceMember struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	SpaceID   string    `json:"space_id" gorm:"type:varchar(36);not null;uniqueIndex:idx_space_user"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);not null;uniqueIndex:idx_space_user"`
	Role      string    `json:"role" gorm:"type:varchar(20);not null;default:viewer"`
	CreatedAt time.Time `json:"created_at"`
}

func (SpaceMember) TableName() string { return "space_members" }
func (m *SpaceMember) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
