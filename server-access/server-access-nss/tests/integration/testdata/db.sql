CREATE TABLE groups
(
    name TEXT PRIMARY KEY,
    gid  INTEGER NOT NULL
);

CREATE TABLE users
(
    name        TEXT PRIMARY KEY NOT NULL,
    uid         INTEGER          NOT NULL,
    gid         INTEGER          NOT NULL,
    gecos       TEXT,
    homedir     TEXT             NOT NULL,
    shell       TEXT             NOT NULL,
    hashed_pass TEXT             NOT NULL
);

CREATE INDEX IF NOT EXISTS "users_uid" ON users(uid);