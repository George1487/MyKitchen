package handler

import (
	"MyKitchen/internal/handler/dto"
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"
)

func acceptJSON(c *gin.Context) bool {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, dto.ErrorResponse{Error: "неподдерживаемый тип содержимого"})
		return false
	}
	return true
}
