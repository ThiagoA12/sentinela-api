CREATE TABLE users (
    id            uuid         PRIMARY KEY DEFAULT uuidv7(),
    name          varchar(100) NOT NULL,
    email         varchar(254) NOT NULL,
    password_hash text         NOT NULL,
    role          text         NOT NULL
                  CHECK (role IN ('admin', 'analista', 'leitor')),
    active        boolean      NOT NULL DEFAULT true,
    failed_logins smallint     NOT NULL DEFAULT 0,
    locked_until  timestamptz,
    created_at    timestamptz   NOT NULL DEFAULT now(),
    updated_at    timestamptz   NOT NULL DEFAULT now(),
    version       integer      NOT NULL DEFAULT 1
);

CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));

CREATE TABLE audit_log (
    id         bigint     PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    actor_id   uuid       REFERENCES users (id),
    action     text       NOT NULL
               CHECK (action IN ('create', 'update', 'delete', 'login', 'login_failed')),
    entity     text       NOT NULL
               CHECK (entity IN ('users', 'assets')),
    entity_id  uuid,
    request_id text       NOT NULL,
    changes    jsonb
);

CREATE INDEX audit_log_entity_idx ON audit_log (entity, entity_id);
CREATE INDEX audit_log_actor_idx ON audit_log (actor_id, occurred_at);