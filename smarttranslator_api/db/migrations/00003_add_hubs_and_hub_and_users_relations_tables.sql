-- +goose Up
-- CREATE TABLE hubs (
--     id       TEXT PRIMARY KEY,
--     name     TEXT NOT NULL,
--     created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
--     deleted_at TIMESTAMP
-- );

-- CREATE TABLE users_and_hubs (
--     hub_id   TEXT NOT NULL REFERENCES hubs(id) ON DELETE CASCADE,
--     user_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
--     is_active BOOLEAN NOT NULL,
--     PRIMARY KEY (hub_id, user_id)
-- );

-- CREATE INDEX ON users_and_hubs (user_id, is_active);
-- CREATE INDEX ON users_and_hubs (user_id, hub_id, is_active);

-- +goose Down
-- DROP TABLE hubs;
-- DROP INDEX ON users_and_hubs (user_id, is_active);
-- DROP INDEX ON users_and_hubs (user_id, is_active);
