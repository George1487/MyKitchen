-- +goose Up
CREATE TABLE orders(
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    client_id INTEGER NOT NULL,
    restaurant_id INTEGER NOT NULL REFERENCES restaurants(id),
    total_price INTEGER NOT NULL,
    status TEXT NOT NULL,
    address TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE orders;
