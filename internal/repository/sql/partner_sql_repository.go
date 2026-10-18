package sql

import (
	"MyKitchen/internal/domain"
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PartnerSqlRepository struct{ db *pgxpool.Pool }

func NewPartnerSqlRepository(db *pgxpool.Pool) *PartnerSqlRepository {
	return &PartnerSqlRepository{db: db}
}

func (r *PartnerSqlRepository) FindActivePartnerByKeyHash(ctx context.Context, hash [32]byte) (domain.PartnerID, error) {
	var id domain.PartnerID
	err := r.db.QueryRow(ctx, `SELECT p.id FROM partners p
		JOIN partner_keys k ON k.partner_id = p.id
		WHERE k.key_hash = $1 AND k.revoked_at IS NULL AND p.is_active`, hash[:]).Scan(&id)
	return id, err
}

func (r *PartnerSqlRepository) OwnsRestaurant(ctx context.Context, partnerID domain.PartnerID, restaurantID domain.RestaurantID) (bool, error) {
	var owned bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM restaurants WHERE id = $1 AND partner_id = $2)`,
		restaurantID, partnerID).Scan(&owned)
	return owned, err
}

// ProvisionPartner is an administrative operation. It creates or rotates a partner key.
func (r *PartnerSqlRepository) ProvisionPartner(ctx context.Context, slug, name, key string) (domain.PartnerID, error) {
	if slug == "" || name == "" || key == "" {
		return 0, fmt.Errorf("slug, name and key are required")
	}
	hash := sha256.Sum256([]byte(key))
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id domain.PartnerID
	err = tx.QueryRow(ctx, `INSERT INTO partners (slug, name) VALUES ($1, $2)
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, is_active = TRUE
		RETURNING id`, slug, name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("upsert partner: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE partner_keys SET revoked_at = NOW()
		WHERE partner_id = $1 AND revoked_at IS NULL AND key_hash <> $2`, id, hash[:])
	if err != nil {
		return 0, fmt.Errorf("revoke old keys: %w", err)
	}
	result, err := tx.Exec(ctx, `INSERT INTO partner_keys (partner_id, key_hash) VALUES ($1, $2)
		ON CONFLICT (key_hash) DO UPDATE SET revoked_at = NULL
		WHERE partner_keys.partner_id = EXCLUDED.partner_id`, id, hash[:])
	if err != nil {
		return 0, fmt.Errorf("store partner key: %w", err)
	}
	if result.RowsAffected() != 1 {
		return 0, fmt.Errorf("key is already assigned to another partner")
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}
