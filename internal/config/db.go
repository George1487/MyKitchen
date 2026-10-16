package config

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB contains the PostgreSQL connection settings and its active pool.
type DB struct {
	Host     string `env:"HOST,required,notEmpty"`
	Port     uint16 `env:"PORT" envDefault:"5432"`
	User     string `env:"USER,required,notEmpty"`
	Password string `env:"PASSWORD,required,notEmpty"`
	Name     string `env:"NAME,required,notEmpty"`
	SSLMode  string `env:"SSLMODE" envDefault:"disable"`

	pool *pgxpool.Pool
}

// NewDB validates the settings and establishes a PostgreSQL connection.
func NewDB(ctx context.Context, settings DB) (*DB, error) {
	if err := settings.validate(); err != nil {
		return nil, err
	}
	pool, err := pgxpool.New(ctx, settings.dsn())
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	settings.pool = pool
	return &settings, nil
}

func (db DB) validate() error {
	for _, field := range []struct {
		name, value string
	}{
		{"DB_HOST", db.Host},
		{"DB_USER", db.User},
		{"DB_PASSWORD", db.Password},
		{"DB_NAME", db.Name},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s must not be empty", field.name)
		}
	}
	if db.Port == 0 {
		return fmt.Errorf("DB_PORT must be between 1 and 65535")
	}
	switch db.SSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("DB_SSLMODE is invalid")
	}
	return nil
}

func (db DB) dsn() string {
	address := net.JoinHostPort(db.Host, strconv.Itoa(int(db.Port)))
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(db.User, db.Password),
		Host:   address,
		Path:   "/" + db.Name,
	}
	u.RawQuery = url.Values{"sslmode": {db.SSLMode}}.Encode()
	return u.String()
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *DB) Close() {
	if db != nil && db.pool != nil {
		db.pool.Close()
	}
}
