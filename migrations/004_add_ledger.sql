-- Migration 004: Table ledger hash-chaîné (Sprint 2)
-- Date: Janvier 2025

-- Table ledger
CREATE TABLE IF NOT EXISTS ledger (
  id SERIAL PRIMARY KEY,
  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  hash TEXT NOT NULL,
  previous_hash TEXT,
  timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
  evidence_jws TEXT
);

-- Index pour recherche rapide
CREATE INDEX IF NOT EXISTS idx_ledger_document_id ON ledger(document_id);
CREATE INDEX IF NOT EXISTS idx_ledger_timestamp ON ledger(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_hash ON ledger(hash);
CREATE INDEX IF NOT EXISTS idx_ledger_prev_hash ON ledger(previous_hash);

-- Index composite pour SELECT previous_hash optimisé (verrou FOR UPDATE)
CREATE INDEX IF NOT EXISTS idx_ledger_ts_id_desc ON ledger(timestamp DESC, id DESC);

-- Contrainte d'unicité (document_id, hash) pour éviter les doublons
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'uq_ledger_doc_hash'
    ) THEN
        ALTER TABLE ledger ADD CONSTRAINT uq_ledger_doc_hash 
        UNIQUE (document_id, hash);
    END IF;
END $$;

