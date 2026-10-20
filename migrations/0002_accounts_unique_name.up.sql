-- RF-01/RF-02: no se permiten dos cuentas con el mismo nombre (alta/edición).
CREATE UNIQUE INDEX idx_accounts_name ON accounts (name);
