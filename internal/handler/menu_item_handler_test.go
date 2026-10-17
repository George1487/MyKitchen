package handler_test

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/handler"
	"MyKitchen/internal/handler/dto"
	"MyKitchen/internal/service"
	"MyKitchen/tests/mocks"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func TestGetMenuItemsByRestaurantID(t *testing.T) {
	storageErr := errors.New("storage failure")
	items := []domain.MenuItem{{ID: 5, Name: "Pizza", Price: 350, IsAvailable: true}}
	type in struct{ restaurantID string }
	type out struct {
		status int
		body   string
	}
	getMenuItemsByRestaurantIDTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockMenuItemService)
	}{
		{
			name: "returns menu", in: in{restaurantID: "17"},
			out: out{status: http.StatusOK, body: jsonBody(t, []dto.MenuItemResponse{dto.NewMenuItemResponse(items[0])})},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().GetMenuItemsByRestaurantID(gomock.Any(), domain.RestaurantID(17)).Return(items, nil)
			},
		},
		{
			name: "returns empty menu", in: in{restaurantID: "17"},
			out: out{status: http.StatusOK, body: "[]"},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().GetMenuItemsByRestaurantID(gomock.Any(), domain.RestaurantID(17)).Return(nil, nil)
			},
		},
		{
			name: "rejects invalid restaurant ID", in: in{restaurantID: "invalid"},
			out:   out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})},
			setup: func(*mocks.MockMenuItemService) {},
		},
		{
			name: "restaurant missing", in: in{restaurantID: "17"},
			out: out{status: http.StatusNotFound, body: jsonBody(t, dto.ErrorResponse{Error: "ресторан не найден"})},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().GetMenuItemsByRestaurantID(gomock.Any(), domain.RestaurantID(17)).Return(nil, service.ErrRestaurantNotFound)
			},
		},
		{
			name: "storage failure", in: in{restaurantID: "17"},
			out: out{status: http.StatusInternalServerError, body: jsonBody(t, dto.ErrorResponse{Error: "внутренняя ошибка сервера"})},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().GetMenuItemsByRestaurantID(gomock.Any(), domain.RestaurantID(17)).Return(nil, storageErr)
			},
		},
	}

	for _, tt := range getMenuItemsByRestaurantIDTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockMenuItemService(gomock.NewController(t))
			tt.setup(mock)
			recorder := callHandler(handler.NewMenuItemHandler(mock).GetMenuItemsByRestaurantID, "", gin.Params{{Key: "id", Value: tt.in.restaurantID}})
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}
