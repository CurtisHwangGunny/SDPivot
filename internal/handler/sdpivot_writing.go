package handler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
)

type SDPivotWritingHandler struct{ db *gorm.DB }

func writingUserID(c *gin.Context) string {
	id, _ := types.UserIDFromContext(c.Request.Context())
	return id
}
func writingTenantID(c *gin.Context) uint64 {
	id, _ := types.TenantIDFromContext(c.Request.Context())
	return id
}

func NewSDPivotWritingHandler(db *gorm.DB) *SDPivotWritingHandler {
	return &SDPivotWritingHandler{db: db}
}

func (h *SDPivotWritingHandler) RegisterRoutes(rg *gin.RouterGroup) {
	w := rg.Group("/writing")
	w.POST("/drafts", h.CreateDraft)
	w.GET("/drafts", h.ListDrafts)
	w.GET("/drafts/:id", h.GetDraft)
	w.PUT("/drafts/:id", h.UpdateDraft)
	w.DELETE("/drafts/:id", h.DeleteDraft)
	w.POST("/generate", h.GenerateContent)
	w.POST("/drafts/:id/export", h.ExportDraft)
}

func (h *SDPivotWritingHandler) CreateDraft(c *gin.Context) {
	var req struct {
		Title            string `json:"title"`
		Category         string `json:"category"`
		SpaceID          string `json:"space_id"`
		SourceType       string `json:"source_type"`
		WebSearchEnabled bool   `json:"web_search_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now()
	draft := types.WritingDraft{ID: uuid.NewString(), UserID: writingUserID(c), TenantID: writingTenantID(c), Title: req.Title, Category: req.Category, SpaceID: req.SpaceID, SourceType: req.SourceType, WebSearchEnabled: req.WebSearchEnabled, Status: "draft", CreatedAt: now, UpdatedAt: now}
	if err := middleware.TenantDB(c, h.db).Create(&draft).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create draft"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"draft": draft})
}

func (h *SDPivotWritingHandler) ListDrafts(c *gin.Context) {
	var drafts []types.WritingDraft
	err := middleware.TenantDB(c, h.db).Where("tenant_id = ? AND user_id = ?", writingTenantID(c), writingUserID(c)).Order("updated_at DESC").Find(&drafts).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list drafts"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"drafts": drafts})
}

func (h *SDPivotWritingHandler) GetDraft(c *gin.Context) {
	var draft types.WritingDraft
	err := middleware.TenantDB(c, h.db).Where("id = ? AND tenant_id = ? AND user_id = ?", c.Param("id"), writingTenantID(c), writingUserID(c)).First(&draft).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"draft": draft})
}

func (h *SDPivotWritingHandler) UpdateDraft(c *gin.Context) {
	var req struct{ Title, Content, Status *string }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	result := middleware.TenantDB(c, h.db).Model(&types.WritingDraft{}).Where("id = ? AND tenant_id = ? AND user_id = ?", c.Param("id"), writingTenantID(c), writingUserID(c)).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update draft"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "draft updated"})
}

func (h *SDPivotWritingHandler) DeleteDraft(c *gin.Context) {
	result := middleware.TenantDB(c, h.db).Where("id = ? AND tenant_id = ? AND user_id = ?", c.Param("id"), writingTenantID(c), writingUserID(c)).Delete(&types.WritingDraft{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete draft"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SDPivotWritingHandler) GenerateContent(c *gin.Context) {
	var req struct {
		Category         string `json:"category"`
		Prompt           string `json:"prompt"`
		KeyPoints        string `json:"key_points"`
		Template         string `json:"template"`
		CustomTemplate   string `json:"custom_template"`
		SourceType       string `json:"source_type"`
		SpaceID          string `json:"space_id"`
		Tags             string `json:"tags"`
		WebSearchEnabled bool   `json:"web_search_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
		return
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = strings.TrimSpace(req.KeyPoints)
	}
	if prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt or key_points is required"})
		return
	}
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "report"
	}
	label, structure := writingLevel(category)
	content := fmt.Sprintf("# %s\n\n## 写作要求\n%s\n\n## 建议结构\n%s\n\n## 参考约束\n请基于已授权的知识库资料补充事实；资料不足处请明确标注。", label, prompt, structure)
	template := strings.TrimSpace(req.Template)
	if template == "" {
		template = strings.TrimSpace(req.CustomTemplate)
	}
	if template != "" {
		content += "\n\n## 自定义模板\n" + template
	}
	c.JSON(http.StatusOK, gin.H{"content": content, "category": category, "template": template, "key_points": prompt, "source_type": req.SourceType, "web_search_enabled": req.WebSearchEnabled, "sources_count": 0})
}

func writingLevel(category string) (string, string) {
	switch strings.ToLower(category) {
	case "brief", "简报":
		return "简报", "结论；关键事实；风险；行动项"
	case "summary", "摘要":
		return "摘要", "背景；核心结论；关键依据；待确认事项"
	case "short", "short_article", "短文":
		return "短文", "标题；导语；主体论述；结语"
	case "notice":
		return "通知", "标题；适用范围；事项；时间地点；要求；联系人"
	case "technical":
		return "技术文档", "背景目标；方案；接口流程；约束风险；验证运维"
	case "minutes":
		return "会议纪要", "会议基本信息；讨论要点；决议；行动项"
	case "proposal":
		return "方案", "执行摘要；问题目标；方案；实施计划；风险与指标"
	default:
		return "报告", "摘要；背景目标；事实分析；问题风险；结论建议"
	}
}

func (h *SDPivotWritingHandler) ExportDraft(c *gin.Context) {
	var req struct {
		Format string `json:"format"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var draft types.WritingDraft
	if err := middleware.TenantDB(c, h.db).Where("id = ? AND tenant_id = ? AND user_id = ?", c.Param("id"), writingTenantID(c), writingUserID(c)).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}
	name := strings.TrimSpace(draft.Title)
	if name == "" {
		name = "writing-draft"
	}
	name = strings.NewReplacer("/", "-", "\\", "-", "\n", " ").Replace(name)
	switch strings.ToLower(req.Format) {
	case "markdown", "md", "":
		c.Header("Content-Disposition", `attachment; filename="`+name+`.md"`)
		c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(draft.Content))
	case "docx":
		c.Header("Content-Disposition", `attachment; filename="`+name+`.docx"`)
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", simpleDocx(draft.Content))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "format must be markdown or docx"})
	}
}

func simpleDocx(markdown string) []byte {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	w, _ := z.Create("[Content_Types].xml")
	_, _ = w.Write([]byte(`<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`))
	w, _ = z.Create("word/document.xml")
	var body strings.Builder
	for _, line := range strings.Split(markdown, "\n") {
		body.WriteString("<w:p><w:r><w:t>")
		body.WriteString(html.EscapeString(strings.TrimLeft(line, "# ")))
		body.WriteString("</w:t></w:r></w:p>")
	}
	_, _ = w.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` + body.String() + `</w:body></w:document>`))
	_ = z.Close()
	return b.Bytes()
}
