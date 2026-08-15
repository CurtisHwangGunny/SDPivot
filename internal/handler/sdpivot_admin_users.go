package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

const maxAdminUserBatchRows = 1000

type SDPivotAdminUsersHandler struct {
	db *gorm.DB
}

type adminUserBatchRow struct {
	Username     string           `json:"username"`
	Name         string           `json:"name"`
	Nickname     string           `json:"nickname"`
	Email        string           `json:"email"`
	Phone        string           `json:"phone"`
	Password     string           `json:"password"`
	DepartmentID string           `json:"department_id"`
	AccessRole   types.AccessRole `json:"access_role"`
}

type adminUserBatchError struct {
	Row     int    `json:"row"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type adminUserBatchSuccess struct {
	Row             int    `json:"row"`
	ID              string `json:"id"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	InitialPassword string `json:"initial_password,omitempty"`
}

func NewSDPivotAdminUsersHandler(db *gorm.DB) *SDPivotAdminUsersHandler {
	return &SDPivotAdminUsersHandler{db: db}
}

func (h *SDPivotAdminUsersHandler) RegisterRoutes(rg *gin.RouterGroup) {
	adminRead := rg.Group("/admin", middleware.RequirePermission(middleware.PermissionKnowledgeRead))
	adminRead.GET("/roles", h.ListRoles)

	adminManage := rg.Group("/admin", middleware.RequirePermission(middleware.PermissionDepartmentManage))
	adminManage.GET("/users", h.ListUsers)
	adminManage.POST("/users", h.CreateUser)
	adminManage.POST("/users/batch", h.BatchCreateUsers)

	adminRole := rg.Group("/admin", middleware.RequirePermission(middleware.PermissionUserRoleAssign))
	adminRole.PUT("/users/:id/role", h.UpdateUserRole)
}

func (h *SDPivotAdminUsersHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	tenantID := middleware.GetTenantID(c)
	tenantDB := middleware.TenantDB(c, h.db)
	db := tenantDB.Table("users u").
		Joins("LEFT JOIN departments d ON d.id = u.department_id AND d.tenant_id = u.tenant_id AND d.deleted_at IS NULL").
		Joins("LEFT JOIN smartknora_user_profiles p ON p.user_id = u.id").
		Where("u.tenant_id = ? AND u.deleted_at IS NULL", tenantID)
	if types.NormalizeAccessRole(middleware.GetRole(c)) == types.AccessRoleDepartmentAdmin {
		ids, err := departmentScopeIDs(tenantDB, tenantID, middleware.GetDepartmentID(c))
		if err != nil || len(ids) == 0 {
			db = db.Where("1 = 0")
		} else {
			db = db.Where("u.department_id IN ?", ids)
		}
	}
	if departmentID := strings.TrimSpace(c.Query("department_id")); departmentID != "" {
		db = db.Where("u.department_id = ?", departmentID)
	}
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("keyword"))
	}
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("search"))
	}
	if keyword != "" {
		like := "%" + escapeILike(keyword) + "%"
		db = db.Where("u.username ILIKE ? OR u.email ILIKE ? OR COALESCE(p.nickname, '') ILIKE ? OR COALESCE(p.phone, '') ILIKE ?", like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count users"})
		return
	}
	type adminUserListItem struct {
		ID             string           `json:"id"`
		Name           string           `json:"name"`
		Username       string           `json:"username"`
		Account        string           `json:"account"`
		Email          string           `json:"email"`
		Phone          string           `json:"phone"`
		DepartmentID   *string          `json:"department_id"`
		DepartmentName string           `json:"department_name"`
		AccessRole     types.AccessRole `json:"access_role"`
		Role           types.AccessRole `json:"role"`
		IsActive       bool             `json:"is_active"`
		Status         string           `json:"status"`
		CreatedAt      time.Time        `json:"created_at"`
	}
	users := make([]adminUserListItem, 0)
	if err := db.Select(`u.id, COALESCE(NULLIF(p.nickname, ''), u.username) AS name, u.username, u.username AS account, u.email,
		COALESCE(p.phone, '') AS phone, u.department_id, COALESCE(d.name, '') AS department_name,
		u.access_role, u.is_active, CASE WHEN u.is_active THEN 'active' ELSE 'disabled' END AS status, u.created_at`).
		Order("u.created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	for index := range users {
		users[index].Role = users[index].AccessRole
	}
	c.JSON(http.StatusOK, gin.H{"users": users, "total": total, "page": page, "page_size": pageSize})
}

