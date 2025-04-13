-- migrations/001_create_customer_table.sql
CREATE TABLE customers (
    id BIGINT PRIMARY KEY, -- Use BIGINT if ID comes from source, BIGSERIAL if generated here
    name VARCHAR(255) NOT NULL,
    address TEXT NOT NULL,
    is_delinquent BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    loan_id BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL, -- Store time provided by event
    updated_at TIMESTAMPTZ NOT NULL, -- Store time provided by event
    CONSTRAINT uq_customers_loan_id UNIQUE (loan_id) -- Optional: Only if LoanID must be unique in notify-db too
);

-- Add index on updated_at for the WHERE clause in ON CONFLICT UPDATE
CREATE INDEX idx_customers_updated_at ON customers (updated_at);