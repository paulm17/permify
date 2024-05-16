-- +goose Up
CREATE TABLE IF NOT EXISTS tenants (
   id   VARCHAR  NOT NULL,
   name VARCHAR NOT NULL,
   created_at TIMESTAMP DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL,
   CONSTRAINT pk_tenants PRIMARY KEY (id)
);

INSERT INTO tenants (id, name) VALUES ('t1', 'example tenant');

-- +goose Down
DROP TABLE IF EXISTS tenants;