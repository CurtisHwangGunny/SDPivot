package handler

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/mail"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	maxUserImportFileSize = 10 << 20
	maxUserImportRows     = 1000
)

var importPhonePattern = regexp.MustCompile(`^\+?[0-9][0-9 -]{5,18}[0-9]$`)

type userImportRow struct {
	row      int
	phone    string
	email    string
	password string
	nickname string
}

type userImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field,omitempty"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}

type userImportResult struct {
	Total    int               `json:"total"`
	Imported int               `json:"imported"`
	Failed   int               `json:"failed"`
	Errors   []userImportError `json:"errors"`
}

type createOpsUserRequest struct {
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

// CreateUser provisions one SDPivot user with the same workspace defaults as
// batch import.
func (h *SDPivotOpsAdminHandler) CreateUser(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}

	var req createOpsUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	row := userImportRow{
		row:      1,
		phone:    normalizeImportPhone(strings.TrimSpace(req.Phone)),
		email:    strings.ToLower(strings.TrimSpace(req.Email)),
		password: req.Password,
		nickname: strings.TrimSpace(req.Nickname),
	}
	if errs := validateUserImportRow(row); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": errs[0].Message, "field": errs[0].Field})
		return
	}
	if conflict := h.findUserImportConflict(row); conflict != nil {
		c.JSON(http.StatusConflict, gin.H{"error": conflict.Message, "field": conflict.Field})
		return
	}
	if err := h.createImportedUser(c.Request.Context(), row); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": userImportDatabaseError(err)})
		return
	}

	username := importedUsername(row)
	h.writeAuditLog(c, "create_user", "user", username, "created user: "+username)
	c.JSON(http.StatusCreated, gin.H{"message": "user created", "username": username})
}

// ImportUsers imports SDPivot users from a multipart CSV or XLSX upload.
// Valid rows are committed independently so one bad row does not discard the
// rest of the batch; row-level validation and database failures are returned.
func (h *SDPivotOpsAdminHandler) ImportUsers(c *gin.Context) {
	if denyIfNotOpsAdmin(c) {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUserImportFileSize)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a CSV or XLSX file is required"})
		return
	}
	defer file.Close()

	rows, err := parseUserImportFile(file, header)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := userImportResult{
		Total:  len(rows),
		Errors: make([]userImportError, 0),
	}
	validRows := h.validateUserImportRows(rows, &result)
	for _, row := range validRows {
		if err := h.createImportedUser(c.Request.Context(), row); err != nil {
			result.Errors = append(result.Errors, userImportError{
				Row:     row.row,
				Message: userImportDatabaseError(err),
			})
			continue
		}
		result.Imported++
	}
	result.Failed = result.Total - result.Imported

	h.writeAuditLog(c, "import_users", "user", "", fmt.Sprintf(
		"file: %s, total: %d, imported: %d, failed: %d",
		filepath.Base(header.Filename), result.Total, result.Imported, result.Failed,
	))
	c.JSON(http.StatusOK, result)
}

func parseUserImportFile(file multipart.File, header *multipart.FileHeader) ([]userImportRow, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	var records [][]string
	var err error

	switch ext {
	case ".csv":
		reader := csv.NewReader(file)
		reader.FieldsPerRecord = -1
		reader.TrimLeadingSpace = true
		records, err = reader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("invalid CSV file: %w", err)
		}
	case ".xlsx":
		workbook, openErr := excelize.OpenReader(file)
		if openErr != nil {
			return nil, fmt.Errorf("invalid XLSX file: %w", openErr)
		}
		defer workbook.Close()
		sheets := workbook.GetSheetList()
		if len(sheets) == 0 {
			return nil, errors.New("XLSX file contains no worksheets")
		}
		records, err = workbook.GetRows(sheets[0])
		if err != nil {
			return nil, fmt.Errorf("failed to read XLSX worksheet: %w", err)
		}
	default:
		return nil, errors.New("unsupported file type; upload a .csv or .xlsx file")
	}

	return importRowsFromRecords(records)
}

