-- +migrate Up
ALTER TABLE products ADD COLUMN colors TEXT[] DEFAULT '{}';
ALTER TABLE products ADD COLUMN sizes TEXT[] DEFAULT '{}';

ALTER TABLE cart_items ADD COLUMN selected_color VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE cart_items ADD COLUMN selected_size VARCHAR(100) NOT NULL DEFAULT '';

ALTER TABLE order_items ADD COLUMN selected_color VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE order_items ADD COLUMN selected_size VARCHAR(100) NOT NULL DEFAULT '';

ALTER TABLE cart_items DROP CONSTRAINT IF EXISTS unique_user_product;
ALTER TABLE cart_items ADD CONSTRAINT unique_user_product_color_size UNIQUE (user_id, product_id, selected_color, selected_size);
