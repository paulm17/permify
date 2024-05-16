-- +goose Up
CREATE TABLE IF NOT EXISTS attributes
(
    id            SERIAL  NOT NULL,
    entity_type   VARCHAR NOT NULL,
    entity_id     VARCHAR NOT NULL,
    attribute     VARCHAR NOT NULL,
    value         jsonb   NOT NULL,
    tenant_id     VARCHAR NOT NULL,
    created_tx_id bigint DEFAULT (txid_current()),
    expired_tx_id bigint DEFAULT ('0'),
    CONSTRAINT pk_attribute PRIMARY KEY (id),
    CONSTRAINT uq_attribute UNIQUE (tenant_id, entity_type, entity_id, attribute, created_tx_id, expired_tx_id),
    CONSTRAINT uq_attribute_not_expired UNIQUE (tenant_id, entity_type, entity_id, attribute, expired_tx_id)
);

-- +goose Down
DROP TABLE IF EXISTS attributes;