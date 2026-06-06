-- +migrate Up
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS image_url VARCHAR(255);

ALTER TABLE products ADD COLUMN IF NOT EXISTS variation_stock JSONB;

CREATE TABLE IF NOT EXISTS coupons (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    discount_type VARCHAR(20) NOT NULL, -- 'percentage' or 'flat'
    discount_value DECIMAL(10, 2) NOT NULL,
    min_cart_value DECIMAL(10, 2) DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS coupon_code VARCHAR(50);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_amount DECIMAL(10, 2) DEFAULT 0;
