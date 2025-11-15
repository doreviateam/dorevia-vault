package crypto

import (
	"context"
)

// Signature représente une signature JWS avec son KID
type Signature struct {
	JWS string // JWS compact
	KID string // Key ID
}

// Signer est une interface pour signer des payloads (abstraction HSM-ready)
// Sprint 6 - Phase 2 : Interface pour permettre l'implémentation future d'un HsmSigner
type Signer interface {
	// SignPayload signe un payload Evidence et retourne une Signature
	// Le payload doit être un JSON marshallé contenant {document_id, sha256, timestamp}
	SignPayload(ctx context.Context, payload []byte) (*Signature, error)

	// KeyID retourne l'identifiant de la clé actuelle
	KeyID() string
}

// EvidencePayload représente le payload à signer pour un document
type EvidencePayload struct {
	DocumentID string `json:"document_id"`
	Sha256     string `json:"sha256"`
	Timestamp  string `json:"timestamp"` // Format RFC3339
}

