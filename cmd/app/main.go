package main

import (
	"context"
	"log"
	"net"

	partnerv1 "MyKitchen/gen/partner/v1"
	"MyKitchen/internal/config"
	"MyKitchen/internal/handler"
	"MyKitchen/internal/middleware"
	"MyKitchen/internal/partnerauth"
	"MyKitchen/internal/repository/sql"
	menuservice "MyKitchen/internal/service/menu_item"
	orderservice "MyKitchen/internal/service/order"
	restaurantservice "MyKitchen/internal/service/restaurant"
	"MyKitchen/internal/transaction"
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
	partnerRepo := sql.NewPartnerSqlRepository(db)
	if _, err := partnerRepo.ProvisionPartner(ctx, "demo", "Demo Kitchen", app.DemoPartnerAPIKey); err != nil {
		log.Fatal(err)
	}
	partnerAuth := partnerauth.NewService(partnerRepo)
	menuItemRepo := sql.NewMenuItemSqlRepository(db)
	orderRepo := sql.NewOrderSqlRepository(db)
	orderItemRepo := sql.NewOrderItemSqlRepository(db)
	txManager := transaction.NewPgxTxManager(db)

	restaurantService := restaurantservice.NewRestaurantService(restaurantRepo, txManager)
	menuItemService := menuservice.NewMenuItemService(menuItemRepo, restaurantRepo, txManager)
	orderService := orderservice.NewOrderService(orderRepo, menuItemRepo, orderItemRepo, restaurantRepo, txManager)

	restaurantHandler := handler.NewRestaurantHandler(restaurantService, menuItemService)
	menuItemHandler := handler.NewMenuItemHandler(menuItemService)
	orderHandler := handler.NewOrderHandler(orderService)

	router := gin.Default()
	router.HandleMethodNotAllowed = true
	router.GET("/restaurants", restaurantHandler.GetAllRestaurants)
	router.GET("/restaurants/:id", restaurantHandler.GetRestaurantByID)
	router.GET("/restaurants/:id/menu", menuItemHandler.GetMenuItemsByRestaurantID)
	router.POST("/orders", orderHandler.CreateOrder)

	// The HTTP partner API remains available for existing integrations.
	partner := router.Group("/partner", middleware.RequirePartnerAPIKey(partnerAuth))
	owner := middleware.RequireRestaurantOwnership(partnerAuth)
	partner.POST("/restaurants", restaurantHandler.RegisterPartnerRestaurant)
	partner.DELETE("/restaurants/:restaurant_id", owner, restaurantHandler.DeletePartnerRestaurant)
	partner.PATCH("/restaurants/:restaurant_id/accepting-orders", owner, restaurantHandler.UpdatePartnerAcceptingOrders)
	partner.POST("/restaurants/:restaurant_id/menu-items", owner, restaurantHandler.CreatePartnerMenuItem)
	partner.PATCH("/restaurants/:restaurant_id/menu-items/:menu_item_id", owner, restaurantHandler.UpdatePartnerMenuItem)
	partner.DELETE("/restaurants/:restaurant_id/menu-items/:menu_item_id", owner, restaurantHandler.DeletePartnerMenuItem)
	partner.GET("/restaurants/:restaurant_id/orders", owner, orderHandler.GetRestaurantOrders)
	partner.PATCH("/restaurants/:restaurant_id/orders/:order_id/status", owner, orderHandler.ChangeRestaurantOrderStatus)

	listener, err := net.Listen("tcp", app.GRPCListenAddr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpcserver.RequirePartnerAPIKey(partnerAuth)))
	partnerv1.RegisterPartnerServiceServer(grpcServer, grpcserver.NewPartnerServer(restaurantService, menuItemService, orderService, partnerAuth))
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
