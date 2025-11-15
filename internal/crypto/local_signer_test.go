package crypto

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalSigner_SignPayload(t *testing.T) {
	// Créer un service JWS de test (nécessite des clés)
	// Pour les tests, on peut utiliser des clés de test ou mocker
	// Pour l'instant, on teste la structure de base

	// Note: Ce test nécessite un service JWS fonctionnel
	// On peut créer un service avec des clés de test ou utiliser un mock
	// Pour l'instant, on teste juste que l'interface fonctionne

	t.Run("SignPayload avec payload valide", func(t *testing.T) {
		// Ce test nécessite un service JWS réel
		// On peut le marquer comme test d'intégration
		t.Skip("Requires JWS service with keys - integration test")
	})

	t.Run("SignPayload avec payload invalide", func(t *testing.T) {
		// Créer un signer avec un service nil (pour tester l'erreur de parsing)
		signer := &LocalSigner{service: nil}

		invalidPayload := []byte(`{"invalid": "payload"}`)
		_, err := signer.SignPayload(context.Background(), invalidPayload)

		// On s'attend à une erreur car le service est nil
		// Mais d'abord on teste le parsing
		assert.Error(t, err)
	})

	t.Run("SignPayload avec JSON invalide", func(t *testing.T) {
		signer := &LocalSigner{service: nil}

		invalidJSON := []byte(`{invalid json}`)
		_, err := signer.SignPayload(context.Background(), invalidJSON)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal evidence payload")
	})

	t.Run("SignPayload avec timestamp invalide", func(t *testing.T) {
		signer := &LocalSigner{service: nil}

		payload := EvidencePayload{
			DocumentID: "test-id",
			Sha256:     "test-hash",
			Timestamp:  "invalid-timestamp",
		}
		payloadBytes, _ := json.Marshal(payload)

		_, err := signer.SignPayload(context.Background(), payloadBytes)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse timestamp")
	})
}

func TestLocalSigner_KeyID(t *testing.T) {
	t.Run("KeyID retourne le KID du service", func(t *testing.T) {
		// Ce test nécessite un service JWS réel
		t.Skip("Requires JWS service - integration test")
	})
}

func TestLocalSigner_IntegrationWithJWSService(t *testing.T) {
	// Test d'intégration complet nécessitant un service JWS avec clés
	// À implémenter dans les tests d'intégration
	t.Skip("Integration test - requires JWS service with keys")
}

// TestEvidencePayload_Marshal teste la sérialisation du payload Evidence
func TestEvidencePayload_Marshal(t *testing.T) {
	payload := EvidencePayload{
		DocumentID: "test-doc-id",
		Sha256:     "test-sha256",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	bytes, err := json.Marshal(payload)
	require.NoError(t, err)

	var unmarshaled EvidencePayload
	err = json.Unmarshal(bytes, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, payload.DocumentID, unmarshaled.DocumentID)
	assert.Equal(t, payload.Sha256, unmarshaled.Sha256)
	assert.Equal(t, payload.Timestamp, unmarshaled.Timestamp)
}

