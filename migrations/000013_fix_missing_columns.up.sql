-- +migrate Up
ALTER TABLE store_settings ADD COLUMN IF NOT EXISTS store_email VARCHAR(255);
ALTER TABLE store_settings ADD COLUMN IF NOT EXISTS store_phone VARCHAR(50);
ALTER TABLE store_settings ADD COLUMN IF NOT EXISTS store_address TEXT;

UPDATE store_settings SET 
    store_email = 'support@eraya.com',
    store_phone = '+8801234567890',
    store_address = 'Dhaka, Bangladesh'
WHERE id = 1;
