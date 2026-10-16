package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	for name, value := range map[string]string{
		"DB_HOST": "localhost", "DB_PORT": "5432", "DB_USER": "test_user",
		"DB_PASSWORD": "secret", "DB_NAME": "test_db", "DB_SSLMODE": "disable",
		"PARTNER_API_KEY": "partner-key", "HTTP_LISTEN_ADDR": ":8080",
		"GRPC_LISTEN_ADDR": ":9090",
	} {
		t.Setenv(name, value)
	}
}

func TestLoadApp(t *testing.T) {
	setValidEnv(t)
	app, err := loadApp()
	if err != nil {
		t.Fatal(err)
	}
	if app.DB.Host != "localhost" || app.DB.Port != 5432 || app.DB.User != "test_user" || app.PartnerAPIKey != "partner-key" {
		t.Fatal("application configuration was parsed incorrectly")
	}
}

func TestLoadAppRejectsInvalidEnvironment(t *testing.T) {
	for _, tc := range []struct {
		name, key, value, want string
	}{
		{"missing host", "DB_HOST", "", "DB_HOST"},
		{"bad port", "DB_PORT", "70000", "Port"},
		{"zero port", "DB_PORT", "0", "DB_PORT"},
		{"blank password", "DB_PASSWORD", "   ", "DB_PASSWORD"},
		{"blank API key", "PARTNER_API_KEY", "   ", "PARTNER_API_KEY"},
		{"bad gRPC address", "GRPC_LISTEN_ADDR", ":invalid", "GRPC_LISTEN_ADDR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv(tc.key, tc.value)
			_, err := loadApp()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("loadApp() error = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestLoadAppFromDotenv(t *testing.T) {
	keys := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"PARTNER_API_KEY", "HTTP_LISTEN_ADDR", "GRPC_LISTEN_ADDR",
	}
	for _, key := range keys {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}

	dir := t.TempDir()
	t.Chdir(dir)
	contents := "DB_HOST=localhost\nDB_USER=dotenv_user\nDB_PASSWORD=dotenv_secret\nDB_NAME=mykitchen\nPARTNER_API_KEY=dotenv_key\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}

	app, err := loadApp()
	if err != nil {
		t.Fatal(err)
	}
	if app.DB.User != "dotenv_user" || app.DB.Port != 5432 || app.PartnerAPIKey != "dotenv_key" {
		t.Fatal("application did not load .env values and defaults")
	}
}
func TestDBDSNEscapesCredentials(t *testing.T) {
	db := DB{Host: "localhost", Port: 5432, User: "user@name", Password: "p@ss:/word", Name: "mykitchen", SSLMode: "disable"}
	parsed, err := url.Parse(db.dsn())
	if err != nil {
		t.Fatal(err)
	}
	password, ok := parsed.User.Password()
	if !ok || parsed.User.Username() != db.User || password != db.Password || parsed.Host != "localhost:5432" || parsed.Query().Get("sslmode") != "disable" {
		t.Fatalf("DSN did not preserve database settings")
	}
}
