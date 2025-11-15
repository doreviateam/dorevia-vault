package models

import (
	"time"

	"github.com/google/uuid"
)

// Document représente un document stocké dans le système
type Document struct {
	// Champs existants
	ID          uuid.UUID `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	SHA256Hex   string    `json:"sha256_hex"`
	StoredPath  string    `json:"stored_path"`
	CreatedAt   time.Time `json:"created_at"`

	// Métadonnées Odoo (Sprint 1)
	Source    *string `json:"source,omitempty"`     // sales|purchase|pos|stock|sale
	OdooModel *string `json:"odoo_model,omitempty"` // account.move, pos.order, etc.
	OdooID    *int    `json:"odoo_id,omitempty"`    // ID dans Odoo
	OdooState *string `json:"odoo_state,omitempty"` // posted, paid, done, etc.

	// Routage PDP (préparation Sprint 2)
	PDPRequired   *bool   `json:"pdp_required,omitempty"`   // Nécessite dispatch PDP ?
	DispatchStatus *string `json:"dispatch_status,omitempty"` // PENDING|SENT|ACK|REJECTED

	// Métadonnées facture (préparation Sprint 2 - validation Factur-X)
	InvoiceNumber *string    `json:"invoice_number,omitempty"`
	InvoiceDate   *time.Time `json:"invoice_date,omitempty"`
	TotalHT       *float64   `json:"total_ht,omitempty"`
	TotalTTC      *float64   `json:"total_ttc,omitempty"`
	Currency      *string    `json:"currency,omitempty"`
	SellerVAT     *string    `json:"seller_vat,omitempty"`
	BuyerVAT      *string    `json:"buyer_vat,omitempty"`

	// Preuves d'intégrité (Sprint 2)
	EvidenceJWS *string `json:"evidence_jws,omitempty"` // Jeton JWS signé
	LedgerHash  *string `json:"ledger_hash,omitempty"`  // Hash dans le ledger

	// Champs POS (Sprint 6) - optionnels, NULL pour documents non-POS
	SourceIDText *string                `json:"source_id_text,omitempty" db:"source_id_text"` // ID textuel (pour POS)
	PayloadJSON  []byte                 `json:"payload_json,omitempty" db:"payload_json"`     // JSON brut (pour POS)
	PosSession   *string                `json:"pos_session,omitempty" db:"pos_session"`
	Cashier      *string                `json:"cashier,omitempty" db:"cashier"`
	Location     *string                `json:"location,omitempty" db:"location"`
}

// DocumentListResponse représente la réponse pour la liste de documents
type DocumentListResponse struct {
	Data       []Document        `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginationResponse contient les informations de pagination
type PaginationResponse struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
	Pages int `json:"pages"`
}

// DocumentQuery représente les paramètres de requête pour la recherche
type DocumentQuery struct {
	Page      int
	Limit     int
	Search    string
	Type      string
	DateFrom  *time.Time
	DateTo    *time.Time
}

