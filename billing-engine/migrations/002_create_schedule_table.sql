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


-- +migrate Down
DROP TRIGGER IF EXISTS set_timestamp_loan_schedule ON loan_schedule;
DROP TABLE IF EXISTS loan_schedule;
DROP TYPE IF EXISTS payment_status;
-- Keep the trigger function trigger_set_timestamp() if other tables use it, or drop it in the last migration using it.