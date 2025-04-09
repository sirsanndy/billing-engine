-- +migrate Up
CREATE TABLE loans (
    id BIGSERIAL PRIMARY KEY,
    principal_amount DECIMAL(15, 2) NOT NULL CHECK (principal_amount > 0),
    interest_rate DECIMAL(5, 4) NOT NULL CHECK (interest_rate >= 0), -- e.g., 0.10 for 10%
    term_weeks INT NOT NULL CHECK (term_weeks > 0),
    weekly_payment_amount DECIMAL(15, 2) NOT NULL CHECK (weekly_payment_amount >= 0),
    total_loan_amount DECIMAL(15, 2) NOT NULL CHECK (total_loan_amount >= principal_amount),
    start_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'PAID_OFF', 'DELINQUENT')), -- Delinquent status might be better tracked dynamically or separately
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


-- +migrate Down
DROP TRIGGER IF EXISTS set_timestamp_loans ON loans;
DROP FUNCTION IF EXISTS trigger_set_timestamp();
DROP TABLE IF EXISTS loans;