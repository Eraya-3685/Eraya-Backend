-- +migrate Up
ALTER TABLE store_settings ADD COLUMN IF NOT EXISTS logo_url VARCHAR(255) DEFAULT '';
