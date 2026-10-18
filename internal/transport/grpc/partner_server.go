package grpcserver

import (
	"context"
	"errors"
	"strings"

	partnerv1 "MyKitchen/gen/partner/v1"
	"MyKitchen/internal/domain"
	"MyKitchen/internal/partnerauth"
	"MyKitchen/internal/service"
	menuservice "MyKitchen/internal/service/menu_item"
	orderservice "MyKitchen/internal/service/order"
	restaurantservice "MyKitchen/internal/service/restaurant"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const partnerAPIKeyMetadata = "x-partner-api-key"

type PartnerServer struct {
	partnerv1.UnimplementedPartnerServiceServer

	restaurants *restaurantservice.RestaurantService
	menuItems   *menuservice.MenuItemService
	orders      *orderservice.OrderService
	auth        partnerauth.Authenticator
}

func NewPartnerServer(
	restaurants *restaurantservice.RestaurantService,
	menuItems *menuservice.MenuItemService,
	orders *orderservice.OrderService,
	auth partnerauth.Authenticator,
) *PartnerServer {
	return &PartnerServer{restaurants: restaurants, menuItems: menuItems, orders: orders, auth: auth}
}

func RequirePartnerAPIKey(auth partnerauth.Authenticator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		values := metadata.ValueFromIncomingContext(ctx, partnerAPIKeyMetadata)
		if len(values) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid partner API key")
		}
		id, err := auth.Authenticate(ctx, values[0])
		if errors.Is(err, partnerauth.ErrInvalidKey) {
			return nil, status.Error(codes.Unauthenticated, "invalid partner API key")
		}
		if err != nil {
			return nil, status.Error(codes.Internal, "partner authentication failed")
		}
		return handler(partnerauth.WithPartnerID(ctx, id), req)
	}
}

func (s *PartnerServer) checkOwned(ctx context.Context, restaurantID int64) error {
	owned, err := s.auth.OwnsRestaurant(ctx, partnerauth.PartnerID(ctx), domain.RestaurantID(restaurantID))
	if err != nil {
		return status.Error(codes.Internal, "failed to check restaurant ownership")
	}
	if !owned {
		return status.Error(codes.NotFound, "restaurant not found")
	}
	return nil
}

func (s *PartnerServer) RegisterRestaurant(ctx context.Context, req *partnerv1.RegisterRestaurantRequest) (*partnerv1.RegisterRestaurantResponse, error) {
	if req == nil || strings.TrimSpace(req.GetName()) == "" || strings.TrimSpace(req.GetPhone()) == "" ||
		strings.TrimSpace(req.GetAddress()) == "" || req.GetOpeningTime() == "" || req.GetClosingTime() == "" {
		return nil, status.Error(codes.InvalidArgument, "restaurant details are required")
	}
	restaurant, err := s.restaurants.RegisterRestaurant(ctx, domain.Restaurant{
		PartnerID: partnerauth.PartnerID(ctx),
		Name:      strings.TrimSpace(req.GetName()), Phone: strings.TrimSpace(req.GetPhone()),
		Address: strings.TrimSpace(req.GetAddress()), OpeningTime: req.GetOpeningTime(),
		ClosingTime: req.GetClosingTime(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to register restaurant")
	}
	return &partnerv1.RegisterRestaurantResponse{RestaurantId: int64(restaurant.ID)}, nil
}

func (s *PartnerServer) CreateMenuItem(ctx context.Context, req *partnerv1.CreateMenuItemRequest) (*partnerv1.CreateMenuItemResponse, error) {
	if req == nil || req.GetRestaurantId() <= 0 || strings.TrimSpace(req.GetName()) == "" ||
		strings.TrimSpace(req.GetDescription()) == "" || req.GetPrice() < 0 || req.GetAvailableQuantity() < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid menu item")
	}
	if err := s.checkOwned(ctx, req.GetRestaurantId()); err != nil {
		return nil, err
	}
	item, err := s.menuItems.Add(ctx, domain.MenuItem{
		RestaurantID: domain.RestaurantID(req.GetRestaurantId()),
		Name:         strings.TrimSpace(req.GetName()), Description: strings.TrimSpace(req.GetDescription()),
		Price: req.GetPrice(), IsAvailable: req.GetIsAvailable(),
		AvailableQuantity: req.GetAvailableQuantity(),
	})
	if errors.Is(err, service.ErrRestaurantNotFound) {
		return nil, status.Error(codes.NotFound, "restaurant not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create menu item")
	}
	return &partnerv1.CreateMenuItemResponse{MenuItemId: int64(item.ID)}, nil
}

func (s *PartnerServer) SetAcceptingOrders(ctx context.Context, req *partnerv1.SetAcceptingOrdersRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetRestaurantId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid restaurant ID")
	}
	if err := s.checkOwned(ctx, req.GetRestaurantId()); err != nil {
		return nil, err
	}
	updated, err := s.restaurants.UpdateAcceptingOrders(ctx, domain.RestaurantID(req.GetRestaurantId()), req.GetIsAcceptingOrders())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update restaurant")
	}
	if !updated {
		return nil, status.Error(codes.NotFound, "restaurant not found")
	}
	return &emptypb.Empty{}, nil
}

