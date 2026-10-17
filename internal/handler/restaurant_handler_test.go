package handler_test

import (
	"MyKitchen/internal/domain"
	"MyKitchen/internal/handler"
	"MyKitchen/internal/handler/dto"
	"MyKitchen/internal/service"
	"MyKitchen/tests/mocks"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func noRestaurantCalls(*mocks.MockRestaurantService) {}
func noMenuItemCalls(*mocks.MockMenuItemService)     {}

func TestGetAllRestaurants(t *testing.T) {
	storageErr := errors.New("storage failure")
	restaurants := []domain.Restaurant{{ID: 17, Name: "Kitchen", IsAcceptingOrders: true}}
	type in struct{}
	type out struct {
		status int
		body   string
	}
	getAllRestaurantsTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockRestaurantService)
	}{
		{
			name: "returns restaurants", out: out{status: http.StatusOK, body: jsonBody(t, []dto.RestaurantResponse{dto.NewRestaurantResponse(restaurants[0])})},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().GetRestaurants(gomock.Any()).Return(restaurants, nil)
			},
		},
		{
			name: "returns empty list", out: out{status: http.StatusOK, body: "[]"},
			setup: func(mock *mocks.MockRestaurantService) { mock.EXPECT().GetRestaurants(gomock.Any()).Return(nil, nil) },
		},
		{
			name: "service failure", out: out{status: http.StatusInternalServerError, body: jsonBody(t, dto.ErrorResponse{Error: "внутренняя ошибка сервера"})},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().GetRestaurants(gomock.Any()).Return(nil, storageErr)
			},
		},
	}
	for _, tt := range getAllRestaurantsTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockRestaurantService(gomock.NewController(t))
			tt.setup(mock)
			recorder := callHandler(handler.NewRestaurantHandler(mock, nil).GetAllRestaurants, "", nil)
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}

func TestGetRestaurantByID(t *testing.T) {
	storageErr := errors.New("storage failure")
	restaurant := &domain.Restaurant{ID: 17, Name: "Kitchen", IsAcceptingOrders: true}
	type in struct{ id string }
	type out struct {
		status int
		body   string
	}
	getRestaurantByIDTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockRestaurantService)
	}{
		{
			name: "returns restaurant", in: in{id: "17"}, out: out{status: http.StatusOK, body: jsonBody(t, dto.NewRestaurantResponse(*restaurant))},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().GetRestaurantByID(gomock.Any(), domain.RestaurantID(17)).Return(restaurant, nil)
			},
		},
		{
			name: "rejects invalid ID", in: in{id: "abc"}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})},
			setup: noRestaurantCalls,
		},
		{
			name: "restaurant missing", in: in{id: "17"}, out: out{status: http.StatusNotFound, body: jsonBody(t, dto.ErrorResponse{Error: "ресторан не найден"})},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().GetRestaurantByID(gomock.Any(), domain.RestaurantID(17)).Return(nil, service.ErrRestaurantNotFound)
			},
		},
		{
			name: "service failure", in: in{id: "17"}, out: out{status: http.StatusInternalServerError, body: jsonBody(t, dto.ErrorResponse{Error: "внутренняя ошибка сервера"})},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().GetRestaurantByID(gomock.Any(), domain.RestaurantID(17)).Return(nil, storageErr)
			},
		},
	}
	for _, tt := range getRestaurantByIDTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockRestaurantService(gomock.NewController(t))
			tt.setup(mock)
			recorder := callHandler(handler.NewRestaurantHandler(mock, nil).GetRestaurantByID, "", gin.Params{{Key: "id", Value: tt.in.id}})
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}

