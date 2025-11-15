package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/ledger"
	"github.com/doreviateam/dorevia-vault/internal/models"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/doreviateam/dorevia-vault/internal/utils"
	"github.com/google/uuid"
)

// PosTicketsServiceInterface définit les opérations du service POS
type PosTicketsServiceInterface interface {
	Ingest(ctx context.Context, input PosTicketInput) (*PosTicketResult, error)
}

// PosTicketsService gère l'ingestion des tickets POS
// Sprint 6 - Phase 3 : Service métier avec interfaces abstraites
type PosTicketsService struct {
	repo   storage.DocumentRepository // Interface, pas *storage.DB
	ledger ledger.Service              // Interface
	signer crypto.Signer
}

// Vérifier que PosTicketsService implémente PosTicketsServiceInterface
var _ PosTicketsServiceInterface = (*PosTicketsService)(nil)

// NewPosTicketsService crée un nouveau service POS
func NewPosTicketsService(
	repo storage.DocumentRepository,
	ledger ledger.Service,
	signer crypto.Signer,
) *PosTicketsService {
	return &PosTicketsService{
		repo:   repo,
		ledger: ledger,
		signer: signer,
	}
}

// PosTicketResult représente le résultat de l'ingestion d'un ticket POS
type PosTicketResult struct {
	ID          uuid.UUID
	Tenant      string
	SHA256Hex   string
	LedgerHash  *string
	EvidenceJWS *string
	CreatedAt   time.Time
}

// Ingest ingère un ticket POS avec idempotence métier stricte
// Hash basé sur ticket + source_id + pos_session (Option A)
func (s *PosTicketsService) Ingest(ctx context.Context, input PosTicketInput) (*PosTicketResult, error) {
	// 1. Construire le hash input pour idempotence métier stricte (Option A)
	// Hash basé sur ticket + source_id + pos_session (plus stable)
	hashInput := map[string]interface{}{
		"ticket":      input.Ticket,
		"source_id":   input.SourceID,
		"pos_session": input.PosSession,
	}

	// 2. Marshal et canonicaliser le hash input
	hashInputBytes, err := json.Marshal(hashInput)
	if err != nil {
		return nil, fmt.Errorf("marshal hash input: %w", err)
	}

	canonicalBytes, err := utils.CanonicalizeJSON(hashInputBytes)
	if err != nil {
		return nil, fmt.Errorf("canonicalize JSON: %w", err)
	}

	// 3. Calculer SHA256 pour idempotence
	hash := sha256.Sum256(canonicalBytes)
	sha256Hex := hex.EncodeToString(hash[:])

	// 4. Vérifier idempotence (par sha256)
	existingDoc, err := s.repo.GetDocumentBySHA256(ctx, sha256Hex)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing document: %w", err)
	}
	if existingDoc != nil {
		// Document déjà existant (idempotence)
		return &PosTicketResult{
			ID:          existingDoc.ID,
			Tenant:      input.Tenant,
			SHA256Hex:   existingDoc.SHA256Hex,
			LedgerHash:  existingDoc.LedgerHash,
			EvidenceJWS: existingDoc.EvidenceJWS,
			CreatedAt:   existingDoc.CreatedAt,
		}, nil
	}

	// 5. Marshal le payload complet pour stockage
	fullPayload := map[string]interface{}{
		"tenant":        input.Tenant,
		"source_system": input.SourceSystem,
		"source_model":  input.SourceModel,
		"source_id":     input.SourceID,
		"currency":      input.Currency,
		"total_incl_tax": input.TotalInclTax,
		"total_excl_tax": input.TotalExclTax,
		"pos_session":   input.PosSession,
		"cashier":       input.Cashier,
		"location":      input.Location,
		"ticket":        input.Ticket,
	}
	fullPayloadBytes, err := json.Marshal(fullPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal full payload: %w", err)
	}
	fullCanonicalBytes, err := utils.CanonicalizeJSON(fullPayloadBytes)
	if err != nil {
		return nil, fmt.Errorf("canonicalize full payload: %w", err)
	}

	// 6. Créer le document
	docID := uuid.New()
	source := "pos"
	now := time.Now()

	doc := &models.Document{
		ID:          docID,
		Filename:    fmt.Sprintf("pos-ticket-%s.json", input.SourceID),
		ContentType: "application/json",
		SizeBytes:   int64(len(fullCanonicalBytes)),
		SHA256Hex:   sha256Hex, // Hash pour idempotence (basé sur ticket + source_id + session)
		StoredPath:  "",        // Pas de fichier, stockage en DB uniquement
		CreatedAt:   now,
		Source:      &source,
		OdooModel:   &input.SourceModel,
		SourceIDText: &input.SourceID, // Stocker l'ID textuel
		// OdooID reste NULL pour les tickets POS (on utilise source_id_text)
		PayloadJSON: fullCanonicalBytes, // JSON complet canonicalisé pour stockage
		Currency:    input.Currency,
		TotalHT:     input.TotalExclTax,
		TotalTTC:    input.TotalInclTax,
		PosSession:  input.PosSession,
		Cashier:     input.Cashier,
		Location:    input.Location,
	}

	// 7. Construire le payload Evidence et signer
	evidencePayload := crypto.EvidencePayload{
		DocumentID: docID.String(),
		Sha256:     sha256Hex,
		Timestamp:  now.UTC().Format(time.RFC3339),
	}
	evidenceBytes, err := json.Marshal(evidencePayload)
	if err != nil {
		return nil, fmt.Errorf("marshal evidence payload: %w", err)
	}

	signature, err := s.signer.SignPayload(ctx, evidenceBytes)
	if err != nil {
		return nil, fmt.Errorf("sign evidence: %w", err)
	}
	evidenceJWS := signature.JWS

	// 8. Insérer le document avec evidence via le repository
	// Le repository gère la transaction, l'insertion, l'ajout au ledger et la mise à jour
	err = s.repo.InsertDocumentWithEvidence(ctx, doc, evidenceJWS, s.ledger)
	if err != nil {
		return nil, fmt.Errorf("insert document: %w", err)
	}

	// 9. Récupérer le ledger_hash depuis le document (mis à jour par le repository)
	ledgerHash := ""
	if doc.LedgerHash != nil {
		ledgerHash = *doc.LedgerHash
	}

	return &PosTicketResult{
		ID:          docID,
		Tenant:      input.Tenant,
		SHA256Hex:   sha256Hex,
		LedgerHash:  &ledgerHash,
		EvidenceJWS: &evidenceJWS,
		CreatedAt:   now,
	}, nil
}

