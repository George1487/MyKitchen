package main

import (
	"MyKitchen/internal/config"
	"MyKitchen/internal/repository/sql"
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"strings"
)

func main() {
	slug := flag.String("slug", "", "stable partner identifier")
	name := flag.String("name", "", "partner display name")
	flag.Parse()
	if strings.TrimSpace(*slug) == "" || strings.TrimSpace(*name) == "" {
		log.Fatal("-slug and -name are required")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		log.Fatal(err)
	}
	key := "pk_" + base64.RawURLEncoding.EncodeToString(secret)
	ctx := context.Background()
	app, err := config.NewApp(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()
	id, err := sql.NewPartnerSqlRepository(app.DB.Pool()).ProvisionPartner(ctx, *slug, *name, key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("partner_id=%d\napi_key=%s\n", id, key)
}
