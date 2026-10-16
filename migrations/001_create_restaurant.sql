-- +goose Up
CREATE TABLE restaurants(
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL,
    phone TEXT NOT NULL,
    address TEXT NOT NULL,
    opening_time TIME NOT NULL,
    closing_time TIME NOT NULL,
    is_accepting_orders BOOLEAN NOT NULL DEFAULT TRUE
);

-- +goose Down
DROP TABLE restaurants;
