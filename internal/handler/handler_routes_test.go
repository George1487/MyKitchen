package handler_test

import (
	"MyKitchen/internal/handler"
	"MyKitchen/internal/handler/dto"
	"MyKitchen/tests/mocks"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func TestHandlerRoutesMethodAndContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	unsupportedType := jsonBody(t, dto.ErrorResponse{Error: "неподдерживаемый тип содержимого"})
	type in struct {
		method      string
		path        string
		body        string
		contentType string
	}
	type out struct {
		status int
		body   string
	}
	routesTest := []struct {
		name string
		in   in
		out  out
	}{
		{name: "restaurants rejects wrong method", in: in{method: http.MethodPut, path: "/restaurants"}, out: out{status: http.StatusMethodNotAllowed, body: "405 method not allowed"}},
		{name: "restaurant by ID rejects wrong method", in: in{method: http.MethodPut, path: "/restaurants/17"}, out: out{status: http.StatusMethodNotAllowed, body: "405 method not allowed"}},
		{name: "menu rejects wrong method", in: in{method: http.MethodPut, path: "/restaurants/17/menu"}, out: out{status: http.StatusMethodNotAllowed, body: "405 method not allowed"}},
		{name: "orders rejects wrong method", in: in{method: http.MethodPut, path: "/orders"}, out: out{status: http.StatusMethodNotAllowed, body: "405 method not allowed"}},
		{name: "partner registration rejects wrong method", in: in{method: http.MethodPut, path: "/partner/restaurants"}, out: out{status: http.StatusMethodNotAllowed, body: "405 method not allowed"}},
		{name: "partner deletion rejects wrong method", in: in{method: http.MethodPut, path: "/partner/restaurants/17"}, out: out{status: http.StatusMethodNotAllowed, body: "405 method not allowed"}},
		{name: "accepting orders rejects wrong method", in: in{method: http.MethodPut, path: "/partner/restaurants/17/accepting-orders"}, out: out{status: http.StatusMethodNotAllowed, body: "405 method not allowed"}},
		{name: "menu item creation rejects wrong method", in: in{method: http.MethodPut, path: "/partner/restaurants/17/menu-items"}, out: out{status: http.StatusMethodNotAllowed, body: "405 method not allowed"}},
		{name: "menu item update rejects wrong method", in: in{method: http.MethodPut, path: "/partner/restaurants/17/menu-items/5"}, out: out{status: http.StatusMethodNotAllowed, body: "405 method not allowed"}},
		{name: "partner orders rejects wrong method", in: in{method: http.MethodPut, path: "/partner/restaurants/17/orders"}, out: out{status: http.StatusMethodNotAllowed, body: "405 method not allowed"}},
		{name: "status update rejects wrong method", in: in{method: http.MethodPut, path: "/partner/restaurants/17/orders/11/status"}, out: out{status: http.StatusMethodNotAllowed, body: "405 method not allowed"}},
		{name: "order creation rejects text", in: in{method: http.MethodPost, path: "/orders", contentType: "text/plain", body: `{}`}, out: out{status: http.StatusUnsupportedMediaType, body: unsupportedType}},
		{name: "order creation rejects missing content type", in: in{method: http.MethodPost, path: "/orders", body: `{}`}, out: out{status: http.StatusUnsupportedMediaType, body: unsupportedType}},
		{name: "order creation rejects malformed content type", in: in{method: http.MethodPost, path: "/orders", contentType: "application/json; charset=", body: `{}`}, out: out{status: http.StatusUnsupportedMediaType, body: unsupportedType}},
		{name: "order creation accepts JSON charset", in: in{method: http.MethodPost, path: "/orders", contentType: "application/json; charset=utf-8", body: "{"}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректные данные заказа"})}},
		{name: "registration rejects text", in: in{method: http.MethodPost, path: "/partner/restaurants", contentType: "text/plain", body: `{}`}, out: out{status: http.StatusUnsupportedMediaType, body: unsupportedType}},
		{name: "accepting orders rejects text", in: in{method: http.MethodPatch, path: "/partner/restaurants/17/accepting-orders", contentType: "text/plain", body: `{}`}, out: out{status: http.StatusUnsupportedMediaType, body: unsupportedType}},
		{name: "menu item creation rejects text", in: in{method: http.MethodPost, path: "/partner/restaurants/17/menu-items", contentType: "text/plain", body: `{}`}, out: out{status: http.StatusUnsupportedMediaType, body: unsupportedType}},
		{name: "menu item update rejects text", in: in{method: http.MethodPatch, path: "/partner/restaurants/17/menu-items/5", contentType: "text/plain", body: `{}`}, out: out{status: http.StatusUnsupportedMediaType, body: unsupportedType}},
		{name: "status update rejects text", in: in{method: http.MethodPatch, path: "/partner/restaurants/17/orders/11/status", contentType: "text/plain", body: `{}`}, out: out{status: http.StatusUnsupportedMediaType, body: unsupportedType}},
	}

	for _, tt := range routesTest {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			restaurants := handler.NewRestaurantHandler(mocks.NewMockRestaurantService(ctrl), mocks.NewMockMenuItemService(ctrl))
			menuItems := handler.NewMenuItemHandler(mocks.NewMockMenuItemService(ctrl))
			orders := handler.NewOrderHandler(mocks.NewMockOrderService(ctrl))
			router := gin.New()
			router.HandleMethodNotAllowed = true
			router.GET("/restaurants", restaurants.GetAllRestaurants)
			router.GET("/restaurants/:id", restaurants.GetRestaurantByID)
			router.GET("/restaurants/:id/menu", menuItems.GetMenuItemsByRestaurantID)
			router.POST("/orders", orders.CreateOrder)
			router.POST("/partner/restaurants", restaurants.RegisterPartnerRestaurant)
			router.DELETE("/partner/restaurants/:restaurant_id", restaurants.DeletePartnerRestaurant)
			router.PATCH("/partner/restaurants/:restaurant_id/accepting-orders", restaurants.UpdatePartnerAcceptingOrders)
			router.POST("/partner/restaurants/:restaurant_id/menu-items", restaurants.CreatePartnerMenuItem)
			router.PATCH("/partner/restaurants/:restaurant_id/menu-items/:menu_item_id", restaurants.UpdatePartnerMenuItem)
			router.DELETE("/partner/restaurants/:restaurant_id/menu-items/:menu_item_id", restaurants.DeletePartnerMenuItem)
			router.GET("/partner/restaurants/:restaurant_id/orders", orders.GetRestaurantOrders)
			router.PATCH("/partner/restaurants/:restaurant_id/orders/:order_id/status", orders.ChangeRestaurantOrderStatus)

			request := httptest.NewRequest(tt.in.method, tt.in.path, strings.NewReader(tt.in.body))
			request.Header.Set("Content-Type", tt.in.contentType)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}
