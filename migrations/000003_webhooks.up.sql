CREATE TABLE webhooks (
    id         BIGSERIAL    PRIMARY KEY,
    url        TEXT         NOT NULL,
    secret     VARCHAR(128) NOT NULL DEFAULT '',
    events     TEXT[]       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