func importRowsFromRecords(records [][]string) ([]userImportRow, error) {
	if len(records) == 0 {
		return nil, errors.New("import file is empty")
	}

	headers := make(map[string]int, len(records[0]))
	for index, value := range records[0] {
		name := normalizeUserImportHeader(value)
		if name != "" {
			if _, exists := headers[name]; exists {
				return nil, fmt.Errorf("duplicate header: %s", name)
			}
			headers[name] = index
		}
	}
	if _, ok := headers["password"]; !ok {
		return nil, errors.New("missing required header: password")
	}
	if _, hasPhone := headers["phone"]; !hasPhone {
		if _, hasEmail := headers["email"]; !hasEmail {
			return nil, errors.New("missing required header: phone or email")
		}
	}

	rows := make([]userImportRow, 0, len(records)-1)
	for index, record := range records[1:] {
		if importRecordEmpty(record) {
			continue
		}
		rows = append(rows, userImportRow{
			row:      index + 2,
			phone:    importRecordValue(record, importHeaderIndex(headers, "phone")),
			email:    strings.ToLower(importRecordValue(record, importHeaderIndex(headers, "email"))),
			password: importRecordValue(record, importHeaderIndex(headers, "password")),
			nickname: importRecordValue(record, importHeaderIndex(headers, "nickname")),
		})
		if len(rows) > maxUserImportRows {
			return nil, fmt.Errorf("import exceeds the maximum of %d data rows", maxUserImportRows)
		}
	}
	if len(rows) == 0 {
		return nil, errors.New("import file contains no data rows")
	}
	return rows, nil
}

func normalizeUserImportHeader(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))
	value = strings.ToLower(strings.ReplaceAll(value, " ", "_"))
	switch value {
	case "phone", "mobile", "phone_number", "手机号", "手机号码":
		return "phone"
	case "email", "email_address", "邮箱", "电子邮箱":
		return "email"
	case "password", "initial_password", "密码", "初始密码":
		return "password"
	case "nickname", "name", "display_name", "昵称", "姓名":
		return "nickname"
	default:
		return ""
	}
}