func TestRegisterPartnerRestaurant(t *testing.T) {
	storageErr := errors.New("storage failure")
	validBody := `{"name":" Kitchen ","phone":" +79990000000 ","address":" Moscow ","opening_time":"09:00","closing_time":"23:00"}`
	restaurant := domain.Restaurant{Name: "Kitchen", Phone: "+79990000000", Address: "Moscow", OpeningTime: "09:00", ClosingTime: "23:00"}
	created := restaurant
	created.ID = 17
	created.IsAcceptingOrders = true
	type in struct{ body string }
	type out struct {
		status int
		body   string
	}
	registerPartnerRestaurantTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockRestaurantService)
	}{
		{
			name: "registers restaurant", in: in{body: validBody}, out: out{status: http.StatusCreated, body: jsonBody(t, dto.NewRestaurantResponse(created))},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().RegisterRestaurant(gomock.Any(), restaurant).Return(&created, nil)
			},
		},
		{
			name: "rejects malformed JSON", in: in{body: "{"}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректные данные ресторана"})},
			setup: noRestaurantCalls,
		},
		{
			name: "rejects blank name", in: in{body: `{"name":" ","phone":"+79990000000","address":"Moscow","opening_time":"09:00","closing_time":"23:00"}`}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректные данные ресторана"})},
			setup: noRestaurantCalls,
		},
		{
			name: "service failure", in: in{body: validBody}, out: out{status: http.StatusInternalServerError, body: jsonBody(t, dto.ErrorResponse{Error: "не удалось зарегистрировать ресторан"})},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().RegisterRestaurant(gomock.Any(), restaurant).Return(nil, storageErr)
			},
		},
	}
	for _, tt := range registerPartnerRestaurantTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockRestaurantService(gomock.NewController(t))
			tt.setup(mock)
			recorder := callHandler(handler.NewRestaurantHandler(mock, nil).RegisterPartnerRestaurant, tt.in.body, nil)
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}

func TestDeletePartnerRestaurant(t *testing.T) {
	storageErr := errors.New("storage failure")
	type in struct{ id string }
	type out struct {
		status int
		body   string
	}
	deletePartnerRestaurantTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockRestaurantService)
	}{
		{
			name: "deletes restaurant", in: in{id: "17"}, out: out{status: http.StatusNoContent},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().DeleteRestaurant(gomock.Any(), domain.RestaurantID(17)).Return(true, nil)
			},
		},
		{
			name: "rejects invalid ID", in: in{id: "0"}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})},
			setup: noRestaurantCalls,
		},
		{
			name: "restaurant missing", in: in{id: "17"}, out: out{status: http.StatusNotFound, body: jsonBody(t, dto.ErrorResponse{Error: "ресторан не найден"})},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().DeleteRestaurant(gomock.Any(), domain.RestaurantID(17)).Return(false, nil)
			},
		},
		{
			name: "service failure", in: in{id: "17"}, out: out{status: http.StatusConflict, body: jsonBody(t, dto.ErrorResponse{Error: "не удалось удалить ресторан"})},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().DeleteRestaurant(gomock.Any(), domain.RestaurantID(17)).Return(false, storageErr)
			},
		},
	}
	for _, tt := range deletePartnerRestaurantTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockRestaurantService(gomock.NewController(t))
			tt.setup(mock)
			recorder := callHandler(handler.NewRestaurantHandler(mock, nil).DeletePartnerRestaurant, "", gin.Params{{Key: "restaurant_id", Value: tt.in.id}})
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}

func TestUpdatePartnerAcceptingOrders(t *testing.T) {
	storageErr := errors.New("storage failure")
	type in struct {
		id   string
		body string
	}
	type out struct {
		status int
		body   string
	}
	updatePartnerAcceptingOrdersTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockRestaurantService)
	}{
		{
			name: "disables orders", in: in{id: "17", body: `{"is_accepting_orders":false}`}, out: out{status: http.StatusNoContent},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().UpdateAcceptingOrders(gomock.Any(), domain.RestaurantID(17), false).Return(true, nil)
			},
		},
		{
			name: "rejects invalid ID", in: in{id: "bad", body: `{"is_accepting_orders":false}`}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})},
			setup: noRestaurantCalls,
		},
		{
			name: "rejects missing flag", in: in{id: "17", body: `{}`}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректные данные для изменения приёма заказов"})},
			setup: noRestaurantCalls,
		},
		{
			name: "restaurant missing", in: in{id: "17", body: `{"is_accepting_orders":true}`}, out: out{status: http.StatusNotFound, body: jsonBody(t, dto.ErrorResponse{Error: "ресторан не найден"})},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().UpdateAcceptingOrders(gomock.Any(), domain.RestaurantID(17), true).Return(false, nil)
			},
		},
		{
			name: "service failure", in: in{id: "17", body: `{"is_accepting_orders":true}`}, out: out{status: http.StatusInternalServerError, body: jsonBody(t, dto.ErrorResponse{Error: "не удалось изменить приём заказов"})},
			setup: func(mock *mocks.MockRestaurantService) {
				mock.EXPECT().UpdateAcceptingOrders(gomock.Any(), domain.RestaurantID(17), true).Return(false, storageErr)
			},
		},
	}
	for _, tt := range updatePartnerAcceptingOrdersTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockRestaurantService(gomock.NewController(t))
			tt.setup(mock)
			recorder := callHandler(handler.NewRestaurantHandler(mock, nil).UpdatePartnerAcceptingOrders, tt.in.body, gin.Params{{Key: "restaurant_id", Value: tt.in.id}})
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}