func (h *SDPivotAdminUsersHandler) ListRoles(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"roles": []gin.H{
		{"code": types.AccessRoleSuperAdmin, "name": "超级管理员"},
		{"code": types.AccessRoleDepartmentAdmin, "name": "部门管理员"},
		{"code": types.AccessRoleKnowledgeEditor, "name": "知识编辑者"},
		{"code": types.AccessRoleKnowledgeViewer, "name": "知识查阅者"},
	}})
}

func (h *SDPivotAdminUsersHandler) BatchCreateUsers(c *gin.Context) {
	rows, err := decodeAdminUserBatch(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(rows) == 0 || len(rows) > maxAdminUserBatchRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("batch must contain 1 to %d users", maxAdminUserBatchRows)})
		return
	}

	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	result := struct {
		Total    int                     `json:"total"`
		Imported int                     `json:"imported"`
		Failed   int                     `json:"failed"`
		Created  []adminUserBatchSuccess `json:"created"`
		Errors   []adminUserBatchError   `json:"errors"`
	}{Total: len(rows), Created: make([]adminUserBatchSuccess, 0), Errors: make([]adminUserBatchError, 0)}

	for index, row := range rows {
		generatedPassword := ""
		if strings.TrimSpace(row.Password) == "" {
			generatedPassword, err = generateInitialPassword()
			if err != nil {
				result.Errors = append(result.Errors, adminUserBatchError{Row: index + 1, Field: "password", Message: "failed to generate initial password"})
				continue
			}
			row.Password = generatedPassword
		}
		user, batchErr := h.createTenantUser(c, db, tenantID, index+1, row)
		if batchErr != nil {
			result.Errors = append(result.Errors, *batchErr)
			continue
		}
		result.Imported++
		result.Created = append(result.Created, adminUserBatchSuccess{Row: index + 1, ID: user.ID, Username: user.Username, Email: user.Email, InitialPassword: generatedPassword})
	}
	result.Failed = result.Total - result.Imported
	writeSDPivotAuditLog(db, c, auditActionUserImported, auditModuleUser, "user", "", map[string]interface{}{
		"total": result.Total, "imported": result.Imported, "failed": result.Failed,
	})
	c.JSON(http.StatusCreated, result)
}

func (h *SDPivotAdminUsersHandler) CreateUser(c *gin.Context) {
	var req adminUserBatchRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user data"})
		return
	}
	password, err := generateInitialPassword()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate initial password"})
		return
	}
	req.Password = password
	db := middleware.TenantDB(c, h.db)
	user, createErr := h.createTenantUser(c, db, middleware.GetTenantID(c), 1, req)
	if createErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": createErr.Message, "field": createErr.Field})
		return
	}
	writeSDPivotAuditLog(db, c, auditActionUserCreated, auditModuleUser, "user", user.ID, map[string]interface{}{
		"username": user.Username, "department_id": user.DepartmentID,
	})
	c.JSON(http.StatusCreated, gin.H{"user": user, "initial_password": password})
}

func generateInitialPassword() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	for index := range buffer {
		buffer[index] = alphabet[int(buffer[index])%len(alphabet)]
	}
	return string(buffer), nil
}

