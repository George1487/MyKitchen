package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	partnerv1 "MyKitchen/gen/partner/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

const rpcTimeout = 5 * time.Second

type config struct {
	grpcAddress   string
	partnerAPIKey string
	decision      string
	pollInterval  time.Duration
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	conn, err := grpc.NewClient(config.grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(partnerKeyInterceptor(config.partnerAPIKey)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	ctx := context.Background()
	waitForMainService(ctx, healthpb.NewHealthClient(conn))
	client := partnerv1.NewPartnerServiceClient(conn)

	restaurantID, err := registerRestaurant(ctx, client)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("restaurant registered: id=%d", restaurantID)

	if err := createDemoMenu(ctx, client, restaurantID); err != nil {
		log.Fatal(err)
	}
	if err := updateAcceptingOrders(ctx, client, restaurantID, true); err != nil {
		log.Fatal(err)
	}
	log.Printf("restaurant id=%d accepts orders; decision=%s", restaurantID, config.decision)

	pollOrders(ctx, client, restaurantID, config.decision, config.pollInterval)
}

func loadConfig() (config, error) {
	grpcAddress := os.Getenv("GRPC_ADDR")
	partnerAPIKey := os.Getenv("DEMO_PARTNER_API_KEY")
	decision := os.Getenv("RESTAURANT_DECISION")
	if decision == "" {
		decision = "accept"
	}
	if decision != "accept" && decision != "cancel" {
		return config{}, fmt.Errorf("RESTAURANT_DECISION must be accept or cancel")
	}

	pollInterval := 3 * time.Second
	if value := os.Getenv("POLL_INTERVAL"); value != "" {
		var err error
		pollInterval, err = time.ParseDuration(value)
		if err != nil || pollInterval <= 0 {
			return config{}, fmt.Errorf("POLL_INTERVAL must be a positive duration")
		}
	}
	if grpcAddress == "" || partnerAPIKey == "" {
		return config{}, fmt.Errorf("GRPC_ADDR and DEMO_PARTNER_API_KEY are required")
	}
	return config{grpcAddress: grpcAddress, partnerAPIKey: partnerAPIKey, decision: decision, pollInterval: pollInterval}, nil
}

func partnerKeyInterceptor(apiKey string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-partner-api-key", apiKey)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func waitForMainService(ctx context.Context, client healthpb.HealthClient) {
	for {
		callCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
		response, err := client.Check(callCtx, &healthpb.HealthCheckRequest{
			Service: partnerv1.PartnerService_ServiceDesc.ServiceName,
		})
		cancel()
		if err == nil && response.GetStatus() == healthpb.HealthCheckResponse_SERVING {
			return
		}
		log.Print("waiting for main gRPC service")
		time.Sleep(time.Second)
	}
}

func registerRestaurant(ctx context.Context, client partnerv1.PartnerServiceClient) (int64, error) {
	callCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()
	response, err := client.RegisterRestaurant(callCtx, &partnerv1.RegisterRestaurantRequest{
		Name: "Demo Kitchen", Phone: "+79990000000",
		Address:     "Москва, Тверская улица, 1",
		OpeningTime: "09:00", ClosingTime: "23:00",
	})
	if err != nil {
		return 0, fmt.Errorf("register restaurant: %w", err)
	}
	return response.GetRestaurantId(), nil
}

func createDemoMenu(ctx context.Context, client partnerv1.PartnerServiceClient, restaurantID int64) error {
	menu := []*partnerv1.CreateMenuItemRequest{
		{
			RestaurantId: restaurantID,
			Name:         "Пицца Маргарита", Description: "Томатный соус и сыр",
			Price: 500, IsAvailable: true, AvailableQuantity: 10,
		},
		{
			RestaurantId: restaurantID,
			Name:         "Лимонад", Description: "Домашний лимонад",
			Price: 150, IsAvailable: true, AvailableQuantity: 20,
		},
	}
	for _, item := range menu {
		callCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
		response, err := client.CreateMenuItem(callCtx, item)
		cancel()
		if err != nil {
			return fmt.Errorf("create menu item: %w", err)
		}
		log.Printf("menu item created: id=%d", response.GetMenuItemId())
	}
	return nil
}

func updateAcceptingOrders(ctx context.Context, client partnerv1.PartnerServiceClient, restaurantID int64, accepting bool) error {
	callCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()
	_, err := client.SetAcceptingOrders(callCtx, &partnerv1.SetAcceptingOrdersRequest{
		RestaurantId: restaurantID, IsAcceptingOrders: accepting,
	})
	return err
}

func pollOrders(ctx context.Context, client partnerv1.PartnerServiceClient, restaurantID int64, decision string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		processOrders(ctx, client, restaurantID, decision)
		<-ticker.C
	}
}

func processOrders(ctx context.Context, client partnerv1.PartnerServiceClient, restaurantID int64, decision string) {
	callCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	response, err := client.ListOrders(callCtx, &partnerv1.ListOrdersRequest{RestaurantId: restaurantID})
	cancel()
	if err != nil {
		log.Printf("list orders: %v", err)
		return
	}
	for _, order := range response.GetOrders() {
		switch order.GetStatus() {
		case partnerv1.OrderStatus_ORDER_STATUS_NEW:
			if decision == "cancel" {
				changeOrderStatus(ctx, client, restaurantID, order.GetOrderId(), partnerv1.OrderStatus_ORDER_STATUS_CANCELLED)
				continue
			}
			if changeOrderStatus(ctx, client, restaurantID, order.GetOrderId(), partnerv1.OrderStatus_ORDER_STATUS_ACCEPTED) {
				log.Printf("order accepted: id=%d", order.GetOrderId())
			}
		case partnerv1.OrderStatus_ORDER_STATUS_ACCEPTED:
			if decision == "accept" && changeOrderStatus(ctx, client, restaurantID, order.GetOrderId(), partnerv1.OrderStatus_ORDER_STATUS_DELIVERED) {
				log.Printf("order delivered: id=%d", order.GetOrderId())
			}
		}
	}
}

func changeOrderStatus(ctx context.Context, client partnerv1.PartnerServiceClient, restaurantID, orderID int64, newStatus partnerv1.OrderStatus) bool {
	callCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()
	_, err := client.ChangeOrderStatus(callCtx, &partnerv1.ChangeOrderStatusRequest{
		RestaurantId: restaurantID, OrderId: orderID, Status: newStatus,
	})
	if err != nil {
		log.Printf("change order id=%d to %s: %v", orderID, newStatus, err)
		return false
	}
	return true
}
