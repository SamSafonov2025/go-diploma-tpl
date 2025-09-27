-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
id uuid UNIQUE NOT NULL PRIMARY KEY,
login text UNIQUE NOT NULL,
password text NOT NULL,
created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS balances (
uid uuid UNIQUE NOT NULL PRIMARY KEY,
current_balance float NOT NULL DEFAULT 0,
withdrawn float NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS orders (
id text UNIQUE NOT NULL PRIMARY KEY,
uid uuid NOT NULL,
accrual float,
accrual_status text NOT NULL,
uploaded_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS withdrawals (
order_id text UNIQUE NOT NULL PRIMARY KEY,
uid uuid NOT NULL,
amount float NOT NULL,
processed_at timestamptz NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP DATABASE users;
DROP DATABASE balances;
DROP DATABASE orders;
DROP DATABASE withdrawals;
-- +goose StatementEnd
