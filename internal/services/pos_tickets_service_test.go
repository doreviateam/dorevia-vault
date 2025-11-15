package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/ledger"
	"github.com/doreviateam/dorevia-vault/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDocumentRepository est un mock pour DocumentRepository
type MockDocumentRepository struct {
	mock.Mock
}

func (m *MockDocumentRepository) GetDocumentBySHA256(ctx context.Context, sha256 string) (*models.Document, error) {
	args := m.Called(ctx, sha256)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Document), args.Error(1)
}

func (m *MockDocumentRepository) InsertDocumentWithEvidence(
	ctx context.Context,
	doc *models.Document,
	evidenceJWS string,
	ledgerService ledger.Service,
) error {
	args := m.Called(ctx, doc, evidenceJWS, ledgerService)
	return args.Error(0)
}

// MockLedgerService est un mock pour ledger.Service
type MockLedgerService struct {
	mock.Mock
}

func (m *MockLedgerService) Append(ctx context.Context, tx pgx.Tx, docID uuid.UUID, shaHex, jws string) (string, error) {
	args := m.Called(ctx, tx, docID, shaHex, jws)
	return args.String(0), args.Error(1)
}

func (m *MockLedgerService) ExistsByDocumentID(ctx context.Context, tx pgx.Tx, docID uuid.UUID) (bool, error) {
	args := m.Called(ctx, tx, docID)
	return args.Bool(0), args.Error(1)
}

// MockSigner est un mock pour crypto.Signer
type MockSigner struct {
	mock.Mock
}

func (m *MockSigner) SignPayload(ctx context.Context, payload []byte) (*crypto.Signature, error) {
	args := m.Called(ctx, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*crypto.Signature), args.Error(1)
}

func (m *MockSigner) KeyID() string {
	args := m.Called()
	return args.String(0)
}

func TestPosTicketsService_Ingest_Success(t *testing.T) {
	repo := new(MockDocumentRepository)
	ledgerSvc := new(MockLedgerService)
	signer := new(MockSigner)

	service := NewPosTicketsService(repo, ledgerSvc, signer)

	input := PosTicketInput{
		Tenant:       "test-tenant",
		SourceSystem: "odoo_pos",
		SourceModel:  "pos.order",
		SourceID:     "POS/001",
		Currency:     stringPtr("EUR"),
		TotalInclTax: floatPtr(12.50),
		TotalExclTax: floatPtr(10.42),
		PosSession:   stringPtr("SESSION/001"),
		Cashier:      stringPtr("Test Cashier"),
		Location:     stringPtr("Test Location"),
		Ticket: map[string]interface{}{
			"lines": []interface{}{
				map[string]interface{}{"product": "Item 1", "quantity": 1},
			},
		},
	}

	ctx := context.Background()

	// Mock : document n'existe pas encore
	repo.On("GetDocumentBySHA256", ctx, mock.AnythingOfType("string")).Return(nil, nil)

	// Mock : signature réussie
	signature := &crypto.Signature{
		JWS: "test-jws-token",
		KID: "test-kid",
	}
	signer.On("SignPayload", ctx, mock.AnythingOfType("[]uint8")).Return(signature, nil)

	// Mock : insertion réussie
	repo.On("InsertDocumentWithEvidence", ctx, mock.AnythingOfType("*models.Document"), "test-jws-token", ledgerSvc).Return(nil)

	result, err := service.Ingest(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, input.Tenant, result.Tenant)
	assert.NotEmpty(t, result.SHA256Hex)
	assert.NotNil(t, result.EvidenceJWS)
	assert.Equal(t, "test-jws-token", *result.EvidenceJWS)

	repo.AssertExpectations(t)
	signer.AssertExpectations(t)
}

