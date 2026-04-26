-- +migrate Up
CREATE TABLE user_permissions (
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(50) NOT NULL,
    PRIMARY KEY (user_id, permission)
);

ALTER TABLE users DROP COLUMN IF EXISTS permissions;