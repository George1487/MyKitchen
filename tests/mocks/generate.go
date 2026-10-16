// Package mocks contains mocks generated from repository interfaces.
package mocks

//go:generate go run go.uber.org/mock/mockgen -source=../../internal/repository/menu_item_repository.go -destination=mock_menu_item_repository.go -package=mocks
//go:generate go run go.uber.org/mock/mockgen -source=../../internal/repository/order_repository.go -destination=mock_order_repository.go -package=mocks
//go:generate go run go.uber.org/mock/mockgen -source=../../internal/repository/order_item_repository.go -destination=mock_order_item_repository.go -package=mocks
//go:generate go run go.uber.org/mock/mockgen -source=../../internal/repository/restaurant_repository.go -destination=mock_restaurant_repository.go -package=mocks