func TestPosTicketsService_Ingest_Idempotence(t *testing.T) {
	repo := new(MockDocumentRepository)
	ledgerSvc := new(MockLedgerService)
	signer := new(MockSigner)

	service := NewPosTicketsService(repo, ledgerSvc, signer)

	input := PosTicketInput{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/001",
		PosSession:  stringPtr("SESSION/001"),
		Ticket: map[string]interface{}{
			"lines": []interface{}{},
		},
	}

	ctx := context.Background()

	// Mock : document existe déjà
	existingDoc := &models.Document{
		ID:        uuid.New(),
		SHA256Hex: "existing-hash",
		CreatedAt: time.Now(),
	}
	existingDoc.EvidenceJWS = stringPtr("existing-jws")
	existingDoc.LedgerHash = stringPtr("existing-ledger-hash")

	repo.On("GetDocumentBySHA256", ctx, mock.AnythingOfType("string")).Return(existingDoc, nil)

	result, err := service.Ingest(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, existingDoc.ID, result.ID)
	assert.Equal(t, existingDoc.SHA256Hex, result.SHA256Hex)
	assert.Equal(t, existingDoc.EvidenceJWS, result.EvidenceJWS)
	assert.Equal(t, existingDoc.LedgerHash, result.LedgerHash)

	// Vérifier que l'insertion n'a pas été appelée
	repo.AssertNotCalled(t, "InsertDocumentWithEvidence")
}

func TestPosTicketsService_Ingest_Idempotence_MetadataChange(t *testing.T) {
	repo := new(MockDocumentRepository)
	ledgerSvc := new(MockLedgerService)
	signer := new(MockSigner)

	service := NewPosTicketsService(repo, ledgerSvc, signer)

	// Premier appel
	input1 := PosTicketInput{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/001",
		PosSession:  stringPtr("SESSION/001"),
		Cashier:     stringPtr("Cashier 1"), // Métadonnée
		Ticket: map[string]interface{}{
			"lines": []interface{}{},
		},
	}

	ctx := context.Background()

	// Mock : document n'existe pas
	repo.On("GetDocumentBySHA256", ctx, mock.AnythingOfType("string")).Return(nil, nil).Once()
	signature := &crypto.Signature{JWS: "jws-1", KID: "kid-1"}
	signer.On("SignPayload", ctx, mock.AnythingOfType("[]uint8")).Return(signature, nil).Once()
	repo.On("InsertDocumentWithEvidence", ctx, mock.AnythingOfType("*models.Document"), "jws-1", ledgerSvc).Return(nil).Once()

	result1, err1 := service.Ingest(ctx, input1)
	require.NoError(t, err1)
	hash1 := result1.SHA256Hex

	// Deuxième appel avec métadonnée différente (cashier changé)
	input2 := input1
	input2.Cashier = stringPtr("Cashier 2") // Métadonnée changée

	// Mock : document existe déjà (même hash car ticket + source_id + session identiques)
	existingDoc := &models.Document{
		ID:        result1.ID,
		SHA256Hex: hash1,
		CreatedAt: result1.CreatedAt,
	}
	existingDoc.EvidenceJWS = result1.EvidenceJWS
	existingDoc.LedgerHash = result1.LedgerHash

	repo.On("GetDocumentBySHA256", ctx, hash1).Return(existingDoc, nil).Once()

	result2, err2 := service.Ingest(ctx, input2)
	require.NoError(t, err2)

	// Vérifier que c'est le même document (idempotent)
	assert.Equal(t, result1.ID, result2.ID)
	assert.Equal(t, result1.SHA256Hex, result2.SHA256Hex)

	repo.AssertExpectations(t)
}

func TestPosTicketsService_Ingest_LedgerError(t *testing.T) {
	repo := new(MockDocumentRepository)
	ledgerSvc := new(MockLedgerService)
	signer := new(MockSigner)

	service := NewPosTicketsService(repo, ledgerSvc, signer)

	input := PosTicketInput{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/001",
		PosSession:  stringPtr("SESSION/001"),
		Ticket:      map[string]interface{}{},
	}

	ctx := context.Background()

	repo.On("GetDocumentBySHA256", ctx, mock.AnythingOfType("string")).Return(nil, nil)
	signature := &crypto.Signature{JWS: "test-jws", KID: "test-kid"}
	signer.On("SignPayload", ctx, mock.AnythingOfType("[]uint8")).Return(signature, nil)

	// Mock : erreur lors de l'insertion (ledger error)
	repo.On("InsertDocumentWithEvidence", ctx, mock.AnythingOfType("*models.Document"), "test-jws", ledgerSvc).
		Return(errors.New("ledger error"))

	result, err := service.Ingest(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "insert document")
}

