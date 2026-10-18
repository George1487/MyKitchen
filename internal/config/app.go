package config

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"MyKitchen/migrations"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// App holds the settings and live database connection for the main service.
type App struct {
	DB                DB     `envPrefix:"DB_"`
	DemoPartnerAPIKey string `env:"DEMO_PARTNER_API_KEY,required,notEmpty"`
	HTTPListenAddr    string `env:"HTTP_LISTEN_ADDR" envDefault:":8080"`
	GRPCListenAddr    string `env:"GRPC_LISTEN_ADDR" envDefault:":9090"`
}

// NewApp loads configuration, connects to PostgreSQL and applies migrations.
func NewApp(ctx context.Context) (*App, error) {
	app, err := loadApp()
	if err != nil {
		return nil, err
	}

	db, err := NewDB(ctx, app.DB)
	if err != nil {
		return nil, err
	}
	app.DB = *db

	if err := migrations.Up(ctx, app.DB.dsn()); err != nil {
		app.Close()
		return nil, fmt.Errorf("migrate PostgreSQL: %w", err)
	}
	return app, nil
}

func loadApp() (*App, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	var app App
	if err := env.Parse(&app); err != nil {
		return nil, fmt.Errorf("read application environment: %w", err)
	}
	if err := app.DB.validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(app.DemoPartnerAPIKey) == "" {
		return nil, fmt.Errorf("DEMO_PARTNER_API_KEY must not be empty")
	}
	for _, address := range []struct {
		name, value string
	}{
		{"HTTP_LISTEN_ADDR", app.HTTPListenAddr},
		{"GRPC_LISTEN_ADDR", app.GRPCListenAddr},
	} {
		_, port, err := net.SplitHostPort(address.value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", address.name, err)
		}
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return nil, fmt.Errorf("%s must have a port between 1 and 65535", address.name)
		}
	}
	return &app, nil
}

func (app *App) Close() {
	if app != nil {
		app.DB.Close()
	}
}
