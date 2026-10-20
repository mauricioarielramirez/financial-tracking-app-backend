-- RF-01/RF-02: two accounts can't have the same name (create/edit).
CREATE UNIQUE INDEX idx_accounts_name ON accounts (name);
