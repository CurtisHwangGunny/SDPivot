package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SmartKnoraUserProfile extends WeKnora's User with smartKnora-specific fields.
// Stored as a separate table joined on user_id, avoiding modifications to WeKnora core types.
type SmartKnoraUserProfile struct {
	ID       string  `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID   string  `json:"user_id" gorm:"type:varchar(36);uniqueIndex;not null"`
	Phone    *string `json:"phone" gorm:"type:varchar(20);uniqueIndex"`
	Nickname string  `json:"nickname" gorm:"type:varchar(100)"`
	Status   string  `json:"status" gorm:"type:varchar(20);not null;default:active;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (SmartKnoraUserProfile) TableName() string { return "smartknora_user_profiles" }

func (p *SmartKnoraUserProfile) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// OrgExt extends WeKnora's Organization with enterprise certification fields.
type OrgExt struct {
	OrgID         string     `json:"org_id" gorm:"type:varchar(36);primaryKey"`
	AuthStatus    string     `json:"auth_status" gorm:"type:varchar(20);not null;default:trial;index"`
	AuthType      string     `json:"auth_type" gorm:"type:varchar(20)"`
	AuthExpiresAt *time.Time `json:"auth_expires_at"`
	TenantID      uint64     `json:"tenant_id" gorm:"not null;index"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (OrgExt) TableName() string { return "org_ext" }

func (oe *OrgExt) IsTrialExpired() bool {
	if oe.AuthExpiresAt == nil {
		return false
	}
	return time.Now().After(*oe.AuthExpiresAt)
}

func (oe OrgExt) DaysRemaining() int {
	if oe.AuthExpiresAt == nil {
		return 0
	}
	days := int(time.Until(*oe.AuthExpiresAt).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// RefreshToken represents an opaque refresh token stored server-side.
type SmartKnoraOrgMember struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	OrgID     string    `json:"org_id" gorm:"type:varchar(36);not null;index"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Role      string    `json:"role" gorm:"type:varchar(20);not null;default:member"`
	Status    string    `json:"status" gorm:"type:varchar(20);not null;default:active"`
	JoinedAt  time.Time `json:"joined_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SmartKnoraOrgMember) TableName() string { return "org_members" }

func (m *SmartKnoraOrgMember) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// RefreshToken represents an opaque refresh token stored server-side.
type RefreshToken struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	TokenHash string    `json:"-" gorm:"type:varchar(64);not null;uniqueIndex"`
	DeviceID  string    `json:"device_id" gorm:"type:varchar(255)"`
	Family    string    `json:"family" gorm:"type:varchar(36);not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func (rt *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if rt.ID == "" {
		rt.ID = uuid.New().String()
	}
	return nil
}

// SmartKnoraTokenUsage records per-request token consumption for smartKnora metering.
type SmartKnoraTokenUsage struct {
	ID               string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID           *string   `json:"user_id" gorm:"type:varchar(36);index"`
	OrgID            *string   `json:"org_id" gorm:"type:varchar(36);index"`
	TenantID         uint64    `json:"tenant_id" gorm:"not null;index"`
	ModelID          string    `json:"model_id" gorm:"type:varchar(64)"`
	PromptTokens     int       `json:"prompt_tokens" gorm:"not null;default:0"`
	CompletionTokens int       `json:"completion_tokens" gorm:"not null;default:0"`
	TotalTokens      int       `json:"total_tokens" gorm:"not null;default:0"`
	APIPath          string    `json:"api_path" gorm:"type:varchar(255)"`
	CreatedAt        time.Time `json:"created_at" gorm:"index"`
}

func (SmartKnoraTokenUsage) TableName() string { return "token_usage" }

func (tu *SmartKnoraTokenUsage) BeforeCreate(tx *gorm.DB) error {
	if tu.ID == "" {
		tu.ID = uuid.New().String()
	}
	if tu.TotalTokens == 0 {
		tu.TotalTokens = tu.PromptTokens + tu.CompletionTokens
	}
	return nil
}

// TokenUsageSummary is the aggregated result for usage queries.
type SmartKnoraTokenUsageSummary struct {
	TotalPromptTokens     int64 `json:"total_prompt_tokens"`
	TotalCompletionTokens int64 `json:"total_completion_tokens"`
	TotalTokens           int64 `json:"total_tokens"`
	RequestCount          int64 `json:"request_count"`
}

