package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

type spaceAccessLevel int

const (
	spaceAccessView spaceAccessLevel = iota
	spaceAccessEdit
	spaceAccessOwner
)

const spaceAccessDeniedMessage = "您不是该空间的成员，无法执行此操作。请联系空间管理员将您添加为成员。"

func respondSpaceAccessDenied(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"error": spaceAccessDeniedMessage,
		"code":  "space_access_denied",
	})
}

func authorizeSpace(c *gin.Context, db *gorm.DB, spaceID string, level spaceAccessLevel) (*types.KnowledgeSpace, bool) {
	var space types.KnowledgeSpace
	err := db.Where("id = ? AND tenant_id = ?", spaceID, middleware.GetTenantID(c)).First(&space).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "space not found"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check space access"})
		return nil, false
	}

	if level == spaceAccessView && space.Visibility != "private" {
		return &space, true
	}

	var member types.SpaceMember
	err = db.Where("space_id = ? AND user_id = ?", spaceID, middleware.GetUserID(c)).First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respondSpaceAccessDenied(c)
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check space access"})
		return nil, false
	}

	allowed := level == spaceAccessView ||
		(level == spaceAccessEdit && (member.Role == "owner" || member.Role == "editor")) ||
		(level == spaceAccessOwner && member.Role == "owner")
	if !allowed {
		respondSpaceAccessDenied(c)
		return nil, false
	}
	return &space, true
}

func visibleSpaceIDsQuery(db *gorm.DB, tenantID uint64, userID string) *gorm.DB {
	return db.Model(&types.KnowledgeSpace{}).
		Select("knowledge_spaces.id").
		Where("knowledge_spaces.tenant_id = ?", tenantID).
		Where("knowledge_spaces.visibility IN ('team', 'org') OR EXISTS (?)",
			db.Model(&types.SpaceMember{}).
				Select("1").
				Where("space_members.space_id = knowledge_spaces.id AND space_members.user_id = ?", userID))
}