func TestCreatePartnerMenuItem(t *testing.T) {
	storageErr := errors.New("storage failure")
	validBody := `{"name":" Pizza ","description":" Cheese ","price":350,"is_available":true,"available_quantity":5}`
	item := domain.MenuItem{Name: "Pizza", Description: "Cheese", Price: 350, IsAvailable: true, AvailableQuantity: 5, RestaurantID: 17}
	created := item
	created.ID = 5
	type in struct {
		id   string
		body string
	}
	type out struct {
		status int
		body   string
	}
	createPartnerMenuItemTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockMenuItemService)
	}{
		{
			name: "creates menu item", in: in{id: "17", body: validBody}, out: out{status: http.StatusCreated, body: jsonBody(t, dto.PartnerMenuItemResponse{ID: 5, Name: "Pizza", Description: "Cheese", Price: 350, IsAvailable: true, AvailableQuantity: 5})},
			setup: func(mock *mocks.MockMenuItemService) { mock.EXPECT().Add(gomock.Any(), item).Return(&created, nil) },
		},
		{
			name: "rejects invalid restaurant ID", in: in{id: "bad", body: validBody}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})},
			setup: noMenuItemCalls,
		},
		{
			name: "rejects malformed JSON", in: in{id: "17", body: "{"}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректные данные позиции меню"})},
			setup: noMenuItemCalls,
		},
		{
			name: "rejects negative price", in: in{id: "17", body: `{"name":"Pizza","description":"Cheese","price":-1}`}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректные данные позиции меню"})},
			setup: noMenuItemCalls,
		},
		{
			name: "restaurant missing", in: in{id: "17", body: validBody}, out: out{status: http.StatusNotFound, body: jsonBody(t, dto.ErrorResponse{Error: "ресторан не найден"})},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().Add(gomock.Any(), item).Return(nil, service.ErrRestaurantNotFound)
			},
		},
		{
			name: "service failure", in: in{id: "17", body: validBody}, out: out{status: http.StatusInternalServerError, body: jsonBody(t, dto.ErrorResponse{Error: "не удалось создать позицию меню"})},
			setup: func(mock *mocks.MockMenuItemService) { mock.EXPECT().Add(gomock.Any(), item).Return(nil, storageErr) },
		},
	}
	for _, tt := range createPartnerMenuItemTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockMenuItemService(gomock.NewController(t))
			tt.setup(mock)
			recorder := callHandler(handler.NewRestaurantHandler(nil, mock).CreatePartnerMenuItem, tt.in.body, gin.Params{{Key: "restaurant_id", Value: tt.in.id}})
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}

