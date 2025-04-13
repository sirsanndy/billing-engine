-- +migrate Up

-- Create the customers table
CREATE TABLE customers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    address TEXT NOT NULL,
    is_delinquent BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    loan_id BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure loan_id, if set, refers to a valid loan
    CONSTRAINT fk_customers_loan
        FOREIGN KEY (loan_id)
        REFERENCES loans(id)
        ON DELETE SET NULL, -- If a loan is deleted, set customer.loan_id to NULL

    -- Ensure that a loan_id can only be assigned to one customer (allows multiple NULLs)
    CONSTRAINT uq_customers_loan_id UNIQUE (loan_id)
);

-- Indexes for common lookups and filtering
CREATE INDEX IF NOT EXISTS idx_customers_loan_id ON customers (loan_id);
CREATE INDEX IF NOT EXISTS idx_customers_active ON customers (active);
CREATE INDEX IF NOT EXISTS idx_customers_is_delinquent ON customers (is_delinquent);
CREATE INDEX IF NOT EXISTS idx_customers_name ON customers (name);

-- Create the trigger for the customers table
CREATE TRIGGER set_timestamp_customers
BEFORE UPDATE ON customers
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();


-- +migrate Down

-- Drop objects in reverse order of creation

DROP TRIGGER IF EXISTS set_timestamp_customers ON customers;

DROP INDEX IF EXISTS idx_customers_name;
DROP INDEX IF EXISTS idx_customers_is_delinquent;
DROP INDEX IF EXISTS idx_customers_active;
DROP INDEX IF EXISTS idx_customers_loan_id;
DROP TABLE IF EXISTS customers;