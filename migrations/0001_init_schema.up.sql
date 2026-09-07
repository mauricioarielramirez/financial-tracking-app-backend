-- Esquema inicial. Escrito en SQL portable (sin AUTOINCREMENT fuera de PK,
-- sin tipos dinámicos de SQLite) para poder migrar a Postgres/MySQL sin
-- reescribir el modelo (Plan Técnico sección 1 y 3).

CREATE TABLE accounts (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL,
    currency        TEXT NOT NULL,
    provider_group  TEXT,
    is_invested     INTEGER NOT NULL DEFAULT 0,
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE TABLE snapshots (
    id              TEXT PRIMARY KEY,
    snapshot_date   TEXT NOT NULL,
    notes           TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

-- RF-08: impedir más de un snapshot por mes salvo override explícito.
-- El override se resuelve en la capa de servicio (permite forzar el insert
-- eliminando primero el snapshot del mes si el usuario confirma); a nivel de
-- esquema se garantiza unicidad por fecha exacta.
CREATE UNIQUE INDEX idx_snapshots_date ON snapshots (snapshot_date);

CREATE TABLE exchange_rates (
    id              TEXT PRIMARY KEY,
    snapshot_id     TEXT NOT NULL REFERENCES snapshots (id) ON DELETE CASCADE,
    currency_code   TEXT NOT NULL,
    rate_to_ars     TEXT NOT NULL, -- decimal almacenado como texto exacto (ver internal/domain)
    source          TEXT NOT NULL,
    fetched_at      TEXT NOT NULL
);

CREATE INDEX idx_exchange_rates_snapshot ON exchange_rates (snapshot_id);

CREATE TABLE account_balances (
    id                    TEXT PRIMARY KEY,
    snapshot_id           TEXT NOT NULL REFERENCES snapshots (id) ON DELETE CASCADE,
    account_id            TEXT NOT NULL REFERENCES accounts (id),
    balance_original      TEXT NOT NULL,
    balance_ars           TEXT NOT NULL,
    percentage_of_total   TEXT NOT NULL,
    UNIQUE (snapshot_id, account_id)
);

CREATE INDEX idx_account_balances_account ON account_balances (account_id);

CREATE TABLE income_statements (
    id                      TEXT PRIMARY KEY,
    snapshot_id             TEXT REFERENCES snapshots (id),
    period_month            TEXT NOT NULL,
    income_total_ars        TEXT NOT NULL,
    expense_total_ars       TEXT NOT NULL,
    savings_ars             TEXT NOT NULL,
    savings_percentage      TEXT NOT NULL,
    cumulative_savings_ars  TEXT NOT NULL,
    income_total_usd        TEXT
);

CREATE UNIQUE INDEX idx_income_statements_period ON income_statements (period_month);
