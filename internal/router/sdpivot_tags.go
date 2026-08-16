package router

import "github.com/gin-gonic/gin"

func RegisterSDPivotTagRoutes(r *gin.RouterGroup, params RouterParams, g *rbacGuards) {
	if params.SDPivotAdminHandler == nil || params.SDPivotTagAutoHandler == nil || params.SDPivotTagDimensionHandler == nil || params.SDPivotTagFeedbackHandler == nil {
		return
	}
	admin := r.Group("", g.Admin())
	params.SDPivotAdminHandler.RegisterRoutes(admin)
	params.SDPivotTagDimensionHandler.RegisterRoutes(admin)
	write := r.Group("", g.Contributor())
	params.SDPivotTagAutoHandler.RegisterRoutes(write)
	params.SDPivotTagFeedbackHandler.RegisterRoutes(write)
}
