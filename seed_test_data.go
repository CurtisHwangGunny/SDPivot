package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	TenantID       uint64         `json:"tenant_id"`
	OwnerEmail     string         `json:"owner_email"`
	Organization   OrganizationIn `json:"organization"`
	Spaces         []SpaceIn      `json:"spaces"`
	QASessions     []QAIn         `json:"qa_sessions"`
	WritingDrafts  []DraftIn      `json:"writing_drafts"`
	TokenUsageDays int            `json:"token_usage_days"`
}

type OrganizationIn struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	AuthStatus  string `json:"auth_status"`
	AuthType    string `json:"auth_type"`
}

type SpaceIn struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Visibility  string       `json:"visibility"`
	Icon        string       `json:"icon"`
	Members     []MemberIn   `json:"members"`
	Documents   []DocumentIn `json:"documents"`
}

type MemberIn struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type DocumentIn struct {
	Title    string   `json:"title"`
	FileName string   `json:"file_name"`
	FileType string   `json:"file_type"`
	Tags     string   `json:"tags"`
	Chunks   []string `json:"chunks"`
}

type QAIn struct {
	Title    string      `json:"title"`
	Space    string      `json:"space"`
	Messages []MessageIn `json:"messages"`
}

type MessageIn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DraftIn struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Space    string `json:"space"`
	Content  string `json:"content"`
	Status   string `json:"status"`
}

type User struct {
	ID       string
	Email    string
	TenantID uint64
}
func (User) TableName() string { return "users" }

