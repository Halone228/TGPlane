CREATE TABLE api_keys (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(16)  NOT NULL,
    key_hash   VARCHAR(64)  NOT NULL UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
