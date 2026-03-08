CREATE TABLE workers (
    id          VARCHAR(128) PRIMARY KEY,
    addr        VARCHAR(256) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
