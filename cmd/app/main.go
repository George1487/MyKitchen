package main

import (
	"context"
	"log"
	"net"

	partnerv1 "MyKitchen/gen/partner/v1"
	"MyKitchen/internal/config"
	"MyKitchen/internal/handler"
	"MyKitchen/internal/middleware"
	"MyKitchen/internal/repository/sql"
	"MyKitchen/internal/service"
	grpcserver "MyKitchen/internal/transport/grpc"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	ctx := context.Background()
	app, err := config.NewApp(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()
	db := app.DB.Pool()

	restaurantRepo := sql.NewRestaurantSqlRepository(db)
	menuItemRepo := sql.NewMenuItemSqlRepository(db)
	orderRepo := sql.NewOrderSqlRepository(db)
	orderItemRepo := sql.NewOrderItemSqlRepository(db)

	restaurantService := service.NewRestaurantService(restaurantRepo)
	menuItemService := service.NewMenuItemService(menuItemRepo, restaurantRepo)
	orderService := service.NewOrderService(orderRepo, menuItemRepo, orderItemRepo, restaurantRepo)

	restaurantHandler := handler.NewRestaurantHandler(restaurantService, menuItemService)
	menuItemHandler := handler.NewMenuItemHandler(menuItemService)
	orderHandler := handler.NewOrderHandler(orderService)

	router := gin.Default()
	router.GET("/restaurants", restaurantHandler.GetAllRestaurants)
	router.GET("/restaurants/:id", restaurantHandler.GetRestaurantByID)
	router.GET("/restaurants/:id/menu", menuItemHandler.GetMenuItemsByRestaurantID)
	router.POST("/orders", orderHandler.CreateOrder)

	// The HTTP partner API remains available for existing integrations.
	partner := router.Group("/partner", middleware.RequirePartnerAPIKey(app.PartnerAPIKey))
	partner.POST("/restaurants", restaurantHandler.RegisterPartnerRestaurant)
	partner.DELETE("/restaurants/:restaurant_id", restaurantHandler.DeletePartnerRestaurant)
	partner.PATCH("/restaurants/:restaurant_id/accepting-orders", restaurantHandler.UpdatePartnerAcceptingOrders)
	partner.POST("/restaurants/:restaurant_id/menu-items", restaurantHandler.CreatePartnerMenuItem)
	partner.PATCH("/restaurants/:restaurant_id/menu-items/:menu_item_id", restaurantHandler.UpdatePartnerMenuItem)
	partner.DELETE("/restaurants/:restaurant_id/menu-items/:menu_item_id", restaurantHandler.DeletePartnerMenuItem)
	partner.GET("/restaurants/:restaurant_id/orders", orderHandler.GetRestaurantOrders)
	partner.PATCH("/restaurants/:restaurant_id/orders/:order_id/status", orderHandler.ChangeRestaurantOrderStatus)

	listener, err := net.Listen("tcp", app.GRPCListenAddr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpcserver.RequirePartnerAPIKey(app.PartnerAPIKey)))
	partnerv1.RegisterPartnerServiceServer(grpcServer, grpcserver.NewPartnerServer(restaurantService, menuItemService, orderService))
	healthServer := health.NewServer()
	healthServer.SetServingStatus(partnerv1.PartnerService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	go func() {
		log.Printf("gRPC listening on %s", listener.Addr())
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	log.Fatal(router.Run(app.HTTPListenAddr))
}