func decodeAdminUserBatch(body io.Reader) ([]adminUserBatchRow, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxUserImportFileSize+1))
	if err != nil || len(data) == 0 {
		return nil, errors.New("request body is required")
	}
	if len(data) > maxUserImportFileSize {
		return nil, errors.New("request body exceeds 10 MB")
	}

	var rows []adminUserBatchRow
	if err := json.Unmarshal(data, &rows); err == nil {
		return rows, nil
	}
	var payload struct {
		CSVBase64 string              `json:"csv_base64"`
		Users     []adminUserBatchRow `json:"users"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, errors.New("body must be a JSON array, contain users, or contain csv_base64")
	}
	if len(payload.Users) > 0 {
		return payload.Users, nil
	}
	if strings.TrimSpace(payload.CSVBase64) == "" {
		return nil, errors.New("body must be a JSON array, contain users, or contain csv_base64")
	}
	csvData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload.CSVBase64))
	if err != nil {
		return nil, errors.New("csv_base64 is invalid")
	}
	return parseAdminUserCSV(csvData)
}

func parseAdminUserCSV(data []byte) ([]adminUserBatchRow, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		return nil, errors.New("CSV must contain a header and at least one data row")
	}
	headers := make(map[string]int, len(records[0]))
	for index, value := range records[0] {
		headers[normalizeAdminUserHeader(value)] = index
	}
	rows := make([]adminUserBatchRow, 0, len(records)-1)
	for _, record := range records[1:] {
		if importRecordEmpty(record) {
			continue
		}
		rows = append(rows, adminUserBatchRow{
			Username:     importRecordValue(record, adminUserHeaderIndex(headers, "username")),
			Name:         importRecordValue(record, adminUserHeaderIndex(headers, "name")),
			Nickname:     importRecordValue(record, adminUserHeaderIndex(headers, "nickname")),
			Email:        importRecordValue(record, adminUserHeaderIndex(headers, "email")),
			Phone:        importRecordValue(record, adminUserHeaderIndex(headers, "phone")),
			Password:     importRecordValue(record, adminUserHeaderIndex(headers, "password")),
			DepartmentID: importRecordValue(record, adminUserHeaderIndex(headers, "department_id")),
			AccessRole:   types.AccessRole(importRecordValue(record, adminUserHeaderIndex(headers, "access_role"))),
		})
	}
	return rows, nil
}

func adminUserHeaderIndex(headers map[string]int, name string) int {
	if index, ok := headers[name]; ok {
		return index
	}
	return -1
}

func normalizeAdminUserHeader(value string) string {
	switch strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))) {
	case "username", "用户名", "账号":
		return "username"
	case "name", "姓名":
		return "name"
	case "nickname", "昵称":
		return "nickname"
	case "email", "邮箱":
		return "email"
	case "phone", "mobile", "手机号":
		return "phone"
	case "password", "初始密码", "密码":
		return "password"
	case "department_id", "部门id", "部门_id":
		return "department_id"
	case "access_role", "role", "角色":
		return "access_role"
	default:
		return "_"
	}
}

func (h *SDPivotAdminUsersHandler) createTenantUser(c *gin.Context, db *gorm.DB, tenantID uint64, rowNumber int, row adminUserBatchRow) (*types.User, *adminUserBatchError) {
	row.Email = strings.ToLower(strings.TrimSpace(row.Email))
	row.Phone = normalizeImportPhone(strings.TrimSpace(row.Phone))
	row.Username = strings.TrimSpace(row.Username)
	row.DepartmentID = strings.TrimSpace(row.DepartmentID)
	if row.Username == "" {
		if row.Email != "" {
			row.Username = row.Email
		} else {
			row.Username = row.Phone
		}
	}
	if row.Username == "" {
		return nil, &adminUserBatchError{Row: rowNumber, Field: "username", Message: "username, email, or phone is required"}
	}
	if row.Email == "" {
		row.Email = row.Username + "@sdpivot.local"
	}
	if address, err := mail.ParseAddress(row.Email); err != nil || !strings.EqualFold(address.Address, row.Email) {
		return nil, &adminUserBatchError{Row: rowNumber, Field: "email", Message: "invalid email address"}
	}
	if row.Phone != "" && !importPhonePattern.MatchString(row.Phone) {
		return nil, &adminUserBatchError{Row: rowNumber, Field: "phone", Message: "invalid phone number"}
	}
	if len(row.Password) < 8 {
		return nil, &adminUserBatchError{Row: rowNumber, Field: "password", Message: "password must be at least 8 characters"}
	}
	if row.DepartmentID == "" {
		return nil, &adminUserBatchError{Row: rowNumber, Field: "department_id", Message: "department_id is required"}
	}
	if !h.canAssignDepartment(c, db, tenantID, row.DepartmentID) {
		return nil, &adminUserBatchError{Row: rowNumber, Field: "department_id", Message: "department is outside the permitted scope"}
	}
	if row.AccessRole == "" {
		row.AccessRole = types.AccessRoleKnowledgeViewer
	}
	if row.AccessRole != types.AccessRoleKnowledgeViewer && row.AccessRole != types.AccessRoleKnowledgeEditor {
		return nil, &adminUserBatchError{Row: rowNumber, Field: "access_role", Message: "invalid access role"}
	}

	var count int64
	if err := db.Unscoped().Model(&types.User{}).Where("username = ? OR email = ?", row.Username, row.Email).Count(&count).Error; err != nil || count > 0 {
		return nil, &adminUserBatchError{Row: rowNumber, Field: "username", Message: "username or email already exists"}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(row.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, &adminUserBatchError{Row: rowNumber, Message: "failed to secure initial password"}
	}
	nickname := strings.TrimSpace(row.Nickname)
	if nickname == "" {
		nickname = strings.TrimSpace(row.Name)
	}
	if nickname == "" {
		nickname = row.Username
	}
	now := time.Now()
	user := types.User{
		ID: uuid.New().String(), Username: row.Username, Email: row.Email, PasswordHash: string(hash), TenantID: tenantID,
		IsActive: true, AccessRole: row.AccessRole, DepartmentID: &row.DepartmentID, MustChangePassword: true,
		CreatedAt: now, UpdatedAt: now,
	}
	var phone *string
	if row.Phone != "" {
		phone = &row.Phone
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&types.TenantMember{UserID: user.ID, TenantID: tenantID, Role: types.TenantRoleViewer, Status: types.TenantMemberStatusActive, JoinedAt: now, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			return err
		}
		return tx.Create(&types.SDPivotUserProfile{UserID: user.ID, Phone: phone, Nickname: nickname, Status: "active", CreatedAt: now, UpdatedAt: now}).Error
	}); err != nil {
		return nil, &adminUserBatchError{Row: rowNumber, Message: userImportDatabaseError(err)}
	}
	return &user, nil
}

func (h *SDPivotAdminUsersHandler) canAssignDepartment(c *gin.Context, db *gorm.DB, tenantID uint64, departmentID string) bool {
	if types.NormalizeAccessRole(middleware.GetRole(c)) == types.AccessRoleSuperAdmin {
		var count int64
		return db.Model(&types.Department{}).Where("tenant_id = ? AND id = ?", tenantID, departmentID).Count(&count).Error == nil && count == 1
	}
	ids, err := departmentScopeIDs(db, tenantID, middleware.GetDepartmentID(c))
	if err != nil {
		return false
	}
	for _, id := range ids {
		if id == departmentID {
			return true
		}
	}
	return false
}

func (h *SDPivotAdminUsersHandler) UpdateUserRole(c *gin.Context) {
	db := middleware.TenantDB(c, h.db)
	tenantID := middleware.GetTenantID(c)
	var req struct {
		Role         types.AccessRole `json:"role" binding:"required"`
		DepartmentID *string          `json:"department_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !req.Role.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}
	var user types.User
	if err := db.Where("id = ? AND tenant_id = ?", c.Param("id"), tenantID).First(&user).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "user not found"})
		return
	}
	departmentID := user.DepartmentID
	if req.DepartmentID != nil {
		value := strings.TrimSpace(*req.DepartmentID)
		if value == "" || !h.canAssignDepartment(c, db, tenantID, value) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id"})
			return
		}
		departmentID = &value
	}
	if req.Role == types.AccessRoleDepartmentAdmin && departmentID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department_id is required for department_admin"})
		return
	}
	oldRole := user.AccessRole
	if err := db.Model(&types.User{}).Where("id = ? AND tenant_id = ?", user.ID, tenantID).Updates(map[string]interface{}{
		"access_role": req.Role, "department_id": departmentID, "is_ops_admin": req.Role == types.AccessRoleSuperAdmin, "updated_at": time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user role"})
		return
	}
	writeSDPivotAuditLog(db, c, auditActionUserRoleUpdated, auditModuleUser, "user", user.ID, map[string]interface{}{
		"old_role": oldRole, "new_role": req.Role, "department_id": departmentID,
	})
	c.JSON(http.StatusOK, gin.H{"message": "user role updated", "role": req.Role, "department_id": departmentID})
}
