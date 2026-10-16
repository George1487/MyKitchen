-- +goose Up
CREATE TABLE order_items(
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    menu_item_id INTEGER NOT NULL REFERENCES menu_items(id),
    price INTEGER NOT NULL,
    quantity INTEGER NOT NULL
);

-- +goose Down
DROP TABLE order_items;
