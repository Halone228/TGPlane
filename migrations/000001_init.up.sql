CREATE TABLE accounts (
    id          BIGSERIAL PRIMARY KEY,
    phone       VARCHAR(32)  NOT NULL UNIQUE,
    session_id  VARCHAR(128) NOT NULL UNIQUE,
    status      VARCHAR(32)  NOT NULL DEFAULT 'pending',
    first_name  VARCHAR(255),
    last_name   VARCHAR(255),
    username    VARCHAR(255),
    tg_user_id  BIGINT,
    worker_id   VARCHAR(128),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE bots (
    id          BIGSERIAL PRIMARY KEY,
    token       VARCHAR(128) NOT NULL UNIQUE,
    session_id  VARCHAR(128) NOT NULL UNIQUE,
    status      VARCHAR(32)  NOT NULL DEFAULT 'pending',
    username    VARCHAR(255),
    tg_user_id  BIGINT,
    worker_id   VARCHAR(128),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounts_status    ON accounts(status);
CREATE INDEX idx_accounts_worker_id ON accounts(worker_id);
CREATE INDEX idx_bots_status        ON bots(status);
CREATE INDEX idx_bots_worker_id     ON bots(worker_id);