type Organization struct {
	ID            string `gorm:"type:varchar(36);primaryKey"`
	Name          string
	Description   string
	OwnerID       string
	OwnerTenantID uint64
	InviteCode    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
func (Organization) TableName() string { return "organizations" }

type OrgExt struct {
	OrgID      string `gorm:"primaryKey"`
	AuthStatus string
	AuthType   string
	TenantID   uint64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
func (OrgExt) TableName() string { return "org_ext" }

type OrgMember struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	OrgID     string
	UserID    string
	Role      string
	Status    string
	JoinedAt  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
func (OrgMember) TableName() string { return "org_members" }

type KnowledgeSpace struct {
	ID          string `gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64
	OrgID       *string
	Name        string
	Description string
	Visibility  string
	Icon        string
	CreatorID   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
func (KnowledgeSpace) TableName() string { return "knowledge_spaces" }

type SpaceMember struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	SpaceID   string
	UserID    string
	Role      string
	CreatedAt time.Time
}
func (SpaceMember) TableName() string { return "space_members" }

type Document struct {
	ID              string `gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64
	SpaceID         string
	UploaderID      string
	Title           string
	FileName        string
	FileType        string
	FileSize        int64
	FilePath        string
	ContentHash     string
	ParseStatus     string
	ChunkCount      int
	EmbeddingStatus string
	Version         int
	Tags            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
func (Document) TableName() string { return "documents" }

type DocumentChunk struct {
	ID         string `gorm:"type:varchar(36);primaryKey"`
	DocumentID string
	TenantID   uint64
	ChunkIndex int
	Content    string
	TokenCount int
	Metadata   string
	CreatedAt  time.Time
}
func (DocumentChunk) TableName() string { return "document_chunks" }

type QASession struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64
	UserID    string
	SpaceID   string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
func (QASession) TableName() string { return "qa_sessions" }

type QAMessage struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	SessionID string
	TenantID  uint64
	Role      string
	Content   string
	Sources   string
	CreatedAt time.Time
}
func (QAMessage) TableName() string { return "qa_messages" }

type WritingDraft struct {
	ID        string `gorm:"type:varchar(36);primaryKey"`
	TenantID  uint64
	UserID    string
	Category  string
	Title     string
	Content   string
	Status    string
	SpaceID   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
func (WritingDraft) TableName() string { return "writing_drafts" }

type TokenUsage struct {
	ID           string `gorm:"type:varchar(36);primaryKey"`
	UserID       string
	TenantID     uint64
	Model        string
	InputTokens  int
	OutputTokens int
	Action       string
	CreatedAt    time.Time
}
func (TokenUsage) TableName() string { return "token_usage" }

func env(key, def string) string {
	if v := os.Getenv(key); v != "" { return v }
	return def
}

func loadConfig(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil { log.Fatal(err) }
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil { log.Fatal(err) }
	if cfg.TenantID == 0 { cfg.TenantID = 1 }
	if strings.TrimSpace(cfg.OwnerEmail) == "" { log.Fatal("owner_email is required") }
	if strings.TrimSpace(cfg.Organization.Name) == "" { log.Fatal("organization.name is required") }
	if cfg.Organization.AuthStatus == "" { cfg.Organization.AuthStatus = "trial" }
	if cfg.TokenUsageDays <= 0 { cfg.TokenUsageDays = 7 }
	return cfg
}

func contentHash(values ...string) string {
	h := sha256.New()
	for _, v := range values { h.Write([]byte(v)); h.Write([]byte("\n")) }
	return hex.EncodeToString(h.Sum(nil))
}

func tokenCount(s string) int {
	return len([]rune(s)) / 2
}

func main() {
	configPath := flag.String("config", "", "JSON config file")
	dryRun := flag.Bool("dry-run", false, "Validate only; do not write database")
	flag.Parse()
	if *configPath == "" { log.Fatal("missing --config") }
	cfg := loadConfig(*configPath)

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		env("SMART_DB_HOST", "localhost"), env("SMART_DB_PORT", "5432"), env("SMART_DB_USER", "postgres"), env("SMART_DB_PASSWORD", ""), env("SMART_DB_NAME", "WeKnora"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil { log.Fatalf("connect database: %v", err) }

	var owner User
	if err := db.Where("email = ?", cfg.OwnerEmail).First(&owner).Error; err != nil {
		log.Fatalf("owner_email not found; create account first: %s", cfg.OwnerEmail)
	}
	if cfg.TenantID == 0 { cfg.TenantID = owner.TenantID }

	fmt.Printf("Loaded seed config. tenant_id=%d owner=%s dry_run=%v\n", cfg.TenantID, cfg.OwnerEmail, *dryRun)
	if *dryRun {
		fmt.Printf("DRY-RUN organization=%s spaces=%d qa_sessions=%d drafts=%d\n", cfg.Organization.Name, len(cfg.Spaces), len(cfg.QASessions), len(cfg.WritingDrafts))
		return
	}

	now := time.Now()
	var org Organization
	err = db.Where("name = ? AND owner_id = ?", cfg.Organization.Name, owner.ID).First(&org).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		org = Organization{ID: uuid.New().String(), Name: cfg.Organization.Name, Description: cfg.Organization.Description, OwnerID: owner.ID, OwnerTenantID: cfg.TenantID, InviteCode: "TEST-" + strings.ToUpper(uuid.New().String()[:8]), CreatedAt: now, UpdatedAt: now}
		if err := db.Create(&org).Error; err != nil { log.Fatalf("create org: %v", err) }
		fmt.Printf("CREATED org: %s\n", org.Name)
	} else if err != nil { log.Fatal(err) } else {
		fmt.Printf("EXISTS org: %s\n", org.Name)
	}
	db.Where("org_id = ?", org.ID).FirstOrCreate(&OrgExt{OrgID: org.ID}, OrgExt{OrgID: org.ID, TenantID: cfg.TenantID, AuthStatus: cfg.Organization.AuthStatus, AuthType: cfg.Organization.AuthType, CreatedAt: now, UpdatedAt: now})
	db.Where("org_id = ? AND user_id = ?", org.ID, owner.ID).FirstOrCreate(&OrgMember{ID: uuid.New().String(), OrgID: org.ID, UserID: owner.ID, Role: "owner", Status: "active", JoinedAt: now, CreatedAt: now, UpdatedAt: now})

	spaceIDs := map[string]string{}
	for _, s := range cfg.Spaces {
		if s.Visibility == "" { s.Visibility = "team" }
		creatorID := owner.ID
		orgID := org.ID
		var space KnowledgeSpace
		err := db.Where("tenant_id = ? AND name = ?", cfg.TenantID, s.Name).First(&space).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			space = KnowledgeSpace{ID: uuid.New().String(), TenantID: cfg.TenantID, OrgID: &orgID, Name: s.Name, Description: s.Description, Visibility: s.Visibility, Icon: s.Icon, CreatorID: &creatorID, CreatedAt: now, UpdatedAt: now}
			if err := db.Create(&space).Error; err != nil { log.Fatalf("create space %s: %v", s.Name, err) }
			fmt.Printf("CREATED space: %s\n", s.Name)
		} else if err != nil { log.Fatal(err) } else { fmt.Printf("EXISTS space: %s\n", s.Name) }
		spaceIDs[s.Name] = space.ID
		db.Where("space_id = ? AND user_id = ?", space.ID, owner.ID).FirstOrCreate(&SpaceMember{ID: uuid.New().String(), SpaceID: space.ID, UserID: owner.ID, Role: "editor", CreatedAt: now})
		for _, m := range s.Members {
			var u User
			if err := db.Where("email = ?", m.Email).First(&u).Error; err != nil { log.Printf("warning: member email not found, skipped: %s", m.Email); continue }
			role := m.Role; if role == "" { role = "viewer" }
			db.Where("space_id = ? AND user_id = ?", space.ID, u.ID).FirstOrCreate(&SpaceMember{ID: uuid.New().String(), SpaceID: space.ID, UserID: u.ID, Role: role, CreatedAt: now})
		}
		for _, d := range s.Documents {
			joined := strings.Join(d.Chunks, "\n")
			hash := contentHash(space.ID, d.Title, joined)
			var doc Document
			err := db.Where("space_id = ? AND content_hash = ?", space.ID, hash).First(&doc).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				doc = Document{ID: uuid.New().String(), TenantID: cfg.TenantID, SpaceID: space.ID, UploaderID: owner.ID, Title: d.Title, FileName: d.FileName, FileType: d.FileType, FileSize: int64(len(joined)), FilePath: "seed://" + d.FileName, ContentHash: hash, ParseStatus: "completed", ChunkCount: len(d.Chunks), EmbeddingStatus: "pending", Version: 1, Tags: d.Tags, CreatedAt: now, UpdatedAt: now}
				if err := db.Create(&doc).Error; err != nil { log.Fatalf("create document %s: %v", d.Title, err) }
				for idx, chunk := range d.Chunks {
					metadata := fmt.Sprintf(`{"source":"seed","title":%q}`, d.Title)
					c := DocumentChunk{ID: uuid.New().String(), DocumentID: doc.ID, TenantID: cfg.TenantID, ChunkIndex: idx, Content: chunk, TokenCount: tokenCount(chunk), Metadata: metadata, CreatedAt: now}
					if err := db.Create(&c).Error; err != nil { log.Fatalf("create chunk: %v", err) }
				}
				fmt.Printf("CREATED document: %s chunks=%d\n", d.Title, len(d.Chunks))
			} else if err != nil { log.Fatal(err) } else { fmt.Printf("EXISTS document: %s\n", d.Title) }
		}
	}

	for _, q := range cfg.QASessions {
		spaceID := spaceIDs[q.Space]
		sess := QASession{ID: uuid.New().String(), TenantID: cfg.TenantID, UserID: owner.ID, SpaceID: spaceID, Title: q.Title, CreatedAt: now, UpdatedAt: now}
		if err := db.Create(&sess).Error; err != nil { log.Fatalf("create qa session: %v", err) }
		for _, msg := range q.Messages {
			m := QAMessage{ID: uuid.New().String(), SessionID: sess.ID, TenantID: cfg.TenantID, Role: msg.Role, Content: msg.Content, Sources: "[]", CreatedAt: now}
			if err := db.Create(&m).Error; err != nil { log.Fatalf("create qa message: %v", err) }
		}
		fmt.Printf("CREATED qa_session: %s messages=%d\n", q.Title, len(q.Messages))
	}

	for _, d := range cfg.WritingDrafts {
		spaceID := spaceIDs[d.Space]
		status := d.Status; if status == "" { status = "draft" }
		draft := WritingDraft{ID: uuid.New().String(), TenantID: cfg.TenantID, UserID: owner.ID, Category: d.Category, Title: d.Title, Content: d.Content, Status: status, SpaceID: spaceID, CreatedAt: now, UpdatedAt: now}
		if err := db.Create(&draft).Error; err != nil { log.Fatalf("create draft: %v", err) }
		fmt.Printf("CREATED writing_draft: %s\n", d.Title)
	}

	for i := 0; i < cfg.TokenUsageDays; i++ {
		createdAt := now.AddDate(0, 0, -i)
		u := TokenUsage{ID: uuid.New().String(), UserID: owner.ID, TenantID: cfg.TenantID, Model: "seed-test-model", InputTokens: 900 + i*17, OutputTokens: 300 + i*11, Action: "seed_test", CreatedAt: createdAt}
		if err := db.Create(&u).Error; err != nil { log.Printf("warning: create token usage failed: %v", err) }
	}
	fmt.Printf("CREATED token_usage days=%d\n", cfg.TokenUsageDays)
}