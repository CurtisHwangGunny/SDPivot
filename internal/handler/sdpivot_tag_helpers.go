package handler

import (
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

func tenantID(c *gin.Context) uint64 {
	value, _ := types.TenantIDFromContext(c.Request.Context())
	return value
}
func userID(c *gin.Context) string {
	value, _ := types.UserIDFromContext(c.Request.Context())
	return value
}
func ptrTenant(value uint64) *uint64 { return &value }
