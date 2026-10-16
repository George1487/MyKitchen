package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

const partnerAPIKeyHeader = "X-Partner-API-Key"

func RequirePartnerAPIKey(expectedAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		providedAPIKey := c.GetHeader(partnerAPIKeyHeader)
		if expectedAPIKey == "" || subtle.ConstantTimeCompare([]byte(providedAPIKey), []byte(expectedAPIKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "некорректный ключ API партнёра"})
			return
		}

		c.Next()
	}
}