// KnowledgeSpace represents a knowledge workspace.
type KnowledgeSpace struct {
	ID          string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64         `json:"tenant_id" gorm:"not null;index"`
	OrgID       *string        `json:"org_id" gorm:"type:varchar(36);index"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	Description string         `json:"description" gorm:"type:text"`
	Visibility  string         `json:"visibility" gorm:"type:varchar(20);not null;default:team"`
	Icon        string         `json:"icon" gorm:"type:varchar(50)"`
	CreatorID   *string        `json:"creator_id" gorm:"type:varchar(36)"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (KnowledgeSpace) TableName() string { return "knowledge_spaces" }

func (ks *KnowledgeSpace) BeforeCreate(tx *gorm.DB) error {
	if ks.ID == "" {
		ks.ID = uuid.New().String()
	}
	return nil
}

// SpaceMember represents a user's membership in a knowledge space.
type SpaceMember struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	SpaceID   string    `json:"space_id" gorm:"type:varchar(36);not null;uniqueIndex:idx_space_user"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);not null;uniqueIndex:idx_space_user"`
	Role      string    `json:"role" gorm:"type:varchar(20);not null;default:viewer"`
	CreatedAt time.Time `json:"created_at"`
}

func (SpaceMember) TableName() string { return "space_members" }

func (sm *SpaceMember) BeforeCreate(tx *gorm.DB) error {
	if sm.ID == "" {
		sm.ID = uuid.New().String()
	}
	return nil
}

// SpaceCategory represents a tag for organizing knowledge spaces.
type SpaceCategory struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64    `json:"tenant_id" gorm:"not null;index"`
	Name      string    `json:"name" gorm:"type:varchar(100);not null"`
	Color     string    `json:"color" gorm:"type:varchar(20)"`
	CreatedAt time.Time `json:"created_at"`
}

func (SpaceCategory) TableName() string { return "space_categories" }

func (sc *SpaceCategory) BeforeCreate(tx *gorm.DB) error {
	if sc.ID == "" {
		sc.ID = uuid.New().String()
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────
// Request / Response DTOs
// ────────────────────────────────────────────────────────────────────

type SmartKnoraLoginRequest struct {
	Phone    string `json:"phone" binding:"omitempty"`
	Email    string `json:"email" binding:"omitempty"`
	Password string `json:"password" binding:"required"`
}

type SmartKnoraRegisterRequest struct {
	Phone    string `json:"phone" binding:"omitempty"`
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"required,min=8"`
	Nickname string `json:"nickname" binding:"omitempty"`
}

type SmartKnoraAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         *User  `json:"user"`
}

type SmartKnoraRefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UpdateProfileRequest struct {
	Nickname  *string `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
	Phone     *string `json:"phone"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type CreateOrgRequest struct {
	Name        string `json:"name" binding:"required,max=255"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
}

type JoinOrgRequest struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

type CreateSpaceRequest struct {
	Name        string `json:"name" binding:"required,max=255"`
	Description string `json:"description"`
	Visibility  string `json:"visibility" binding:"omitempty,oneof=private team org"`
	Icon        string `json:"icon"`
	OrgID       string `json:"org_id"`
}

type UpdateSpaceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Visibility  *string `json:"visibility"`
	Icon        *string `json:"icon"`
}

type AddSpaceMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"omitempty,oneof=editor viewer"`
}

type CreateCategoryRequest struct {
	Name  string `json:"name" binding:"required,max=100"`
	Color string `json:"color"`
}

type SmartKnoraTokenUsageQuery struct {
	OrgID   *string `form:"org_id"`
	UserID  *string `form:"user_id"`
	StartAt *string `form:"start_at"`
	EndAt   *string `form:"end_at"`
	GroupBy string  `form:"group_by" binding:"omitempty,oneof=day week month"`
}

// WritingDraft represents an AI writing draft.
type WritingDraft struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	TenantID  uint64    `json:"tenant_id" gorm:"not null;index"`
	Title     string    `json:"title" gorm:"type:varchar(500)"`
	Category  string    `json:"category" gorm:"type:varchar(50)"`
	Content   string    `json:"content" gorm:"type:text"`
	SpaceID   string    `json:"space_id" gorm:"type:varchar(36)"`
	Status    string    `json:"status" gorm:"type:varchar(20);default:draft"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WritingDraft) TableName() string { return "writing_drafts" }

// Announcement represents a system announcement.
type Announcement struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Title     string    `json:"title" gorm:"type:varchar(500);not null"`
	Content   string    `json:"content" gorm:"type:text"`
	Status    string    `json:"status" gorm:"type:varchar(20);default:draft"`
	CreatedBy string    `json:"created_by" gorm:"type:varchar(36)"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Announcement) TableName() string { return "announcements" }

// QASession represents a Q&A conversation session.
type QASession struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	TenantID  uint64    `json:"tenant_id" gorm:"not null;index"`
	SpaceID   string    `json:"space_id" gorm:"type:varchar(36)"`
	Title     string    `json:"title" gorm:"type:varchar(500)"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (QASession) TableName() string { return "qa_sessions" }

// QAMessage represents a message in a Q&A session.
type QAMessage struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	SessionID string    `json:"session_id" gorm:"type:varchar(36);not null;index"`
	Role      string    `json:"role" gorm:"type:varchar(20);not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	Sources   string    `json:"sources" gorm:"type:jsonb;default:'[]'"`
	CreatedAt time.Time `json:"created_at"`
}

func (QAMessage) TableName() string { return "qa_messages" }