func TestPosTicketsService_Ingest_SignerError(t *testing.T) {
	repo := new(MockDocumentRepository)
	ledgerSvc := new(MockLedgerService)
	signer := new(MockSigner)

	service := NewPosTicketsService(repo, ledgerSvc, signer)

	input := PosTicketInput{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/001",
		PosSession:  stringPtr("SESSION/001"),
		Ticket:      map[string]interface{}{},
	}

	ctx := context.Background()

	repo.On("GetDocumentBySHA256", ctx, mock.AnythingOfType("string")).Return(nil, nil)

	// Mock : erreur lors de la signature
	signer.On("SignPayload", ctx, mock.AnythingOfType("[]uint8")).
		Return(nil, errors.New("signer error"))

	result, err := service.Ingest(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "sign evidence")
}

func TestPosTicketsService_Ingest_RepositoryError(t *testing.T) {
	repo := new(MockDocumentRepository)
	ledgerSvc := new(MockLedgerService)
	signer := new(MockSigner)

	service := NewPosTicketsService(repo, ledgerSvc, signer)

	input := PosTicketInput{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/001",
		PosSession:  stringPtr("SESSION/001"),
		Ticket:      map[string]interface{}{},
	}

	ctx := context.Background()

	// Mock : erreur lors de la vérification d'existence
	repo.On("GetDocumentBySHA256", ctx, mock.AnythingOfType("string")).
		Return(nil, errors.New("repository error"))

	result, err := service.Ingest(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check existing document")
}

func TestPosTicketsService_Canonicalization(t *testing.T) {
	repo := new(MockDocumentRepository)
	ledgerSvc := new(MockLedgerService)
	signer := new(MockSigner)

	service := NewPosTicketsService(repo, ledgerSvc, signer)

	// Test que deux JSON différents mais canoniquement identiques produisent le même hash
	input1 := PosTicketInput{
		Tenant:      "test",
		SourceModel: "pos.order",
		SourceID:    "POS/001",
		PosSession:  stringPtr("SESSION/001"),
		Ticket: map[string]interface{}{
			"b": 2.0,
			"a": 1,
			"c": nil,
		},
	}

	input2 := PosTicketInput{
		Tenant:      "test",
		SourceModel: "pos.order",
		SourceID:    "POS/001",
		PosSession:  stringPtr("SESSION/001"),
		Ticket: map[string]interface{}{
			"a": 1.0,
			"b": 2,
		},
	}

	ctx := context.Background()

	// Premier appel
	repo.On("GetDocumentBySHA256", ctx, mock.AnythingOfType("string")).Return(nil, nil).Once()
	signature := &crypto.Signature{JWS: "jws", KID: "kid"}
	signer.On("SignPayload", ctx, mock.AnythingOfType("[]uint8")).Return(signature, nil).Once()
	repo.On("InsertDocumentWithEvidence", ctx, mock.AnythingOfType("*models.Document"), "jws", ledgerSvc).Return(nil).Once()

	result1, err1 := service.Ingest(ctx, input1)
	require.NoError(t, err1)
	hash1 := result1.SHA256Hex

	// Deuxième appel (JSON différent mais canoniquement identique)
	existingDoc := &models.Document{
		ID:        result1.ID,
		SHA256Hex: hash1,
		CreatedAt: result1.CreatedAt,
	}
	existingDoc.EvidenceJWS = result1.EvidenceJWS
	existingDoc.LedgerHash = result1.LedgerHash

	repo.On("GetDocumentBySHA256", ctx, hash1).Return(existingDoc, nil).Once()

	result2, err2 := service.Ingest(ctx, input2)
	require.NoError(t, err2)

	// Vérifier que le hash est identique (canonicalisation)
	assert.Equal(t, hash1, result2.SHA256Hex)
}

// Helpers
func stringPtr(s string) *string {
	return &s
}

func floatPtr(f float64) *float64 {
	return &f
}

