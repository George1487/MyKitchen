package config

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	for name, value := range map[string]string{
		"DB_HOST": "localhost", "DB_PORT": "5432", "DB_USER": "test_user",
		"DB_PASSWORD": "secret", "DB_NAME": "test_db", "DB_SSLMODE": "disable",
		"DEMO_PARTNER_API_KEY": "partner-key", "HTTP_LISTEN_ADDR": ":8080",
		"GRPC_LISTEN_ADDR": ":9090",
	} {
		t.Setenv(name, value)
	}
}

func TestLoadApp(t *testing.T) {
	setValidEnv(t)
	app, err := loadApp()
	require.NoError(t, err)
	require.Equal(t, "localhost", app.DB.Host)
	require.Equal(t, uint16(5432), app.DB.Port)
	require.Equal(t, "test_user", app.DB.User)
	require.Equal(t, "partner-key", app.DemoPartnerAPIKey)
}

func TestLoadAppRejectsInvalidEnvironment(t *testing.T) {
	for _, tc := range []struct {
		name, key, value, want string
	}{
		{"missing host", "DB_HOST", "", "DB_HOST"},
		{"bad port", "DB_PORT", "70000", "Port"},
		{"zero port", "DB_PORT", "0", "DB_PORT"},
		{"blank password", "DB_PASSWORD", "   ", "DB_PASSWORD"},
		{"blank API key", "DEMO_PARTNER_API_KEY", "   ", "DEMO_PARTNER_API_KEY"},
		{"bad gRPC address", "GRPC_LISTEN_ADDR", ":invalid", "GRPC_LISTEN_ADDR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv(tc.key, tc.value)
			_, err := loadApp()
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestLoadAppFromDotenv(t *testing.T) {
	keys := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"DEMO_PARTNER_API_KEY", "HTTP_LISTEN_ADDR", "GRPC_LISTEN_ADDR",
	}
	for _, key := range keys {
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
	}

	dir := t.TempDir()
	t.Chdir(dir)
	contents := "DB_HOST=localhost\nDB_USER=dotenv_user\nDB_PASSWORD=dotenv_secret\nDB_NAME=mykitchen\nDEMO_PARTNER_API_KEY=dotenv_key\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(contents), 0600))

	app, err := loadApp()
	require.NoError(t, err)
	require.Equal(t, "dotenv_user", app.DB.User)
	require.Equal(t, uint16(5432), app.DB.Port)
	require.Equal(t, "dotenv_key", app.DemoPartnerAPIKey)
}
func TestDBDSNEscapesCredentials(t *testing.T) {
	db := DB{Host: "localhost", Port: 5432, User: "user@name", Password: "p@ss:/word", Name: "mykitchen", SSLMode: "disable"}
	parsed, err := url.Parse(db.dsn())
	require.NoError(t, err)
	require.NotNil(t, parsed.User)
	password, ok := parsed.User.Password()
	require.True(t, ok)
	require.Equal(t, db.User, parsed.User.Username())
	require.Equal(t, db.Password, password)
	require.Equal(t, "localhost:5432", parsed.Host)
	require.Equal(t, "disable", parsed.Query().Get("sslmode"))
}
