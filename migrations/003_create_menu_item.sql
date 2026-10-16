-- +goose Up
CREATE TABLE menu_items(
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    price INTEGER NOT NULL,
    available_quantity INTEGER NOT NULL DEFAULT 0,
    restaurant_id INTEGER NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    is_available BOOLEAN NOT NULL DEFAULT TRUE
);

-- +goose Down
DROP TABLE menu_items;
