package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// auditServiceContextKey is the gin context key used by
// AuditServiceProvider to stash the running AuditLogService so that
// middleware functions (rbac.go's RequireRole / RequireOwnershipOrRole)
// can pull it out without needing the service threaded into their
// signatures. Same pattern as the langfuse gin middleware.
const auditServiceContextKey = "weknora.audit_service"

// AuditServiceProvider returns a gin middleware that injects the audit
// service into every request's gin.Context. Wiring is centralised in
// router.NewRouter so each request gets the same instance for the
// lifetime of the process; the middleware is a no-op when svc is nil
// (e.g. lite mode where audit isn't configured) so the rbac reject
// path degrades gracefully.
func AuditServiceProvider(svc interfaces.AuditLogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc != nil {
			c.Set(auditServiceContextKey, svc)
		}
		c.Next()
	}
}

// AuditSystemAdminOperation records successful mutating system-admin calls.
func AuditSystemAdminOperation() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Request == nil || c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions || c.Writer.Status() >= http.StatusBadRequest {
			return
		}
		svc := AuditServiceFromContext(c)
		if svc == nil {
			return
		}
		actorID, _ := types.UserIDFromContext(c.Request.Context())
		details, _ := json.Marshal(map[string]any{"status": c.Writer.Status()})
		_ = svc.Log(c.Request.Context(), &types.AuditLog{
			TenantID:      0,
			ActorUserID:   actorID,
			ActorRole:     "system_admin",
			Action:        types.AuditActionSystemAdminOperation,
			TargetType:    "system_admin_api",
			TargetID:      auditTargetID(c),
			RequestPath:   c.FullPath(),
			RequestMethod: c.Request.Method,
			Outcome:       types.AuditOutcomeSuccess,
			Details:       types.JSON(details),
			IPAddress:     c.ClientIP(),
		})
	}
}

// AuditAdminOperation records a successful tenant-scoped mutation after an
// Admin or Owner role guard authorizes it.
func AuditAdminOperation(c *gin.Context, requiredRole types.TenantRole) {
	if c == nil || c.Request == nil || c.IsAborted() || c.Writer.Status() >= http.StatusBadRequest || !isMutatingMethod(c.Request.Method) {
		return
	}
	svc := AuditServiceFromContext(c)
	if svc == nil {
		return
	}
	ctx := c.Request.Context()
	tenantID, _ := types.TenantIDFromContext(ctx)
	actorID, _ := types.UserIDFromContext(ctx)
	details, _ := json.Marshal(map[string]any{
		"required_role": requiredRole,
		"status":        c.Writer.Status(),
	})
	_ = svc.Log(ctx, &types.AuditLog{
		TenantID:      tenantID,
		ActorUserID:   actorID,
		ActorRole:     string(types.TenantRoleFromContext(ctx)),
		Action:        types.AuditActionAdminOperation,
		TargetType:    "admin_api",
		TargetID:      auditTargetID(c),
		RequestPath:   c.FullPath(),
		RequestMethod: c.Request.Method,
		Outcome:       types.AuditOutcomeSuccess,
		Details:       types.JSON(details),
		IPAddress:     c.ClientIP(),
	})
}

// AuditKnowledgeAccess records successful knowledge read and search requests.
func AuditKnowledgeAccess(targetType, param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Request == nil || c.Writer.Status() >= http.StatusBadRequest {
			return
		}
		svc := AuditServiceFromContext(c)
		if svc == nil {
			return
		}
		ctx := c.Request.Context()
		tenantID, _ := types.TenantIDFromContext(ctx)
		actorID, _ := types.UserIDFromContext(ctx)
		_ = svc.Log(ctx, &types.AuditLog{
			TenantID:      tenantID,
			ActorUserID:   actorID,
			ActorRole:     string(types.TenantRoleFromContext(ctx)),
			Action:        types.AuditActionKnowledgeAccessed,
			TargetType:    targetType,
			TargetID:      strings.TrimSpace(c.Param(param)),
			RequestPath:   c.FullPath(),
			RequestMethod: c.Request.Method,
			Outcome:       types.AuditOutcomeSuccess,
			IPAddress:     c.ClientIP(),
		})
	}
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func auditTargetID(c *gin.Context) string {
	for _, key := range []string{"id", "key", "user_id"} {
		if value := strings.TrimSpace(c.Param(key)); value != "" {
			return value
		}
	}
	return ""
}

// AuditServiceFromContext fetches the audit service injected by
// AuditServiceProvider, or returns nil if no provider was wired
// upstream. Callers MUST nil-check before invoking — audit failure
// must never break the underlying business operation.
func AuditServiceFromContext(c *gin.Context) interfaces.AuditLogService {
	if v, ok := c.Get(auditServiceContextKey); ok {
		if svc, ok := v.(interfaces.AuditLogService); ok {
			return svc
		}
	}
	return nil
}
