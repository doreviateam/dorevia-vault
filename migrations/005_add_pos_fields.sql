-- Migration 005: Ajout des champs POS (Sprint 6)
-- Date: 2025-01-14
-- Description: Ajoute les champs spécifiques aux tickets POS à la table documents

-- Champ pour stocker le JSON brut du ticket POS (pour tickets POS uniquement)
ALTER TABLE documents ADD COLUMN IF NOT EXISTS payload_json JSONB;

-- Champ pour source_id textuel (pour POS avec IDs string comme "POS/2025/0001")
ALTER TABLE documents ADD COLUMN IF NOT EXISTS source_id_text TEXT;

-- Champs métier POS (optionnels, NULL pour les documents non-POS)
ALTER TABLE documents ADD COLUMN IF NOT EXISTS pos_session TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS cashier TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS location TEXT;

-- Index pour recherche rapide sur payload_json (GIN index pour JSONB)
CREATE INDEX IF NOT EXISTS idx_documents_payload_json ON documents USING GIN (payload_json);

-- Index pour recherche POS
CREATE INDEX IF NOT EXISTS idx_documents_source_id_text ON documents(source_id_text) WHERE source = 'pos';
CREATE INDEX IF NOT EXISTS idx_documents_pos_session ON documents(pos_session) WHERE source = 'pos';
CREATE INDEX IF NOT EXISTS idx_documents_cashier ON documents(cashier) WHERE source = 'pos';
CREATE INDEX IF NOT EXISTS idx_documents_location ON documents(location) WHERE source = 'pos';

-- Index composite pour recherche par source + odoo_model (optimisation POS)
CREATE INDEX IF NOT EXISTS idx_documents_source_model ON documents(source, odoo_model) 
    WHERE source = 'pos' AND odoo_model = 'pos.order';

