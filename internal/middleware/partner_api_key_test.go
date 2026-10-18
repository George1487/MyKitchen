package middleware

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/partnerauth"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakePartnerAuth struct{}

func (fakePartnerAuth) Authenticate(_ context.Context, key string) (domain.PartnerID, error) {
	switch key {
	case "key-a":
		return 1, nil
	case "key-b":
		return 2, nil
	default:
		return 0, partnerauth.ErrInvalidKey
	}
}

func (fakePartnerAuth) OwnsRestaurant(_ context.Context, partnerID domain.PartnerID, restaurantID domain.RestaurantID) (bool, error) {
	return partnerID == 1 && restaurantID == 7, nil
}

func TestPartnerCannotAccessAnotherRestaurant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := fakePartnerAuth{}
	router := gin.New()
	partner := router.Group("/partner", RequirePartnerAPIKey(auth))
	partner.GET("/restaurants/:restaurant_id/orders", RequireRestaurantOwnership(auth), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	for _, tc := range []struct {
		key  string
		want int
	}{
		{"key-a", http.StatusOK},
		{"key-b", http.StatusNotFound},
		{"unknown", http.StatusUnauthorized},
	} {
		req := httptest.NewRequest(http.MethodGet, "/partner/restaurants/7/orders", nil)
		req.Header.Set("X-Partner-API-Key", tc.key)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		require.Equal(t, tc.want, response.Code, "key=%s", tc.key)
	}
}
