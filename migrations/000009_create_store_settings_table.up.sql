-- +migrate Up
CREATE TABLE IF NOT EXISTS store_settings (
    id SERIAL PRIMARY KEY,
    free_shipping_threshold DECIMAL(10, 2) NOT NULL DEFAULT 1999.00,
    standard_delivery_fee DECIMAL(10, 2) NOT NULL DEFAULT 85.00,
    tax_percentage DECIMAL(5, 2) NOT NULL DEFAULT 5.00
);

INSERT INTO store_settings (free_shipping_threshold, standard_delivery_fee, tax_percentage)
VALUES (1999.00, 85.00, 5.00)
ON CONFLICT DO NOTHING;
