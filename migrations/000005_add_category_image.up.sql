-- +migrate Up
ALTER TABLE categories ADD COLUMN IF NOT EXISTS image_url VARCHAR(255);

-- +migrate Down
ALTER TABLE categories DROP COLUMN IF EXISTS image_url;
