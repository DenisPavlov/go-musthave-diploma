-- +goose Up
CREATE TABLE IF NOT EXISTS orders
(
    number      TEXT PRIMARY KEY,
    username    TEXT      NOT NULL,
    status      TEXT      NOT NULL,
    uploaded_at TIMESTAMP NOT NULL,
    accrual     DECIMAL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS users
(
    username      TEXT PRIMARY KEY,
    password_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS withdrawal
(
    order_number TEXT PRIMARY KEY,
    sum          DECIMAL   NOT NULL DEFAULT 0,
    username     TEXT      NOT NULL REFERENCES users (username),
    processed_at TIMESTAMP NOT NULL DEFAULT now()
);

-- +goose Down
drop table withdrawal;
drop table users;
drop table orders