func TestUpdatePartnerMenuItem(t *testing.T) {
	storageErr := errors.New("storage failure")
	current := domain.MenuItem{ID: 5, Name: "Pizza", Description: "Cheese", Price: 350, IsAvailable: true, AvailableQuantity: 5, RestaurantID: 17}
	type in struct {
		restaurantID string
		itemID       string
		body         string
	}
	type out struct {
		status int
		body   string
	}
	updatePartnerMenuItemTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockMenuItemService)
	}{
		{
			name: "updates menu item", in: in{restaurantID: "17", itemID: "5", body: `{"name":" New Pizza ","available_quantity":0}`},
			out: out{status: http.StatusOK, body: jsonBody(t, dto.PartnerMenuItemResponse{ID: 5, Name: "New Pizza", Description: "Cheese", Price: 350, IsAvailable: true, AvailableQuantity: 0})},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().Update(gomock.Any(), domain.MenuItemID(5), domain.RestaurantID(17), gomock.Any()).DoAndReturn(
					func(_ context.Context, _ domain.MenuItemID, _ domain.RestaurantID, apply func(*domain.MenuItem) error) (*domain.MenuItem, error) {
						item := current
						require.NoError(t, apply(&item))
						require.Equal(t, "New Pizza", item.Name)
						require.Zero(t, item.AvailableQuantity)
						return &item, nil
					},
				)
			},
		},
		{
			name: "rejects invalid restaurant ID", in: in{restaurantID: "bad", itemID: "5", body: `{}`},
			out:   out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})},
			setup: noMenuItemCalls,
		},
		{
			name: "rejects invalid item ID", in: in{restaurantID: "17", itemID: "0", body: `{}`},
			out:   out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор позиции меню"})},
			setup: noMenuItemCalls,
		},
		{
			name: "rejects malformed JSON", in: in{restaurantID: "17", itemID: "5", body: "{"},
			out:   out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректные данные позиции меню"})},
			setup: noMenuItemCalls,
		},
		{
			name: "rejects empty update", in: in{restaurantID: "17", itemID: "5", body: `{}`},
			out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректные данные позиции меню"})},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().Update(gomock.Any(), domain.MenuItemID(5), domain.RestaurantID(17), gomock.Any()).DoAndReturn(
					func(_ context.Context, _ domain.MenuItemID, _ domain.RestaurantID, apply func(*domain.MenuItem) error) (*domain.MenuItem, error) {
						item := current
						return nil, apply(&item)
					},
				)
			},
		},
		{
			name: "item missing", in: in{restaurantID: "17", itemID: "5", body: `{"price":400}`},
			out: out{status: http.StatusNotFound, body: jsonBody(t, dto.ErrorResponse{Error: "позиция меню не найдена"})},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().Update(gomock.Any(), domain.MenuItemID(5), domain.RestaurantID(17), gomock.Any()).Return(nil, service.ErrMenuItemNotFound)
			},
		},
		{
			name: "service failure", in: in{restaurantID: "17", itemID: "5", body: `{"price":400}`},
			out: out{status: http.StatusInternalServerError, body: jsonBody(t, dto.ErrorResponse{Error: "не удалось обновить позицию меню"})},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().Update(gomock.Any(), domain.MenuItemID(5), domain.RestaurantID(17), gomock.Any()).Return(nil, storageErr)
			},
		},
	}
	for _, tt := range updatePartnerMenuItemTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockMenuItemService(gomock.NewController(t))
			tt.setup(mock)
			params := gin.Params{{Key: "restaurant_id", Value: tt.in.restaurantID}, {Key: "menu_item_id", Value: tt.in.itemID}}
			recorder := callHandler(handler.NewRestaurantHandler(nil, mock).UpdatePartnerMenuItem, tt.in.body, params)
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}

func TestDeletePartnerMenuItem(t *testing.T) {
	storageErr := errors.New("storage failure")
	type in struct {
		restaurantID string
		itemID       string
	}
	type out struct {
		status int
		body   string
	}
	deletePartnerMenuItemTest := []struct {
		name  string
		in    in
		out   out
		setup func(*mocks.MockMenuItemService)
	}{
		{
			name: "deletes menu item", in: in{restaurantID: "17", itemID: "5"}, out: out{status: http.StatusNoContent},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().Delete(gomock.Any(), domain.MenuItemID(5), domain.RestaurantID(17)).Return(nil)
			},
		},
		{
			name: "rejects invalid restaurant ID", in: in{restaurantID: "bad", itemID: "5"}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор ресторана"})},
			setup: noMenuItemCalls,
		},
		{
			name: "rejects invalid item ID", in: in{restaurantID: "17", itemID: "bad"}, out: out{status: http.StatusBadRequest, body: jsonBody(t, dto.ErrorResponse{Error: "некорректный идентификатор позиции меню"})},
			setup: noMenuItemCalls,
		},
		{
			name: "item missing", in: in{restaurantID: "17", itemID: "5"}, out: out{status: http.StatusNotFound, body: jsonBody(t, dto.ErrorResponse{Error: "позиция меню не найдена"})},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().Delete(gomock.Any(), domain.MenuItemID(5), domain.RestaurantID(17)).Return(service.ErrMenuItemNotFound)
			},
		},
		{
			name: "service failure", in: in{restaurantID: "17", itemID: "5"}, out: out{status: http.StatusInternalServerError, body: jsonBody(t, dto.ErrorResponse{Error: "не удалось удалить позицию меню"})},
			setup: func(mock *mocks.MockMenuItemService) {
				mock.EXPECT().Delete(gomock.Any(), domain.MenuItemID(5), domain.RestaurantID(17)).Return(storageErr)
			},
		},
	}
	for _, tt := range deletePartnerMenuItemTest {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockMenuItemService(gomock.NewController(t))
			tt.setup(mock)
			params := gin.Params{{Key: "restaurant_id", Value: tt.in.restaurantID}, {Key: "menu_item_id", Value: tt.in.itemID}}
			recorder := callHandler(handler.NewRestaurantHandler(nil, mock).DeletePartnerMenuItem, "", params)
			checkResponse(t, recorder, tt.out.status, tt.out.body)
		})
	}
}
