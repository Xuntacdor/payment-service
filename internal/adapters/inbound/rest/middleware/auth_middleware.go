package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIKeyAuthMiddleware checks for an X-API-Key header and compares it to expectedKey.
func APIKeyAuthMiddleware(expectedKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If expectedKey is not configured (e.g. empty), you could technically skip auth
		// but typically it should be required in production.
		if expectedKey == "" {
			c.Next()
			return
		}

		clientKey := c.GetHeader("X-API-Key")
		if clientKey == "" || clientKey != expectedKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: missing or invalid API key",
			})
			return
		}

		c.Next()
	}
}
