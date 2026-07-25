package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders applies baseline browser hardening to every response.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		headers.Set("Permissions-Policy", "camera=(), geolocation=(), microphone=(), payment=(), usb=()")
		headers.Set("X-Frame-Options", "SAMEORIGIN")
		c.Next()
	}
}