func importRecordEmpty(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func importRecordValue(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func importHeaderIndex(headers map[string]int, name string) int {
	index, ok := headers[name]
	if !ok {
		return -1
	}
	return index
}

func (h *SDPivotOpsAdminHandler) validateUserImportRows(rows []userImportRow, result *userImportResult) []userImportRow {
	valid := make([]userImportRow, 0, len(rows))
	seenPhones := make(map[string]struct{}, len(rows))
	seenEmails := make(map[string]struct{}, len(rows))
	seenUsernames := make(map[string]struct{}, len(rows))

	for _, row := range rows {
		row.phone = normalizeImportPhone(row.phone)
		rowErrors := validateUserImportRow(row)
		username := importedUsername(row)
		if row.phone != "" {
			if _, exists := seenPhones[row.phone]; exists {
				rowErrors = append(rowErrors, userImportError{Row: row.row, Field: "phone", Value: row.phone, Message: "duplicate phone in import file"})
			}
		}
		email := importedEmail(row)
		if email != "" {
			if _, exists := seenEmails[email]; exists {
				rowErrors = append(rowErrors, userImportError{Row: row.row, Field: "email", Value: email, Message: "duplicate email in import file"})
			}
		}
		if username != "" {
			if _, exists := seenUsernames[username]; exists {
				rowErrors = append(rowErrors, userImportError{Row: row.row, Field: "username", Value: username, Message: "duplicate username in import file"})
			}
		}

		if len(rowErrors) == 0 {
			if conflict := h.findUserImportConflict(row); conflict != nil {
				rowErrors = append(rowErrors, *conflict)
			}
		}
		if len(rowErrors) > 0 {
			result.Errors = append(result.Errors, rowErrors...)
		} else {
			valid = append(valid, row)
		}

		if row.phone != "" {
			seenPhones[row.phone] = struct{}{}
		}
		if email != "" {
			seenEmails[email] = struct{}{}
		}
		if username != "" {
			seenUsernames[username] = struct{}{}
		}
	}
	return valid
}

func normalizeImportPhone(value string) string {
	prefix := ""
	if strings.HasPrefix(value, "+") {
		prefix = "+"
	}
	value = strings.NewReplacer(" ", "", "-", "").Replace(value)
	return prefix + strings.TrimPrefix(value, "+")
}

func validateUserImportRow(row userImportRow) []userImportError {
	errs := make([]userImportError, 0, 4)
	if row.phone == "" && row.email == "" {
		errs = append(errs, userImportError{Row: row.row, Field: "phone_or_email", Message: "phone or email is required"})
	}
	if row.phone != "" && !importPhonePattern.MatchString(row.phone) {
		errs = append(errs, userImportError{Row: row.row, Field: "phone", Value: row.phone, Message: "invalid phone number"})
	}
	if row.email != "" && !validImportEmail(row.email) {
		errs = append(errs, userImportError{Row: row.row, Field: "email", Value: row.email, Message: "invalid email address"})
	}
	if len(row.password) < 8 {
		errs = append(errs, userImportError{Row: row.row, Field: "password", Message: "password must be at least 8 characters"})
	}
	if len(row.nickname) > 100 {
		errs = append(errs, userImportError{Row: row.row, Field: "nickname", Message: "nickname must not exceed 100 characters"})
	}
	return errs
}

func validImportEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && strings.EqualFold(address.Address, value) && strings.Contains(value, "@")
}

func importedUsername(row userImportRow) string {
	if row.phone != "" {
		return row.phone
	}
	return row.email
}

func importedEmail(row userImportRow) string {
	if row.email != "" {
		return row.email
	}
	if row.phone != "" {
		return row.phone + "@sdpivot.local"
	}
	return ""
}

func (h *SDPivotOpsAdminHandler) findUserImportConflict(row userImportRow) *userImportError {
	username := importedUsername(row)
	var count int64
	if err := h.db.Unscoped().Model(&types.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return &userImportError{Row: row.row, Message: "failed to validate username"}
	}
	if count > 0 {
		return &userImportError{Row: row.row, Field: "username", Value: username, Message: "username already exists"}
	}
	email := importedEmail(row)
	if email != "" {
		if err := h.db.Unscoped().Model(&types.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
			return &userImportError{Row: row.row, Message: "failed to validate email"}
		}
		if count > 0 {
			return &userImportError{Row: row.row, Field: "email", Value: email, Message: "email already exists"}
		}
	}
	if row.phone != "" {
		if err := h.db.Unscoped().Model(&types.SDPivotUserProfile{}).Where("phone = ?", row.phone).Count(&count).Error; err != nil {
			return &userImportError{Row: row.row, Message: "failed to validate phone"}
		}
		if count > 0 {
			return &userImportError{Row: row.row, Field: "phone", Value: row.phone, Message: "phone already exists"}
		}
	}
	return nil
}

func (h *SDPivotOpsAdminHandler) createImportedUser(ctx context.Context, row userImportRow) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(row.password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	username := importedUsername(row)
	email := importedEmail(row)
	nickname := row.nickname
	if nickname == "" {
		nickname = username
	}
	user := types.User{
		ID:                 uuid.New().String(),
		Username:           username,
		Email:              email,
		PasswordHash:       string(hash),
		IsActive:           true,
		MustChangePassword: true,
		TrialStartedAt:     &now,
		TrialPhase:         "30day",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	tenant := types.Tenant{
		Name:        username + " 的工作区",
		Description: "SDPivot 默认工作区",
		APIKey:      uuid.New().String(),
		Status:      "active",
		Business:    "sdpivot",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	trialExpiresAt := now.Add(30 * 24 * time.Hour)
	org := types.Organization{
		ID:         uuid.New().String(),
		Name:       "默认组织",
		OwnerID:    user.ID,
		InviteCode: generateInviteCode(),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	var phone *string
	if row.phone != "" {
		phone = &row.phone
	}
	profile := types.SDPivotUserProfile{
		UserID:    user.ID,
		Phone:     phone,
		Nickname:  nickname,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tenant).Error; err != nil {
			return err
		}
		user.TenantID = tenant.ID
		org.OwnerTenantID = tenant.ID
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&types.TenantMember{
			UserID: user.ID, TenantID: tenant.ID, Role: types.TenantRoleOwner,
			Status: types.TenantMemberStatusActive, JoinedAt: now, CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&org).Error; err != nil {
			return err
		}
		if err := tx.Create(&types.OrgExt{
			OrgID: org.ID, TenantID: tenant.ID, AuthStatus: "trial",
			AuthExpiresAt: &trialExpiresAt, CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&types.SDPivotOrgMember{
			OrgID: org.ID, UserID: user.ID, Role: "owner", Status: "active", JoinedAt: now,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&profile).Error
	})
}

func userImportDatabaseError(err error) string {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "email") && strings.Contains(message, "unique"):
		return "email already exists"
	case strings.Contains(message, "username") && strings.Contains(message, "unique"):
		return "username already exists"
	case strings.Contains(message, "phone") && strings.Contains(message, "unique"):
		return "phone already exists"
	default:
		return "failed to create user"
	}
}
