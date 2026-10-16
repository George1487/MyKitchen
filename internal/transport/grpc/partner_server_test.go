package grpcserver

import (
	"context"
	"net"
	"testing"
	"time"

	partnerv1 "MyKitchen/gen/partner/v1"
	"MyKitchen/internal/domain"
	"MyKitchen/internal/service"
	"MyKitchen/tests/mocks"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func testClient(t *testing.T, partner *PartnerServer) partnerv1.PartnerServiceClient {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(RequirePartnerAPIKey("test-key")))
	partnerv1.RegisterPartnerServiceServer(server, partner)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return partnerv1.NewPartnerServiceClient(conn)
}

func authorizedContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return metadata.AppendToOutgoingContext(ctx, "x-partner-api-key", "test-key")
}

func TestPartnerGRPCAuthenticationAndRegistration(t *testing.T) {
	ctrl := gomock.NewController(t)
	restaurants := mocks.NewMockRestaurantRepository(ctrl)
	restaurants.EXPECT().AddRestaurant(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, restaurant domain.Restaurant) (domain.RestaurantID, error) {
			if restaurant.Name != "Demo Kitchen" {
				t.Errorf("unexpected restaurant name: %q", restaurant.Name)
			}
			return 17, nil
		},
	)
	client := testClient(t, NewPartnerServer(service.NewRestaurantService(restaurants), nil, nil))
	request := &partnerv1.RegisterRestaurantRequest{
		Name: "Demo Kitchen", Phone: "+79990000000", Address: "Moscow",
		OpeningTime: "09:00", ClosingTime: "23:00",
	}

	noKeyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.RegisterRestaurant(noKeyCtx, request); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing key: got %v, want Unauthenticated", err)
	}
	response, err := client.RegisterRestaurant(authorizedContext(t), request)
	if err != nil {
		t.Fatal(err)
	}
	if response.GetRestaurantId() != 17 {
		t.Fatalf("restaurant ID = %d, want 17", response.GetRestaurantId())
	}
}

func TestPartnerGRPCOrderFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	orders := mocks.NewMockOrderRepository(ctrl)
	items := mocks.NewMockOrderItemRepository(ctrl)
	orders.EXPECT().GetOrdersByRestaurantID(gomock.Any(), domain.RestaurantID(7)).
		Return([]domain.Order{{ID: 42, RestaurantID: 7, Status: domain.OrderStatusNew}}, nil)
	items.EXPECT().GetByOrderID(gomock.Any(), domain.OrderID(42)).
		Return([]domain.OrderItem{}, nil)
	orders.EXPECT().GetByID(gomock.Any(), domain.OrderID(42)).
		Return(&domain.Order{ID: 42, RestaurantID: 7, Status: domain.OrderStatusNew}, nil)
	orders.EXPECT().UpdateStatus(gomock.Any(), domain.OrderID(42), domain.OrderStatusNew, domain.OrderStatusAccepted).
		Return(true, nil)

	orderService := service.NewOrderService(orders, nil, items, nil)
	client := testClient(t, NewPartnerServer(nil, nil, orderService))
	ctx := authorizedContext(t)

	list, err := client.ListOrders(ctx, &partnerv1.ListOrdersRequest{RestaurantId: 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.GetOrders()) != 1 || list.GetOrders()[0].GetOrderId() != 42 ||
		list.GetOrders()[0].GetStatus() != partnerv1.OrderStatus_ORDER_STATUS_NEW {
		t.Fatalf("unexpected orders: %v", list.GetOrders())
	}
	_, err = client.ChangeOrderStatus(ctx, &partnerv1.ChangeOrderStatusRequest{
		RestaurantId: 7, OrderId: 42, Status: partnerv1.OrderStatus_ORDER_STATUS_ACCEPTED,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ChangeOrderStatus(ctx, &partnerv1.ChangeOrderStatusRequest{
		RestaurantId: 7, OrderId: 42, Status: partnerv1.OrderStatus_ORDER_STATUS_UNSPECIFIED,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid status: got %v, want InvalidArgument", err)
	}
}