func (s *PartnerServer) ListOrders(ctx context.Context, req *partnerv1.ListOrdersRequest) (*partnerv1.ListOrdersResponse, error) {
	if req == nil || req.GetRestaurantId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid restaurant ID")
	}
	if err := s.checkOwned(ctx, req.GetRestaurantId()); err != nil {
		return nil, err
	}
	orders, err := s.orders.GetOrdersByRestaurantID(ctx, domain.RestaurantID(req.GetRestaurantId()))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list orders")
	}
	response := &partnerv1.ListOrdersResponse{Orders: make([]*partnerv1.OrderSummary, 0, len(orders))}
	for _, order := range orders {
		orderStatus, ok := toProtoStatus(order.Status)
		if !ok {
			return nil, status.Error(codes.Internal, "unknown order status")
		}
		response.Orders = append(response.Orders, &partnerv1.OrderSummary{OrderId: int64(order.ID), Status: orderStatus})
	}
	return response, nil
}

func (s *PartnerServer) ChangeOrderStatus(ctx context.Context, req *partnerv1.ChangeOrderStatusRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetRestaurantId() <= 0 || req.GetOrderId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid restaurant or order ID")
	}
	newStatus, ok := fromProtoStatus(req.GetStatus())
	if !ok || newStatus == domain.OrderStatusNew {
		return nil, status.Error(codes.InvalidArgument, "invalid target order status")
	}
	if err := s.checkOwned(ctx, req.GetRestaurantId()); err != nil {
		return nil, err
	}
	err := s.orders.ChangeStatusForRestaurant(ctx, domain.RestaurantID(req.GetRestaurantId()), domain.OrderID(req.GetOrderId()), newStatus)
	if errors.Is(err, service.ErrOrderNotFound) {
		return nil, status.Error(codes.NotFound, "order not found")
	}
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "order status cannot be changed")
	}
	return &emptypb.Empty{}, nil
}

func toProtoStatus(value domain.OrderStatus) (partnerv1.OrderStatus, bool) {
	switch value {
	case domain.OrderStatusNew:
		return partnerv1.OrderStatus_ORDER_STATUS_NEW, true
	case domain.OrderStatusAccepted:
		return partnerv1.OrderStatus_ORDER_STATUS_ACCEPTED, true
	case domain.OrderStatusCancelled:
		return partnerv1.OrderStatus_ORDER_STATUS_CANCELLED, true
	case domain.OrderStatusDelivered:
		return partnerv1.OrderStatus_ORDER_STATUS_DELIVERED, true
	default:
		return partnerv1.OrderStatus_ORDER_STATUS_UNSPECIFIED, false
	}
}

func fromProtoStatus(value partnerv1.OrderStatus) (domain.OrderStatus, bool) {
	switch value {
	case partnerv1.OrderStatus_ORDER_STATUS_NEW:
		return domain.OrderStatusNew, true
	case partnerv1.OrderStatus_ORDER_STATUS_ACCEPTED:
		return domain.OrderStatusAccepted, true
	case partnerv1.OrderStatus_ORDER_STATUS_CANCELLED:
		return domain.OrderStatusCancelled, true
	case partnerv1.OrderStatus_ORDER_STATUS_DELIVERED:
		return domain.OrderStatusDelivered, true
	default:
		return "", false
	}
}
