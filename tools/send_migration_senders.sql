-- SQL to create the `senders` table for quick local/CI application
-- Intended for temporary use by maintainers to allow e2e tests to run
-- Run with: psql -U <user> -d <db> -f send_migration_senders.sql

BEGIN;

CREATE TABLE IF NOT EXISTS senders (
  id SERIAL PRIMARY KEY,
  email VARCHAR(255) NOT NULL,
  name VARCHAR(255),
  verified BOOLEAN DEFAULT false,
  verification_code VARCHAR(128),
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_senders_email_lower ON senders (LOWER(email));

COMMIT;
