-- +goose Up
CREATE TABLE partners (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE partner_keys (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    partner_id BIGINT NOT NULL REFERENCES partners(id) ON DELETE CASCADE,
    key_hash BYTEA NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

ALTER TABLE restaurants ADD COLUMN partner_id BIGINT REFERENCES partners(id);
CREATE INDEX restaurants_partner_id_idx ON restaurants(partner_id);

-- Existing restaurants have no verified owner and must not receive new orders.
UPDATE restaurants SET is_accepting_orders = FALSE WHERE partner_id IS NULL;

-- +goose Down
DROP INDEX restaurants_partner_id_idx;
ALTER TABLE restaurants DROP COLUMN partner_id;
DROP TABLE partner_keys;
DROP TABLE partners;
