-- +migrate Up
ALTER TABLE orders ADD COLUMN sender_number VARCHAR(20);
ALTER TABLE orders ADD COLUMN paid_amount NUMERIC(10, 2);
ALTER TABLE orders ADD COLUMN processing_at TIMESTAMP WITH TIME ZONE;
