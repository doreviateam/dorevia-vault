package crypto

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// LocalSigner implémente Signer en utilisant crypto.Service (implémentation locale)
// Sprint 6 - Phase 2 : Adaptateur depuis le service JWS existant
type LocalSigner struct {
	service *Service
}

// NewLocalSigner crée un LocalSigner depuis un Service existant
func NewLocalSigner(service *Service) *LocalSigner {
	return &LocalSigner{service: service}
}

// SignPayload signe un payload Evidence
func (s *LocalSigner) SignPayload(ctx context.Context, payload []byte) (*Signature, error) {
	// Parser le payload pour extraire document_id, sha256, timestamp
	var evidence EvidencePayload
	if err := json.Unmarshal(payload, &evidence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal evidence payload: %w", err)
	}

	// Parser le timestamp
	timestamp, err := time.Parse(time.RFC3339, evidence.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Utiliser le service JWS existant
	jws, err := s.service.SignEvidence(evidence.DocumentID, evidence.Sha256, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to sign evidence: %w", err)
	}

	return &Signature{
		JWS: jws,
		KID: s.service.GetKID(),
	}, nil
}

// KeyID retourne le KID actuel
func (s *LocalSigner) KeyID() string {
	return s.service.GetKID()
}

