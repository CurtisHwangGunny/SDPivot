package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

type spaceMemberCandidateUser struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Username       string           `json:"username"`
	Account        string           `json:"account"`
	Email          string           `json:"email"`
	DepartmentID   *string          `json:"department_id"`
	DepartmentName string           `json:"department_name"`
	AccessRole     types.AccessRole `json:"access_role"`
	IsMember       bool             `json:"is_member"`
}

type spaceMemberCandidateDepartment struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	ParentID            string `json:"parent_id"`
	SortOrder           int    `json:"sort_order"`
	MemberCount         int64  `json:"member_count"`
	ExistingMemberCount int64  `json:"existing_member_count"`
	IsFullyAdded        bool   `json:"is_fully_added"`
}

type addSpaceMembersRequest struct {
	UserIDs       []string `json:"user_ids"`
	DepartmentIDs []string `json:"department_ids"`
}

func (h *SDPivotSpaceHandler) ListSpaceMemberCandidates(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	spaceID := strings.TrimSpace(c.Param("id"))
	scopeIDs, ok := h.authorizeMemberSelection(c, db, spaceID)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 100
	}

	usersQuery := db.Table("users u").
		Joins("LEFT JOIN departments d ON d.id = u.department_id AND d.tenant_id = u.tenant_id AND d.deleted_at IS NULL").
		Joins("LEFT JOIN smartknora_user_profiles p ON p.user_id = u.id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN space_members sm ON sm.space_id = ? AND sm.user_id = u.id", spaceID).
		Where("u.tenant_id = ? AND u.deleted_at IS NULL AND u.is_active = ?", tenantID, true)
	if scopeIDs != nil {
		usersQuery = usersQuery.Where("u.department_id IN ?", scopeIDs)
	}
	if departmentID := strings.TrimSpace(c.Query("department_id")); departmentID != "" {
		usersQuery = usersQuery.Where("u.department_id = ?", departmentID)
	}
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := "%" + escapeILike(keyword) + "%"
		usersQuery = usersQuery.Where("u.username ILIKE ? OR u.email ILIKE ? OR COALESCE(p.nickname, '') ILIKE ?", like, like, like)
	}

	var total int64
	if err := usersQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count candidate users"})
		return
	}
	users := make([]spaceMemberCandidateUser, 0)
	if err := usersQuery.Select(`u.id, COALESCE(NULLIF(p.nickname, ''), u.username) AS name,
		u.username, u.username AS account, u.email, u.department_id,
		COALESCE(d.name, '') AS department_name, u.access_role,
		CASE WHEN sm.user_id IS NULL THEN false ELSE true END AS is_member`).
		Order("COALESCE(d.sort_order, 0) ASC, COALESCE(d.name, '') ASC, name ASC, u.username ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Scan(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list candidate users"})
		return
	}

	departments, err := h.listCandidateDepartments(db, tenantID, spaceID, scopeIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list candidate departments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"users": users, "departments": departments, "total": total, "page": page, "page_size": pageSize,
	})
}

func (h *SDPivotSpaceHandler) listCandidateDepartments(db *gorm.DB, tenantID uint64, spaceID string, scopeIDs []string) ([]spaceMemberCandidateDepartment, error) {
	query := db.Table("departments d").
		Select(`d.id, d.name, d.parent_id, d.sort_order,
			COUNT(u.id) AS member_count,
			COUNT(sm.user_id) AS existing_member_count`).
		Joins("LEFT JOIN users u ON u.department_id = d.id AND u.tenant_id = d.tenant_id AND u.deleted_at IS NULL AND u.is_active = ?", true).
		Joins("LEFT JOIN space_members sm ON sm.space_id = ? AND sm.user_id = u.id", spaceID).
		Where("d.tenant_id = ? AND d.deleted_at IS NULL", tenantID)
	if scopeIDs != nil {
		query = query.Where("d.id IN ?", scopeIDs)
	}
	rows := make([]spaceMemberCandidateDepartment, 0)
	if err := query.Group("d.id, d.name, d.parent_id, d.sort_order").
		Order("d.sort_order ASC, d.name ASC, d.id ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for index := range rows {
		rows[index].IsFullyAdded = rows[index].MemberCount > 0 && rows[index].ExistingMemberCount == rows[index].MemberCount
	}
	return rows, nil
}

