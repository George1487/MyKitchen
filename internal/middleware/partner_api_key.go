package middleware

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/partnerauth"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const partnerAPIKeyHeader = "X-Partner-API-Key"

func RequirePartnerAPIKey(auth partnerauth.Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := auth.Authenticate(c.Request.Context(), c.GetHeader(partnerAPIKeyHeader))
		if errors.Is(err, partnerauth.ErrInvalidKey) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "некорректный ключ API партнёра"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
			return
		}
		c.Request = c.Request.WithContext(partnerauth.WithPartnerID(c.Request.Context(), id))

		c.Next()
	}
}

func RequireRestaurantOwnership(auth partnerauth.Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("restaurant_id"), 10, 64)
		if err != nil || id <= 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "некорректный идентификатор ресторана"})
			return
		}
		owned, err := auth.OwnsRestaurant(c.Request.Context(), partnerauth.PartnerID(c.Request.Context()), domain.RestaurantID(id))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "внутренняя ошибка сервера"})
			return
		}
		if !owned {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "ресторан не найден"})
			return
		}
		c.Next()
	}
}
