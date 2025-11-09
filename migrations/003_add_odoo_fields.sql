-- Migration 003: Ajout des champs Odoo et métadonnées (Sprint 1)
-- Date: Janvier 2025

-- Métadonnées Odoo
ALTER TABLE documents ADD COLUMN IF NOT EXISTS source TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_model TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_id INTEGER;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_state TEXT;

-- Routage PDP (préparation Sprint 2)
ALTER TABLE documents ADD COLUMN IF NOT EXISTS pdp_required BOOLEAN DEFAULT false;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS dispatch_status TEXT DEFAULT 'PENDING';

-- Métadonnées facture (préparation Sprint 2 - validation Factur-X)
ALTER TABLE documents ADD COLUMN IF NOT EXISTS invoice_number TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS invoice_date DATE;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS total_ht DECIMAL(12,2);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS total_ttc DECIMAL(12,2);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS currency TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS seller_vat TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS buyer_vat TEXT;

-- Index pour recherche rapide
CREATE INDEX IF NOT EXISTS idx_documents_odoo_id ON documents(odoo_id);
CREATE INDEX IF NOT EXISTS idx_documents_dispatch_status ON documents(dispatch_status);
CREATE INDEX IF NOT EXISTS idx_documents_source ON documents(source);

-- Contrainte sur dispatch_status
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'chk_dispatch_status'
    ) THEN
        ALTER TABLE documents ADD CONSTRAINT chk_dispatch_status 
        CHECK (dispatch_status IN ('PENDING', 'SENT', 'ACK', 'REJECTED'));
    END IF;
END $$;

-- Contrainte sur source
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'chk_source'
    ) THEN
        ALTER TABLE documents ADD CONSTRAINT chk_source
        CHECK (source IN ('sales','purchase','pos','stock','sale') OR source IS NULL);
    END IF;
END $$;

-- Preuves d'intégrité (Sprint 2)
ALTER TABLE documents ADD COLUMN IF NOT EXISTS evidence_jws TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS ledger_hash TEXT;