func (h *SDPivotSpaceHandler) AddSpaceMembers(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	spaceID := strings.TrimSpace(c.Param("id"))
	scopeIDs, ok := h.authorizeMemberSelection(c, db, spaceID)
	if !ok {
		return
	}

	var req addSpaceMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid member selection"})
		return
	}
	req.UserIDs = uniqueNonEmptyStrings(req.UserIDs)
	req.DepartmentIDs = uniqueNonEmptyStrings(req.DepartmentIDs)
	if len(req.UserIDs) == 0 && len(req.DepartmentIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_ids or department_ids is required"})
		return
	}

	if ok := validateSelectedDepartments(c, db, tenantID, req.DepartmentIDs, scopeIDs); !ok {
		return
	}

	selectedIDs := append([]string(nil), req.UserIDs...)
	if len(req.DepartmentIDs) > 0 {
		var departmentUserIDs []string
		if err := db.Model(&types.User{}).
			Where("tenant_id = ? AND department_id IN ? AND is_active = ?", tenantID, req.DepartmentIDs, true).
			Order("username ASC").Pluck("id", &departmentUserIDs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to expand department members"})
			return
		}
		selectedIDs = append(selectedIDs, departmentUserIDs...)
	}
	selectedIDs = uniqueNonEmptyStrings(selectedIDs)
	if len(selectedIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"added_ids": []string{}, "skipped_ids": []string{}, "invalid_ids": []string{}})
		return
	}

	validIDs, invalidIDs, err := validateSelectedUsers(db, tenantID, selectedIDs, scopeIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate selected users"})
		return
	}
	if len(invalidIDs) > 0 && scopeIDs != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "one or more users are outside the permitted scope", "user_ids": invalidIDs})
		return
	}

	var existingIDs []string
	if len(validIDs) > 0 {
		if err := db.Model(&types.SpaceMember{}).Where("space_id = ? AND user_id IN ?", spaceID, validIDs).Pluck("user_id", &existingIDs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check existing space members"})
			return
		}
	}
	existing := stringSet(existingIDs)
	addedIDs := make([]string, 0, len(validIDs))
	skippedIDs := make([]string, 0, len(existingIDs))
	members := make([]types.SpaceMember, 0, len(validIDs))
	now := time.Now()
	for _, userID := range validIDs {
		if _, found := existing[userID]; found {
			skippedIDs = append(skippedIDs, userID)
			continue
		}
		addedIDs = append(addedIDs, userID)
		members = append(members, types.SpaceMember{ID: uuid.New().String(), SpaceID: spaceID, UserID: userID, Role: "viewer", CreatedAt: now})
	}
	if len(members) > 0 {
		if err := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "space_id"}, {Name: "user_id"}}, DoNothing: true}).Create(&members).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add space members"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"added_ids": addedIDs, "skipped_ids": skippedIDs, "invalid_ids": invalidIDs,
		"added_count": len(addedIDs), "skipped_count": len(skippedIDs),
	})
}

// authorizeMemberSelection returns nil scope for tenant-wide access and a
// concrete department scope for department administrators.
func (h *SDPivotSpaceHandler) authorizeMemberSelection(c *gin.Context, db *gorm.DB, spaceID string) ([]string, bool) {
	var space types.KnowledgeSpace
	if err := db.Where("id = ? AND tenant_id = ?", spaceID, middleware.GetTenantID(c)).First(&space).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "space not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load space"})
		}
		return nil, false
	}

	switch types.NormalizeAccessRole(middleware.GetRole(c)) {
	case types.AccessRoleSuperAdmin:
		return nil, true
	case types.AccessRoleDepartmentAdmin:
		ids, err := departmentScopeIDs(db, middleware.GetTenantID(c), middleware.GetDepartmentID(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load department scope"})
			return nil, false
		}
		if len(ids) == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "department scope is not available"})
			return nil, false
		}
		return ids, true
	}

	var count int64
	if err := db.Model(&types.SpaceMember{}).
		Where("space_id = ? AND user_id = ? AND role = ?", spaceID, middleware.GetUserID(c), "owner").
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check space owner"})
		return nil, false
	}
	if count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "space member management denied"})
		return nil, false
	}
	return nil, true
}

func validateSelectedDepartments(c *gin.Context, db *gorm.DB, tenantID uint64, selectedIDs, scopeIDs []string) bool {
	if len(selectedIDs) == 0 {
		return true
	}
	var existingIDs []string
	if err := db.Model(&types.Department{}).Where("tenant_id = ? AND id IN ?", tenantID, selectedIDs).Pluck("id", &existingIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate selected departments"})
		return false
	}
	existing := stringSet(existingIDs)
	for _, id := range selectedIDs {
		if _, found := existing[id]; !found {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id", "department_id": id})
			return false
		}
	}
	if scopeIDs == nil {
		return true
	}
	scope := stringSet(scopeIDs)
	for _, id := range selectedIDs {
		if _, found := scope[id]; !found {
			c.JSON(http.StatusForbidden, gin.H{"error": "department is outside the permitted scope", "department_id": id})
			return false
		}
	}
	return true
}

func validateSelectedUsers(db *gorm.DB, tenantID uint64, selectedIDs, scopeIDs []string) ([]string, []string, error) {
	query := db.Model(&types.User{}).Where("tenant_id = ? AND id IN ? AND is_active = ?", tenantID, selectedIDs, true)
	if scopeIDs != nil {
		query = query.Where("department_id IN ?", scopeIDs)
	}
	var foundIDs []string
	if err := query.Pluck("id", &foundIDs).Error; err != nil {
		return nil, nil, err
	}
	found := stringSet(foundIDs)
	valid := make([]string, 0, len(foundIDs))
	invalid := make([]string, 0)
	for _, id := range selectedIDs {
		if _, ok := found[id]; ok {
			valid = append(valid, id)
		} else {
			invalid = append(invalid, id)
		}
	}
	return valid, invalid, nil
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, found := seen[value]; found {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
