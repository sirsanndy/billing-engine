-- +migrate Up
CREATE TABLE loans (
    id BIGSERIAL PRIMARY KEY,
    principal_amount DECIMAL(15, 2) NOT NULL CHECK (principal_amount > 0),
    interest_rate DECIMAL(5, 4) NOT NULL CHECK (interest_rate >= 0),
    term_weeks INT NOT NULL CHECK (term_weeks > 0),
    weekly_payment_amount DECIMAL(15, 2) NOT NULL CHECK (weekly_payment_amount >= 0),
    total_loan_amount DECIMAL(15, 2) NOT NULL CHECK (total_loan_amount >= principal_amount),
    start_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'PAID_OFF', 'DELINQUENT')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_timestamp_loans
BEFORE UPDATE ON loans
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- +migrate Up
CREATE TYPE payment_status AS ENUM ('PENDING', 'PAID', 'MISSED');

CREATE TABLE loan_schedule (
    id BIGSERIAL PRIMARY KEY,
    loan_id BIGINT NOT NULL REFERENCES loans(id) ON DELETE CASCADE,
    week_number INT NOT NULL CHECK (week_number > 0),
    due_date DATE NOT NULL,
    due_amount DECIMAL(15, 2) NOT NULL CHECK (due_amount >= 0),
    paid_amount DECIMAL(15, 2) DEFAULT 0.00 CHECK (paid_amount >= 0),
    payment_date TIMESTAMPTZ NULL,
    status payment_status NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (loan_id, week_number) -- Ensure only one entry per week per loan
);

-- Indexes for performance
CREATE INDEX idx_loan_schedule_loan_id_status ON loan_schedule(loan_id, status);
CREATE INDEX idx_loan_schedule_due_date ON loan_schedule(due_date);
CREATE INDEX idx_loan_schedule_loan_id_due_date_status ON loan_schedule(loan_id, due_date, status);


-- Use the same timestamp update function
CREATE TRIGGER set_timestamp_loan_schedule
BEFORE UPDATE ON loan_schedule
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

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
