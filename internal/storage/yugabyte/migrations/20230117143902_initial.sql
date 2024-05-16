-- +goose Up
CREATE TABLE IF NOT EXISTS relation_tuples (
    id               SERIAL  NOT NULL,
    entity_type      VARCHAR NOT NULL,
    entity_id        VARCHAR NOT NULL,
    relation         VARCHAR NOT NULL,
    subject_type     VARCHAR NOT NULL,
    subject_id       VARCHAR NOT NULL,
    subject_relation VARCHAR NOT NULL,
    created_tx_id    bigint DEFAULT (txid_current()),
    expired_tx_id    bigint DEFAULT ('0'),
    tenant_id VARCHAR NOT NULL DEFAULT 't1',
    CONSTRAINT pk_relation_tuple PRIMARY KEY (id),
    CONSTRAINT uq_relation_tuple UNIQUE (tenant_id, entity_type, entity_id, relation, subject_type, subject_id, subject_relation, created_tx_id, expired_tx_id),
    CONSTRAINT uq_relation_tuple_not_expired UNIQUE (tenant_id, entity_type, entity_id, relation, subject_type, subject_id, subject_relation, expired_tx_id)
);

CREATE TABLE IF NOT EXISTS schema_definitions (
    name           VARCHAR NOT NULL,
    serialized_definition BYTEA    NOT NULL,
    tenant_id VARCHAR NOT NULL DEFAULT 't1',
    version               CHAR(20) NOT NULL,
    CONSTRAINT pk_schema_definition PRIMARY KEY (tenant_id, name, version)
);

CREATE TABLE IF NOT EXISTS transactions (
    id        bigint        DEFAULT (txid_current())     NOT NULL,
    tenant_id VARCHAR NOT NULL DEFAULT 't1',
    snapshot  txid_snapshot DEFAULT (txid_current_snapshot())    NOT NULL,
    timestamp TIMESTAMP   DEFAULT (now() AT TIME ZONE 'UTC') NOT NULL,
    CONSTRAINT pk_transaction PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_tuples_subject_relation ON relation_tuples (subject_type, subject_relation, entity_type, relation);

-- +goose Down
DROP TABLE IF EXISTS relation_tuples;
DROP TABLE IF EXISTS schema_definitions;
DROP TABLE IF EXISTS transactions;