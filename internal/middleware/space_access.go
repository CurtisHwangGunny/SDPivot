package middleware

import (
	"errors"
	"net/http"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SpaceAccessLevel int

const (
	SpaceAccessView SpaceAccessLevel = iota
	SpaceAccessEdit
	SpaceAccessOwner
)

const spaceAccessDeniedMessage = "您不是该空间的成员，无法执行此操作。请联系空间管理员将您添加为成员。"

// AuthorizeSpace enforces OP visibility + member ACL. It deliberately scopes
// the lookup to the active official tenant before checking membership.
func AuthorizeSpace(c *gin.Context, db *gorm.DB, spaceID string, level SpaceAccessLevel) (*types.KnowledgeSpace, bool) {
	var space types.KnowledgeSpace
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	userID := c.GetString(types.UserIDContextKey.String())
	err := db.Where("id = ? AND tenant_id = ?", spaceID, tenantID).First(&space).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "space not found"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check space access"})
		return nil, false
	}

	if level == SpaceAccessView && space.Visibility != "private" {
		return &space, true
	}
	var member types.SpaceMember
	err = db.Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusForbidden, gin.H{"error": spaceAccessDeniedMessage, "code": "space_access_denied"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check space access"})
		return nil, false
	}
	allowed := level == SpaceAccessView ||
		(level == SpaceAccessEdit && (member.Role == "owner" || member.Role == "editor")) ||
		(level == SpaceAccessOwner && member.Role == "owner")
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": spaceAccessDeniedMessage, "code": "space_access_denied"})
		return nil, false
	}
	return &space, true
}

// VisibleSpaceIDsQuery is the canonical list filter for private spaces.
func VisibleSpaceIDsQuery(db *gorm.DB, tenantID uint64, userID string) *gorm.DB {
	return db.Model(&types.KnowledgeSpace{}).
		Select("knowledge_spaces.id").
		Where("knowledge_spaces.tenant_id = ?", tenantID).
		Where("knowledge_spaces.visibility <> 'private' OR EXISTS (?)", db.Model(&types.SpaceMember{}).
			Select("1").Where("space_members.space_id = knowledge_spaces.id AND space_members.user_id = ?", userID))
}
